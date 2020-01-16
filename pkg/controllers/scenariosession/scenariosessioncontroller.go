package scenariosession

import (
	"fmt"
	"github.com/golang/glog"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	hfListers "github.com/hobbyfarm/gargantua/pkg/client/listers/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	"time"
)

type ScenarioSessionController struct {
	hfClientSet *hfClientset.Clientset

	//ssWorkqueue workqueue.RateLimitingInterface
	ssWorkqueue workqueue.DelayingInterface

	vmLister  hfListers.VirtualMachineLister
	vmcLister hfListers.VirtualMachineClaimLister
	ssLister  hfListers.ScenarioSessionLister

	vmSynced  cache.InformerSynced
	vmcSynced cache.InformerSynced
	ssSynced  cache.InformerSynced
}

func NewScenarioSessionController(hfClientSet *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*ScenarioSessionController, error) {
	ssController := ScenarioSessionController{}
	ssController.hfClientSet = hfClientSet
	ssController.vmSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer().HasSynced
	ssController.vmcSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Informer().HasSynced
	ssController.ssSynced = hfInformerFactory.Hobbyfarm().V1().ScenarioSessions().Informer().HasSynced

	//ssController.ssWorkqueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ScenarioSession")
	ssController.ssWorkqueue = workqueue.NewNamedDelayingQueue("ssc-ss")
	ssController.vmLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Lister()
	ssController.vmcLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Lister()
	ssController.ssLister = hfInformerFactory.Hobbyfarm().V1().ScenarioSessions().Lister()

	ssInformer := hfInformerFactory.Hobbyfarm().V1().ScenarioSessions().Informer()

	ssInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: ssController.enqueueSS,
		UpdateFunc: func(old, new interface{}) {
			ssController.enqueueSS(new)
		},
		DeleteFunc: ssController.enqueueSS,
	}, time.Minute*30)

	return &ssController, nil
}

func (s *ScenarioSessionController) enqueueSS(obj interface{}) {
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

func (s *ScenarioSessionController) Run(stopCh <-chan struct{}) error {
	defer s.ssWorkqueue.ShutDown()

	glog.V(4).Infof("Starting Scenario Session controller")
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

func (s *ScenarioSessionController) runSSWorker() {
	glog.V(6).Infof("Starting scenario session worker")
	for s.processNextScenarioSession() {

	}
}

func (s *ScenarioSessionController) processNextScenarioSession() bool {
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

		err = s.reconcileScenarioSession(objName)

		if err != nil {
			glog.Error(err)
		}
		//s.ssWorkqueue.Forget(obj)
		glog.V(8).Infof("ss processed by scenario session controller %v", objName)

		return nil

	}()

	if err != nil {
		return true
	}

	return true
}

func (s *ScenarioSessionController) reconcileScenarioSession(ssName string) error {
	glog.V(4).Infof("reconciling scenario session %s", ssName)

	ss, err := s.ssLister.Get(ssName)

	if err != nil {
		return err
	}

	now := time.Now()

	expires, err := time.Parse(time.UnixDate, ss.Status.ExpirationTime)

	if err != nil {
		return err
	}

	timeUntilExpires := expires.Sub(now)

	if expires.Before(now) && !ss.Status.Finished {
		// we need to set the scenario session to finished and delete the vm's
		if ss.Status.Paused && ss.Status.PausedTime != "" {
			pausedExpiration, err := time.Parse(time.UnixDate, ss.Status.PausedTime)
			if err != nil {
				glog.Error(err)
			}

			if pausedExpiration.After(now) {
				glog.V(4).Infof("Scenario session %s was paused, and the pause expiration is after now, skipping clean up.", ss.Spec.Id)
				return nil
			}

			glog.V(4).Infof("Scenario session %s was paused, but the pause expiration was before now, so cleaning up.", ss.Spec.Id)
		}
		for _, vmc := range ss.Spec.VmClaimSet {
			vmcObj, err := s.vmcLister.Get(vmc)

			if err != nil {
				break
			}

			for _, vm := range vmcObj.Spec.VirtualMachines {
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
			result, getErr := s.hfClientSet.HobbyfarmV1().ScenarioSessions().Get(ssName, metav1.GetOptions{})
			if getErr != nil {
				return getErr
			}

			result.Status.Finished = true
			result.Status.Active = false

			result, updateErr := s.hfClientSet.HobbyfarmV1().ScenarioSessions().Update(result)
			if updateErr != nil {
				return updateErr
			}
			glog.V(4).Infof("updated result for ss")

			verifyErr := util.VerifyScenarioSession(s.ssLister, result)

			if verifyErr != nil {
				return verifyErr
			}
			return nil
		})
		if retryErr != nil {
			return retryErr
		}
	} else if expires.Before(now) && ss.Status.Finished {
		glog.V(8).Infof("scenario session %s is finished and expired before now", ssName)
	} else {
		glog.V(8).Infof("adding scenario session %s to workqueue after %s", ssName, timeUntilExpires.String())
		s.ssWorkqueue.AddAfter(ssName, timeUntilExpires)
		glog.V(8).Infof("added scenario session %s to workqueue", ssName)
	}

	return nil
}

func (s *ScenarioSessionController) taintVM(vmName string) error {
	glog.V(5).Infof("tainting VM %s", vmName)
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := s.hfClientSet.HobbyfarmV1().VirtualMachines().Get(vmName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		result.Labels["ready"] = "false"
		result.Status.Tainted = true

		result, updateErr := s.hfClientSet.HobbyfarmV1().VirtualMachines().Update(result)
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

func (s *ScenarioSessionController) taintVMC(vmcName string) error {
	glog.V(5).Infof("tainting VMC %s", vmcName)
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := s.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Get(vmcName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		result.Status.Tainted = true

		result, updateErr := s.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Update(result)
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
