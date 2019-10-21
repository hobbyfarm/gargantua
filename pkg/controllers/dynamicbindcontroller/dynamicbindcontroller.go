package dynamicbindcontroller

import (
	"fmt"
	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	hfListers "github.com/hobbyfarm/gargantua/pkg/client/listers/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"math/rand"
	"strings"
	"time"
)

type DynamicBindController struct {
	hfClientSet    *hfClientset.Clientset
	dynamicBindRequestWorkqueue    workqueue.RateLimitingInterface

	dynamicBindRequestLister hfListers.DynamicBindRequestLister

	dynamicBindRequestsSynced cache.InformerSynced
}

func NewDynamicBindController(hfClientSet *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*DynamicBindController, error) {
	dynamicBindController := DynamicBindController{}
	dynamicBindController.hfClientSet = hfClientSet

	dynamicBindController.dynamicBindRequestsSynced = hfInformerFactory.Hobbyfarm().V1().DynamicBindRequests().Informer().HasSynced

	dynamicBindController.dynamicBindRequestWorkqueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "DynamicBindRequest")

	dynamicBindRequestInformer := hfInformerFactory.Hobbyfarm().V1().DynamicBindRequests().Informer()

	dynamicBindRequestInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: dynamicBindController.enqueueDynamicBindRequest,
		UpdateFunc: func(old, new interface{}) {
			dynamicBindController.enqueueDynamicBindRequest(new)
		},
		DeleteFunc: dynamicBindController.enqueueDynamicBindRequest,
	}, time.Second*30)

	return &dynamicBindController, nil
}

func (d *DynamicBindController) enqueueDynamicBindRequest(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		//utilruntime.HandleError(err)
		return
	}
	glog.V(8).Infof("Enqueueing dynamic bind request %s", key)
	d.dynamicBindRequestWorkqueue.AddRateLimited(key)
}

func (d *DynamicBindController) Run(stopCh <-chan struct{}) error {
	defer d.dynamicBindRequestWorkqueue.ShutDown()

	glog.V(4).Infof("Starting dynamic bind controller")
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, d.dynamicBindRequestsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	glog.Info("Dynamic bind controller caches synced, starting controller loops")
	go wait.Until(d.runDynamicBindWorker, time.Second, stopCh)
	glog.Info("Started Dynamic Bind Controller worker")
	<-stopCh
	return nil
}

func (d *DynamicBindController) runDynamicBindWorker() {
	for d.processNextDynamicBindRequest() {

	}
}

func (d *DynamicBindController) processNextDynamicBindRequest() bool {
	obj, shutdown := d.dynamicBindRequestWorkqueue.Get()

	glog.V(8).Infof("processing dynamic bind request")

	if shutdown {
		return false
	}

	err := func() error {
		defer d.dynamicBindRequestWorkqueue.Done(obj)
		glog.V(4).Infof("processing dynamic bind request in dynamic bind controller: %v", obj)
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string)) // this is actually not necessary because dynamic bind request's are not namespaced yet...
		if err != nil {
			glog.Errorf("error while splitting meta namespace key %v", err)
			//e.vmWorkqueue.AddRateLimited(obj)
			return nil
		}

		dynamicBindRequest, err := d.hfClientSet.HobbyfarmV1().DynamicBindRequests().Get(objName, metav1.GetOptions{})
		if err != nil {
			glog.Errorf("error while retrieving dynamic bind request %s, likely deleted %v", objName, err)
			d.dynamicBindRequestWorkqueue.Forget(obj)
			return nil
		}

		err = d.reconcileDynamicBindRequest(dynamicBindRequest)
		if err != nil {
			glog.Error(err)
		}

		d.dynamicBindRequestWorkqueue.Forget(obj)
		glog.V(4).Infof("dynamic bind request processed by dynamic bind controller %v", objName)

		return nil

	}()

	if err != nil {
		return true
	}

	return true
}

func (d *DynamicBindController) reconcileDynamicBindRequest(dynamicBindRequest *hfv1.DynamicBindRequest) error {

	dynamicBindConfiguration, err := d.hfClientSet.HobbyfarmV1().DynamicBindConfigurations().Get(dynamicBindRequest.Spec.DynamicBindConfiguration, metav1.GetOptions{})

	if err != nil {
		glog.Errorf("Error while retrieving dynamic bind configuration, %v", err)
	}

	var chosenEnvironment *hfv1.Environment
	chosenEnvironmentCapacityCount := 0

	for _, environmentId := range dynamicBindConfiguration.Spec.Environments {
		environment, err := d.hfClientSet.HobbyfarmV1().Environments().Get(environmentId, metav1.GetOptions{})

		if err != nil {
			glog.Errorf("Error while retrieving environment %v", err)
		}

		if !environment.Spec.BurstCapable {
			glog.V(8).Infof("Environment %s is not burst capable. Ignoring.", environmentId)
			continue
		}

		availableVMCount := 0

		if environment.Spec.CapacityMode == hfv1.CapacityModeRaw {
			currentVMs, err := d.hfClientSet.HobbyfarmV1().VirtualMachines().List(metav1.ListOptions{
				LabelSelector: fmt.Sprintf("dynamic=true,environment=%s", environmentId),
			})
			if err != nil {
				glog.V(4).Infof("error retrieving current vm list, assuming environment empty")
			}

			availableCapacity := util.AvailableRawCapacity(d.hfClientSet, environment.Spec.BurstCapacity, currentVMs.Items)

			availableVMCount = util.MaxVMCountRaw(d.hfClientSet, dynamicBindConfiguration.Spec.VirtualMachineTemplate, *availableCapacity)

		} else {
			currentVMs, err := d.hfClientSet.HobbyfarmV1().VirtualMachines().List(metav1.ListOptions{
				LabelSelector: fmt.Sprintf("dynamic=true,environment=%s,template=%s", environmentId, dynamicBindConfiguration.Spec.VirtualMachineTemplate),
			})
			if err != nil {
				glog.V(4).Infof("error retrieving current vm list, assuming environment empty")
			}
			availableVMCount = environment.Spec.BurstCountCapacity[dynamicBindConfiguration.Spec.VirtualMachineTemplate] - len(currentVMs.Items)
		}

		if availableVMCount > chosenEnvironmentCapacityCount {
			chosenEnvironmentCapacityCount = availableVMCount
			chosenEnvironment = environment
		}
	}

	if chosenEnvironmentCapacityCount > 0 {
		vmName := strings.Join([]string{dynamicBindConfiguration.Spec.BaseName, fmt.Sprintf("%08x", rand.Uint32())}, "-")
		vm := &hfv1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name: vmName,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "hobbyfarm.io/v1",
						Kind:       "DynamicBindRequest",
						Name:       dynamicBindRequest.Name,
						UID:        dynamicBindRequest.UID,
					},
				},
				Labels: map[string]string{
					"dynamic":     "true",
					"dynamicbindrequest": dynamicBindRequest.Name,
					"template":    dynamicBindRequest.Spec.VirtualMachineTemplate,
					"environment": chosenEnvironment.Name,
					"bound":       "true",
					"ready":       "false",
				},
			},
			Spec: hfv1.VirtualMachineSpec{
				Id:                       vmName,
				VirtualMachineTemplateId: dynamicBindConfiguration.Spec.VirtualMachineTemplate,
				KeyPair:                  "",
				VirtualMachineClaimId:    dynamicBindRequest.Spec.VirtualMachineClaim,
				UserId:                   "",
				Provision:                true,
				VirtualMachineSetId:      "",
			},
			Status: hfv1.VirtualMachineStatus{
				Status:        hfv1.VmStatusRFP,
				Allocated:     false,
				Tainted:       false,
				WsEndpoint:    chosenEnvironment.Spec.WsEndpoint,
				PublicIP:      "",
				PrivateIP:     "",
				EnvironmentId: chosenEnvironment.Name,
				Hostname:      "",
			},
		}
		if dynamicBindConfiguration.Spec.RestrictedBind {
			vm.ObjectMeta.Labels["restrictedbind"] = "true"
			vm.ObjectMeta.Labels["restrictedbindvalue"] = dynamicBindConfiguration.Spec.RestrictedBindValue
		} else {
			vm.ObjectMeta.Labels["restrictedbind"] = "false"
		}
		vm, err := d.hfClientSet.HobbyfarmV1().VirtualMachines().Create(vm)
		if err != nil {
			glog.Error(err)
		}

	}

	glog.V(5).Infof("provisioning %d vm")

	return nil
}
