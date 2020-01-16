package course

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

type CourseController struct {
	hfClientSet *hfClientset.Clientset

	cWorkqueue workqueue.DelayingInterface
	cLister    hfListers.CourseLister
	ssLister   hfListers.CourseSessionLister

	cSynced cache.InformerSynced
}

func NewCourseController(hfClientSet *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*CourseController, error) {
	cController := CourseController{}
	cController.hfClientSet = hfClientSet
	cController.cSynced = hfInformerFactory.Hobbyfarm().V1().Courses().Informer().HasSynced

	//ssController.ssWorkqueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "CourseSession")
	cController.cWorkqueue = workqueue.NewNamedDelayingQueue("cc-c")
	cController.cLister = hfInformerFactory.Hobbyfarm().V1().Courses().Lister()
	cController.ssLister = hfInformerFactory.Hobbyfarm().V1().CourseSessions().Lister()

	cInformer := hfInformerFactory.Hobbyfarm().V1().Courses().Informer()

	cInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: cController.enqueueC,
		UpdateFunc: func(old, new interface{}) {
			cController.enqueueC(new)
		},
		DeleteFunc: cController.enqueueC,
	}, time.Minute*30)

	return &cController, nil
}

func (c *CourseController) enqueueC(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		//utilruntime.HandleError(err)
		return
	}
	glog.V(8).Infof("Enqueueing ss %s", key)
	//s.ssWorkqueue.AddRateLimited(key)
	c.cWorkqueue.Add(key)
}

func (c *CourseController) Run(stopCh <-chan struct{}) error {
	defer c.cWorkqueue.ShutDown()

	glog.V(4).Infof("Starting Course controller")
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.cSynced); !ok {
		return fmt.Errorf("failed to wait for course cache to sync")
	}
	glog.Info("Starting course controller workers")
	go wait.Until(c.runCWorker, time.Second, stopCh)
	glog.Info("Started course controller workers")
	//if ok := cache.WaitForCacheSync(stopCh, )
	<-stopCh
	return nil
}

func (c *CourseController) runCWorker() {
	glog.V(6).Infof("Starting course worker")
	for c.processNextCourse() {

	}
}

func (c *CourseController) processNextCourse() bool {
	obj, shutdown := c.cWorkqueue.Get()

	if shutdown {
		return false
	}

	err := func() error {
		defer c.cWorkqueue.Done(obj)
		glog.V(8).Infof("processing course in course controller: %v", obj)
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string))
		if err != nil {
			glog.Errorf("error while splitting meta namespace key %v", err)
			return nil
		}

		err = c.reconcileCourse(objName)

		if err != nil {
			glog.Error(err)
		}
		//s.ssWorkqueue.Forget(obj)
		glog.V(8).Infof("Course processed by course controller %v", objName)

		return nil

	}()

	if err != nil {
		return true
	}

	return true
}

func (c *CourseController) reconcileCourse(cName string) error {
	glog.V(4).Infof("reconciling course %s", cName)

	course, err := c.cLister.Get(cName)

	if err != nil {
		return err
	}

	now := time.Now()

	expires, err := time.Parse(time.UnixDate, course.Status.ExpirationTime)

	if err != nil {
		return err
	}

	timeUntilExpires := expires.Sub(now)

	if expires.Before(now) && !course.Status.Finished {
		// we need to set the course session to finished and delete the vm's
		if course.Status.Paused && course.Status.PausedTime != "" {
			pausedExpiration, err := time.Parse(time.UnixDate, course.Status.PausedTime)
			if err != nil {
				glog.Error(err)
			}

			if pausedExpiration.After(now) {
				glog.V(4).Infof("Course %s was paused, and the pause expiration is after now, skipping clean up.", course.Spec.Id)
				return nil
			}

			glog.V(4).Infof("Course %s was paused, but the pause expiration was before now, so cleaning up.", course.Spec.Id)
		}

		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			result, getErr := c.hfClientSet.HobbyfarmV1().Courses().Get(cName, metav1.GetOptions{})
			if getErr != nil {
				return getErr
			}

			result.Status.Finished = true
			result.Status.Active = false

			result, updateErr := c.hfClientSet.HobbyfarmV1().Courses().Update(result)
			if updateErr != nil {
				return updateErr
			}
			glog.V(4).Infof("updated result for course")

			verifyErr := util.VerifyCourse(c.cLister, result)

			if verifyErr != nil {
				return verifyErr
			}
			return nil
		})
		if retryErr != nil {
			return retryErr
		}
	} else if expires.Before(now) && c.Status.Finished {
		glog.V(8).Infof("course %s is finished and expired before now", cName)
	} else {
		glog.V(8).Infof("adding course %s to workqueue after %s", cName, timeUntilExpires.String())
		c.cWorkqueue.AddAfter(cName, timeUntilExpires)
		glog.V(8).Infof("added course %s to workqueue", cName)
	}

	return nil
}
