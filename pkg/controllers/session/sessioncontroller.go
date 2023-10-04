package session

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	hfListers "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
)

const (
	SessionExpireTime = time.Hour * 3
)

type SessionController struct {
	hfClientSet hfClientset.Interface

	//ssWorkqueue workqueue.RateLimitingInterface
	ssWorkqueue workqueue.DelayingInterface

	vmLister  hfListers.VirtualMachineLister
	vmcLister hfListers.VirtualMachineClaimLister
	ssLister  hfListers.SessionLister

	vmSynced  cache.InformerSynced
	vmcSynced cache.InformerSynced
	ssSynced  cache.InformerSynced
	ctx       context.Context
}

func NewSessionController(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context) (*SessionController, error) {
	ssController := SessionController{}
	ssController.ctx = ctx
	ssController.hfClientSet = hfClientSet
	ssController.vmSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer().HasSynced
	ssController.vmcSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Informer().HasSynced
	ssController.ssSynced = hfInformerFactory.Hobbyfarm().V1().Sessions().Informer().HasSynced

	//ssController.ssWorkqueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Session")
	ssController.ssWorkqueue = workqueue.NewNamedDelayingQueue("ssc-ss")
	ssController.vmLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Lister()
	ssController.vmcLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Lister()
	ssController.ssLister = hfInformerFactory.Hobbyfarm().V1().Sessions().Lister()

	ssInformer := hfInformerFactory.Hobbyfarm().V1().Sessions().Informer()

	ssInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: ssController.enqueueSS,
		UpdateFunc: func(old, new interface{}) {
			ssController.enqueueSS(new)
		},
		DeleteFunc: ssController.enqueueSS,
	}, time.Minute*30)

	return &ssController, nil
}

func (s *SessionController) enqueueSS(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		//utilruntime.HandleError(err)
		return
	}
	glog.V(8).Infof("Enqueueing ss %s", key)
	//s.ssWorkqueue.AddRateLimited(key)
	s.ssWorkqueue.Add(key)
}

func (s *SessionController) Run(stopCh <-chan struct{}) error {
	defer s.ssWorkqueue.ShutDown()

	glog.V(4).Infof("Starting Session controller")
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, s.vmSynced, s.vmcSynced, s.ssSynced); !ok {
		return fmt.Errorf("failed to wait for vm, vmc, and ss caches to sync")
	}
	glog.Info("Starting ss controller workers")
	go wait.Until(s.runSSWorker, time.Second, stopCh)
	glog.Info("Started ss controller workers")
	//if ok := cache.WaitForCacheSync(stopCh, )
	<-stopCh
	return nil
}

func (s *SessionController) runSSWorker() {
	glog.V(6).Infof("Starting session worker")
	for s.processNextSession() {

	}
}

func (s *SessionController) processNextSession() bool {
	obj, shutdown := s.ssWorkqueue.Get()

	if shutdown {
		return false
	}

	err := func() error {
		defer s.ssWorkqueue.Done(obj)
		glog.V(8).Infof("processing ss in ss controller: %v", obj)
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string))
		if err != nil {
			glog.Errorf("error while splitting meta namespace key %v", err)
			return nil
		}

		err = s.reconcileSession(objName)

		if err != nil {
			glog.Error(err)
		}
		//s.ssWorkqueue.Forget(obj)
		glog.V(8).Infof("ss processed by session controller %v", objName)

		return nil

	}()

	if err != nil {
		return true
	}

	return true
}

func (s *SessionController) reconcileSession(ssName string) error {
	glog.V(4).Infof("reconciling session %s", ssName)

	ss, err := s.ssLister.Sessions(util.GetReleaseNamespace()).Get(ssName)

	if err != nil {
		return err
	}

	now := time.Now()

	expires, err := time.Parse(time.UnixDate, ss.Status.ExpirationTime)

	if err != nil {
		return err
	}

	timeUntilExpires := expires.Sub(now)

	// clean up sessions if they are finished
	if ss.Status.Finished {
		glog.V(6).Infof("deleted finished session  %s", ss.Name)

		// now that the vmclaims are deleted, go ahead and delete the session
		err = s.hfClientSet.HobbyfarmV1().Sessions(util.GetReleaseNamespace()).Delete(s.ctx, ss.Name, metav1.DeleteOptions{})

		if err != nil {
			return fmt.Errorf("error deleting session %s: %s", ss.Name, err)
		}

		glog.V(6).Infof("deleted old session %s", ss.Name)

		s.FinishProgress(ss.Name, ss.Spec.UserId)

		return nil
	}

	if expires.Before(now) && !ss.Status.Finished {
		// we need to set the session to finished and delete the vm's
		if ss.Status.Active && ss.Status.Paused && ss.Status.PausedTime != "" {
			pausedExpiration, err := time.Parse(time.UnixDate, ss.Status.PausedTime)
			if err != nil {
				glog.Error(err)
			}

			if pausedExpiration.After(now) {
				glog.V(4).Infof("Session %s was paused, and the pause expiration is after now, skipping clean up.", ss.Name)
				return nil
			}

			glog.V(4).Infof("Session %s was paused, but the pause expiration was before now, so cleaning up.", ss.Name)
		}
		for _, vmc := range ss.Spec.VmClaimSet {
			vmcObj, err := s.vmcLister.VirtualMachineClaims(util.GetReleaseNamespace()).Get(vmc)

			if err != nil {
				break
			}

			for _, vm := range vmcObj.Spec.VirtualMachines {
				if len(vm.VirtualMachineId) == 0 {
					// VM was not even provisioned / assigned yet.
					continue
				}
				taintErr := s.taintVM(vm.VirtualMachineId)
				if taintErr != nil {
					glog.Error(taintErr)
				}
			}

			taintErr := s.taintVMC(vmcObj.Name)
			if taintErr != nil {
				glog.Error(taintErr)
			}
		}

		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			result, getErr := s.hfClientSet.HobbyfarmV1().Sessions(util.GetReleaseNamespace()).Get(s.ctx, ssName, metav1.GetOptions{})
			if getErr != nil {
				return getErr
			}

			result.Status.Finished = true
			result.Status.Active = false

			result, updateErr := s.hfClientSet.HobbyfarmV1().Sessions(util.GetReleaseNamespace()).UpdateStatus(s.ctx, result, metav1.UpdateOptions{})
			if updateErr != nil {
				return updateErr
			}
			glog.V(4).Infof("updated result for ss")

			verifyErr := util.VerifySession(s.ssLister, result)

			if verifyErr != nil {
				return verifyErr
			}
			return nil
		})
		if retryErr != nil {
			return retryErr
		}
	} else if expires.Before(now) && ss.Status.Finished {
		glog.V(8).Infof("session %s is finished and expired before now", ssName)
	} else {
		glog.V(8).Infof("adding session %s to workqueue after %s", ssName, timeUntilExpires.String())
		s.ssWorkqueue.AddAfter(ssName, timeUntilExpires)
		glog.V(8).Infof("added session %s to workqueue", ssName)
	}

	return nil
}

func (s *SessionController) taintVM(vmName string) error {
	glog.V(5).Infof("tainting VM %s", vmName)
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := s.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).Get(s.ctx, vmName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		result.Status.Tainted = true

		result, updateErr := s.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).UpdateStatus(s.ctx, result, metav1.UpdateOptions{})
		if updateErr != nil {
			return updateErr
		}
		glog.V(4).Infof("updated result for vm")

		verifyErr := util.VerifyVM(s.vmLister, result)

		if verifyErr != nil {
			return verifyErr
		}
		return nil
	})
	if retryErr != nil {
		return retryErr
	}

	return nil
}

func (s *SessionController) taintVMC(vmcName string) error {
	glog.V(5).Infof("tainting VMC %s", vmcName)
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := s.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).Get(s.ctx, vmcName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		result.Status.Tainted = true

		result, updateErr := s.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).UpdateStatus(s.ctx, result, metav1.UpdateOptions{})
		if updateErr != nil {
			return updateErr
		}
		verifyErr := util.VerifyVMClaim(s.vmcLister, result)
		if verifyErr != nil {
			return verifyErr
		}
		glog.V(4).Infof("updated result for vmc")
		return nil
	})
	if retryErr != nil {
		return retryErr
	}

	return nil
}

func (s *SessionController) FinishProgress(sessionId string, userId string) {
	now := time.Now()

	progress, err := s.hfClientSet.HobbyfarmV1().Progresses(util.GetReleaseNamespace()).List(s.ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=%s,finished=false", util.SessionLabel, sessionId, util.UserLabel, userId)})

	if err != nil {
		glog.Errorf("error while retrieving progress %v", err)
		return
	}

	for _, p := range progress.Items {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			p.Labels["finished"] = "true"
			p.Spec.LastUpdate = now.Format(time.UnixDate)
			p.Spec.Finished = "true"

			_, updateErr := s.hfClientSet.HobbyfarmV1().Progresses(util.GetReleaseNamespace()).Update(s.ctx, &p, metav1.UpdateOptions{})
			glog.V(4).Infof("updated progress with ID %s", p.Name)

			return updateErr
		})
		if retryErr != nil {
			glog.Errorf("error finishing progress %v", err)
			return
		}
	}
}
