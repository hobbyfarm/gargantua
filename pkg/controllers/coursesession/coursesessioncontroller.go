package coursesession

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

type CourseSessionController struct {
	hfClientSet *hfClientset.Clientset

	//csWorkqueue workqueue.RateLimitingInterface
	csWorkqueue workqueue.DelayingInterface

	vmLister  hfListers.VirtualMachineLister
	vmcLister hfListers.VirtualMachineClaimLister
	csLister  hfListers.CourseSessionLister

	vmSynced  cache.InformerSynced
	vmcSynced cache.InformerSynced
	csSynced  cache.InformerSynced
}

func NewCourseSessionController(hfClientSet *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*CourseSessionController, error) {
	csController := CourseSessionController{}
	csController.hfClientSet = hfClientSet
	csController.vmSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer().HasSynced
	csController.vmcSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Informer().HasSynced
	csController.csSynced = hfInformerFactory.Hobbyfarm().V1().CourseSessions().Informer().HasSynced

	//ssController.ssWorkqueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "CourseSession")
	csController.csWorkqueue = workqueue.NewNamedDelayingQueue("ssc-ss")
	csController.vmLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Lister()
	csController.vmcLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachineClaims().Lister()
	csController.csLister = hfInformerFactory.Hobbyfarm().V1().CourseSessions().Lister()

	ssInformer := hfInformerFactory.Hobbyfarm().V1().CourseSessions().Informer()

	ssInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: csController.enqueueCS,
		UpdateFunc: func(old, new interface{}) {
			csController.enqueueCS(new)
		},
		DeleteFunc: csController.enqueueCS,
	}, time.Minute*30)

	return &csController, nil
}

func (s *CourseSessionController) enqueueCS(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		//utilruntime.HandleError(err)
		return
	}
	glog.V(8).Infof("Enqueueing cs %s", key)
	//s.ssWorkqueue.AddRateLimited(key)
	s.csWorkqueue.Add(key)
}

func (s *CourseSessionController) Run(stopCh <-chan struct{}) error {
	defer s.csWorkqueue.ShutDown()

	glog.V(4).Infof("Starting Course Session controller")
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, s.vmSynced, s.vmcSynced, s.csSynced); !ok {
		return fmt.Errorf("failed to wait for vm, vmc, and cs caches to sync")
	}
	glog.Info("Starting cs controller workers")
	go wait.Until(s.runCSWorker, time.Second, stopCh)
	glog.Info("Started cs controller workers")
	//if ok := cache.WaitForCacheSync(stopCh, )
	<-stopCh
	return nil
}

func (s *CourseSessionController) runCSWorker() {
	glog.V(6).Infof("Starting scenario session worker")
	for s.processNextCourseSession() {

	}
}

func (s *CourseSessionController) processNextCourseSession() bool {
	obj, shutdown := s.csWorkqueue.Get()

	if shutdown {
		return false
	}

	err := func() error {
		defer s.csWorkqueue.Done(obj)
		glog.V(8).Infof("processing cs in cs controller: %v", obj)
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string))
		if err != nil {
			glog.Errorf("error while splitting meta namespace key %v", err)
			return nil
		}

		err = s.reconcileCourseSession(objName)

		if err != nil {
			glog.Error(err)
		}
		//c.csWorkqueue.Forget(obj)
		glog.V(8).Infof("cs processed by course session controller %v", objName)

		return nil

	}()

	if err != nil {
		return true
	}

	return true
}

func (s *CourseSessionController) reconcileCourseSession(csName string) error {
	glog.V(4).Infof("reconciling course session %s", csName)

	cs, err := s.csLister.Get(csName)

	if err != nil {
		return err
	}

	now := time.Now()

	expires, err := time.Parse(time.UnixDate, cs.Status.ExpirationTime)

	if err != nil {
		return err
	}

	timeUntilExpires := expires.Sub(now)

	if expires.Before(now) && !cs.Status.Finished {
		// we need to set the course session to finished and delete the vm's
		if cs.Status.Paused && cs.Status.PausedTime != "" {
			pausedExpiration, err := time.Parse(time.UnixDate, cs.Status.PausedTime)
			if err != nil {
				glog.Error(err)
			}

			if pausedExpiration.After(now) {
				glog.V(4).Infof("Course session %s was paused, and the pause expiration is after now, skipping clean up.", cs.Spec.Id)
				return nil
			}

			glog.V(4).Infof("Course session %s was paused, but the pause expiration was before now, so cleaning up.", cs.Spec.Id)
		}
		for _, vmc := range cs.Spec.VmClaimSet {
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
			result, getErr := s.hfClientSet.HobbyfarmV1().CourseSessions().Get(csName, metav1.GetOptions{})
			if getErr != nil {
				return getErr
			}

			result.Status.Finished = true
			result.Status.Active = false

			result, updateErr := s.hfClientSet.HobbyfarmV1().CourseSessions().Update(result)
			if updateErr != nil {
				return updateErr
			}
			glog.V(4).Infof("updated result for cs")

			verifyErr := util.VerifyCourseSession(s.csLister, result)

			if verifyErr != nil {
				return verifyErr
			}
			return nil
		})
		if retryErr != nil {
			return retryErr
		}
	} else if expires.Before(now) && cs.Status.Finished {
		glog.V(8).Infof("course session %s is finished and expired before now", csName)
	} else {
		glog.V(8).Infof("adding course session %s to workqueue after %s", csName, timeUntilExpires.String())
		s.csWorkqueue.AddAfter(csName, timeUntilExpires)
		glog.V(8).Infof("added course session %s to workqueue", csName)
	}

	return nil
}

func (s *CourseSessionController) taintVM(vmName string) error {
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

func (s *CourseSessionController) taintVMC(vmcName string) error {
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
