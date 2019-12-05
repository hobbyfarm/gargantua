package dynamicbindcontroller

import (
	"fmt"
	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	hfListers "github.com/hobbyfarm/gargantua/pkg/client/listers/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
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
	}, time.Minute*30)

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

		if dynamicBindRequest.Status.Fulfilled || dynamicBindRequest.Status.Expired {
			d.dynamicBindRequestWorkqueue.Forget(obj)
			return nil
		}

		err = d.reconcileDynamicBindRequest(dynamicBindRequest)
		if err != nil {
			d.dynamicBindRequestWorkqueue.AddRateLimited(obj)
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

	d.updateDynamicBindRequestStatus(dynamicBindRequest.Status.CurrentAttempts+1, false, false, "", make(map[string]string), dynamicBindRequest.Spec.Id)

	vmClaim, err := d.hfClientSet.HobbyfarmV1().VirtualMachineClaims().Get(dynamicBindRequest.Spec.VirtualMachineClaim, metav1.GetOptions{})

	if err != nil {
		glog.Errorf("error retrieving corresponding virtual machine claim %s for dynamic bind request %s", dynamicBindRequest.Spec.VirtualMachineClaim, dynamicBindRequest.Spec.Id)
	}

	var dbcSelector metav1.ListOptions
	if vmClaim.Spec.RestrictedBind {
		dbcSelector = metav1.ListOptions{
			LabelSelector: fmt.Sprintf("restrictedbind=true,restrictedbindvalue=%s", vmClaim.Spec.RestrictedBindValue),
		}
	} else {
		dbcSelector = metav1.ListOptions{
			LabelSelector: "restrictedbind=false",
		}
	}

	dynamicBindConfigurations, err := d.hfClientSet.HobbyfarmV1().DynamicBindConfigurations().List(dbcSelector)

	if err != nil {
		glog.Errorf("Error while retrieving dynamic bind configurations, %v", err)
	}

	var chosenDynamicBindConfiguration *hfv1.DynamicBindConfiguration
	var chosenEnvironment *hfv1.Environment
	var dbcChosen bool
	dbcChosen = false

	for _, dynamicBindConfiguration := range dynamicBindConfigurations.Items {
			environment, err := d.hfClientSet.HobbyfarmV1().Environments().Get(dynamicBindConfiguration.Spec.Environment, metav1.GetOptions{})

			if err != nil {
				glog.Errorf("Error while retrieving environment %v", err)
				return nil
			}

			if !environment.Spec.BurstCapable {
				glog.V(8).Infof("Environment %s is not burst capable. Ignoring.", environment.Name)
				continue
			}

			suitable := true

			needed := make(map[string]int)

			for _, template := range vmClaim.Spec.VirtualMachines {
				if template.VirtualMachineId == "" {
					if val, ok := needed[template.Template]; ok {
						needed[template.Template] = val + 1
					} else {
						needed[template.Template] = 1
					}
				}
			}

			if environment.Spec.CapacityMode == hfv1.CapacityModeRaw {
				currentVMs, err := d.hfClientSet.HobbyfarmV1().VirtualMachines().List(metav1.ListOptions{
					LabelSelector: fmt.Sprintf("dynamic=true,environment=%s", environment.Name),
				})
				if err != nil {
					glog.V(4).Infof("error retrieving current vm list, assuming environment empty")
				}

				availableCapacity := util.AvailableRawCapacity(d.hfClientSet, environment.Spec.BurstCapacity, currentVMs.Items)
				availableVMCount := util.MaxVMCountsRaw(d.hfClientSet, needed, *availableCapacity)

				if availableVMCount <= 0 {
					suitable = false
				}

			} else {
				for vmTemplate, vmsNeeded := range needed {
					// first, let's see if the environment itself has capacity

					currentVMs, err := d.hfClientSet.HobbyfarmV1().VirtualMachines().List(metav1.ListOptions{
						LabelSelector: fmt.Sprintf("dynamic=true,environment=%s,template=%s", environment.Name, vmTemplate),
					})
					if err != nil {
						glog.V(4).Infof("error retrieving current vm list, assuming environment empty")
					}
					availableVMCount := environment.Spec.BurstCountCapacity[vmTemplate] - len(currentVMs.Items)

					if availableVMCount < vmsNeeded {
						// this environment does not suit us. move on.
						suitable = false
						break
					}

					currentVMs, err = d.hfClientSet.HobbyfarmV1().VirtualMachines().List(metav1.ListOptions{
						LabelSelector: fmt.Sprintf("dynamic=true,dynamicbindconfig=%s,environment=%s,template=%s", dynamicBindConfiguration.Spec.Id, environment.Name, vmTemplate),
					})
					if err != nil {
						glog.V(4).Infof("error retrieving current vm list, assuming environment empty")
					}
					availableVMCount = dynamicBindConfiguration.Spec.BurstCountCapacity[vmTemplate] - len(currentVMs.Items)

					if availableVMCount < vmsNeeded {
						// this DBC has no more provisioning capacity
						suitable = false
						break
					}
				}
			}

			if suitable {
				chosenDynamicBindConfiguration = &dynamicBindConfiguration
				chosenEnvironment = environment
				dbcChosen = true
				break
			}
	}

	if dbcChosen {
		virtualMachines := make(map[string]string)

		for vmClaimVMName, vmX := range vmClaim.Spec.VirtualMachines {
			vmName := strings.Join([]string{chosenDynamicBindConfiguration.Spec.BaseName, fmt.Sprintf("%08x", rand.Uint32())}, "-")
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
						"dynamic":                  "true",
						"dynamicbindrequest":       dynamicBindRequest.Name,
						"dynamicbindconfiguration": chosenDynamicBindConfiguration.Spec.Id,
						"template":                 vmX.Template,
						"environment":              chosenDynamicBindConfiguration.Spec.Environment,
						"bound":                    "true",
						"ready":                    "false",
					},
				},
				Spec: hfv1.VirtualMachineSpec{
					Id:                       vmName,
					VirtualMachineTemplateId: vmX.Template,
					KeyPair:                  "",
					VirtualMachineClaimId:    dynamicBindRequest.Spec.VirtualMachineClaim,
					UserId:                   vmClaim.Spec.UserId,
					Provision:                true,
					VirtualMachineSetId:      "",
				},
				Status: hfv1.VirtualMachineStatus{
					Status:        hfv1.VmStatusRFP,
					Allocated:     true,
					Tainted:       false,
					WsEndpoint:    chosenEnvironment.Spec.WsEndpoint,
					PublicIP:      "",
					PrivateIP:     "",
					EnvironmentId: chosenEnvironment.Name,
					Hostname:      "",
				},
			}
			if chosenDynamicBindConfiguration.Spec.RestrictedBind {
				vm.ObjectMeta.Labels["restrictedbind"] = "true"
				vm.ObjectMeta.Labels["restrictedbindvalue"] = chosenDynamicBindConfiguration.Spec.RestrictedBindValue
			} else {
				vm.ObjectMeta.Labels["restrictedbind"] = "false"
			}
			vm, err := d.hfClientSet.HobbyfarmV1().VirtualMachines().Create(vm)
			if err != nil {
				glog.Error(err)
			}
			virtualMachines[vmClaimVMName] = vm.Name
		}

		d.updateDynamicBindRequestStatus(dynamicBindRequest.Spec.Attempts, false, true, chosenDynamicBindConfiguration.Spec.Id, virtualMachines, dynamicBindRequest.Spec.Id)

	} else {
		// check to see if we're above our attempt threshold
		if dynamicBindRequest.Status.CurrentAttempts > dynamicBindRequest.Spec.Attempts {
			d.updateDynamicBindRequestStatus(dynamicBindRequest.Status.CurrentAttempts, true, false, "", make(map[string]string), dynamicBindRequest.Spec.Id)
		} else {
			return fmt.Errorf("did not bind dynamic vms for dynamic bind request %s", dynamicBindRequest.Spec.Id)
		}
	}

	return nil
}

func (d *DynamicBindController) updateDynamicBindRequestStatus(dynamicBindAttempts int, expired bool, fulfilled bool, dynamicBindConfigurationId string, virtualMachineIds map[string]string, dynamicBindRequestId string) (error) {

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := d.hfClientSet.HobbyfarmV1().DynamicBindRequests().Get(dynamicBindRequestId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of DynamicBindRequest %s: %v", dynamicBindRequestId, getErr)
		}

		result.Status.CurrentAttempts = dynamicBindAttempts
		result.Status.Expired = expired
		result.Status.Fulfilled = fulfilled
		result.Status.DynamicBindConfigurationId = dynamicBindConfigurationId
		result.Status.VirtualMachineIds = virtualMachineIds

		_, updateErr := d.hfClientSet.HobbyfarmV1().DynamicBindRequests().Update(result)
		if updateErr != nil {
			return updateErr
		}
		glog.V(4).Infof("updated result for dynamic bind request")
		return nil
	})
	if retryErr != nil {
		return fmt.Errorf("error updating DynamicBindRequest: %s, %v", dynamicBindRequestId, retryErr)
	}
	return nil
}
