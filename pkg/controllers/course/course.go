package course

import (
	"fmt"
	"github.com/golang/glog"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	hfListers "github.com/hobbyfarm/gargantua/pkg/client/listers/hobbyfarm.io/v1"
	// "github.com/hobbyfarm/gargantua/pkg/util"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	// "k8s.io/client-go/util/retry"
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

	// course, err := c.cLister.Get(cName)
	// if err != nil {
	// 	return err
	// }

	return nil
}
