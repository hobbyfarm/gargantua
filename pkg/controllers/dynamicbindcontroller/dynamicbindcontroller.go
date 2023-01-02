package dynamicbindcontroller

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

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
)

type DynamicBindController struct {
	hfClientSet                 hfClientset.Interface
	vmtLister 	  hfListers.VirtualMachineTemplateLister

	dynamicBindRequestWorkqueue workqueue.RateLimitingInterface

	dynamicBindRequestLister hfListers.DynamicBindRequestLister

	dynamicBindRequestsSynced cache.InformerSynced
	ctx                       context.Context
}

func NewDynamicBindController(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context) (*DynamicBindController, error) {
	dynamicBindController := DynamicBindController{}
	dynamicBindController.hfClientSet = hfClientSet
	dynamicBindController.vmtLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachineTemplates().Lister()

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
	dynamicBindController.ctx = ctx

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

		dynamicBindRequest, err := d.hfClientSet.HobbyfarmV1().DynamicBindRequests(util.GetReleaseNamespace()).Get(d.ctx, objName, metav1.GetOptions{})
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

	vmClaim, err := d.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util.GetReleaseNamespace()).Get(d.ctx, dynamicBindRequest.Spec.VirtualMachineClaim, metav1.GetOptions{})

	if err != nil {
		glog.Errorf("error retrieving corresponding virtual machine claim %s for dynamic bind request %s", dynamicBindRequest.Spec.VirtualMachineClaim, dynamicBindRequest.Spec.Id)
		return err
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

	dynamicBindConfigurations, err := d.hfClientSet.HobbyfarmV1().DynamicBindConfigurations(util.GetReleaseNamespace()).List(d.ctx, dbcSelector)

	if err != nil {
		glog.Errorf("Error while retrieving dynamic bind configurations, %v", err)
		return err
	}

	var chosenDynamicBindConfiguration *hfv1.DynamicBindConfiguration
	var chosenEnvironment *hfv1.Environment
	var dbcChosen bool
	var provision bool
	dbcChosen = false
	provision = true
	for _, dynamicBindConfiguration := range dynamicBindConfigurations.Items {
		environment, err := d.hfClientSet.HobbyfarmV1().Environments(util.GetReleaseNamespace()).Get(d.ctx, dynamicBindConfiguration.Spec.Environment, metav1.GetOptions{})

		if provisionMethod, ok := environment.Annotations["hobbyfarm.io/provisioner"]; ok {
			if provisionMethod == "external" {
				provision = false
			}
		}
		if err != nil {
			glog.Errorf("Error while retrieving environment %v", err)
			return err
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
			currentVMs, err := d.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).List(d.ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("dynamic=true,environment=%s", environment.Name),
			})
			if err != nil {
				glog.V(4).Infof("error retrieving current vm list, assuming environment empty")
			}

			availableCapacity := util.AvailableRawCapacity(d.hfClientSet, environment.Spec.BurstCapacity, currentVMs.Items, d.ctx)
			availableVMCount := util.MaxVMCountsRaw(d.hfClientSet, needed, *availableCapacity, d.ctx)

			if availableVMCount <= 0 {
				suitable = false
			}

		} else {
			for vmTemplate, vmsNeeded := range needed {
				// first, let's see if the environment itself has capacity

				currentVMs, err := d.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).List(d.ctx, metav1.ListOptions{
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

				currentVMs, err = d.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).List(d.ctx, metav1.ListOptions{
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
						"dynamic":                          "true",
						"dynamicbindrequest":               dynamicBindRequest.Name,
						"dynamicbindconfiguration":         chosenDynamicBindConfiguration.Spec.Id,
						"template":                         vmX.Template,
						"environment":                      chosenDynamicBindConfiguration.Spec.Environment,
						"bound":                            "true",
						"ready":                            "false",
						util.ScheduledEventLabel: chosenDynamicBindConfiguration.ObjectMeta.Labels[util.ScheduledEventLabel],
					},
				},
				Spec: hfv1.VirtualMachineSpec{
					Id:                       vmName,
					VirtualMachineTemplateId: vmX.Template,
					SecretName:               "",
					Protocol:                 "ssh", //default protocol is ssh
					VirtualMachineClaimId:    dynamicBindRequest.Spec.VirtualMachineClaim,
					UserId:                   vmClaim.Spec.UserId,
					Provision:                provision,
					VirtualMachineSetId:      "",
				},
			}

			vmt, err := d.vmtLister.VirtualMachineTemplates(util.GetReleaseNamespace()).Get(vmX.Template)
			if err != nil {
				glog.Errorf("error getting vmt %v", err)
				return err
			}

			config := util.GetVMConfig(chosenEnvironment,vmt)
      
      		protocol, exists := config["protocol"]
      		if exists {
			   vm.Spec.Protocol = protocol
		  	}

			sshUser, exists := config["ssh_username"]
			if exists {
				vm.Spec.SshUsername = sshUser
			}

			// extra label to indicate external provisioning so tfpcontroller ignores this request //
			if provisionMethod, ok := chosenEnvironment.Annotations["hobbyfarm.io/provisioner"]; ok {
				vm.ObjectMeta.Labels["hobbyfarm.io/provisioner"] = provisionMethod
				vm.Spec.Provision = false
			}

			// label to indicate which vmtemplate being used for querying reasons
			vm.ObjectMeta.Labels["hobbyfarm.io/vmtemplate"] = vm.Spec.VirtualMachineTemplateId

			if chosenDynamicBindConfiguration.Spec.RestrictedBind {
				vm.ObjectMeta.Labels["restrictedbind"] = "true"
				vm.ObjectMeta.Labels["restrictedbindvalue"] = chosenDynamicBindConfiguration.Spec.RestrictedBindValue
			} else {
				vm.ObjectMeta.Labels["restrictedbind"] = "false"
			}
			newVm, err := d.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).Create(d.ctx, vm, metav1.CreateOptions{})

			if err != nil {
				glog.Error(err)
			}

			newVm.Status = hfv1.VirtualMachineStatus{
				Status:        hfv1.VmStatusRFP,
				Allocated:     true,
				Tainted:       false,
				WsEndpoint:    chosenEnvironment.Spec.WsEndpoint,
				PublicIP:      "",
				PrivateIP:     "",
				EnvironmentId: chosenEnvironment.Name,
				Hostname:      "",
			}

			_, err = d.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).UpdateStatus(d.ctx, newVm, metav1.UpdateOptions{})

			if err != nil {
				glog.Error(err)
			}

			virtualMachines[vmClaimVMName] = newVm.Name
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

func (d *DynamicBindController) updateDynamicBindRequestStatus(dynamicBindAttempts int, expired bool, fulfilled bool, dynamicBindConfigurationId string, virtualMachineIds map[string]string, dynamicBindRequestId string) error {

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := d.hfClientSet.HobbyfarmV1().DynamicBindRequests(util.GetReleaseNamespace()).Get(d.ctx, dynamicBindRequestId, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("error retrieving latest version of DynamicBindRequest %s: %v", dynamicBindRequestId, getErr)
		}

		result.Status.CurrentAttempts = dynamicBindAttempts
		result.Status.Expired = expired
		result.Status.Fulfilled = fulfilled
		result.Status.DynamicBindConfigurationId = dynamicBindConfigurationId
		result.Status.VirtualMachineIds = virtualMachineIds

		_, updateErr := d.hfClientSet.HobbyfarmV1().DynamicBindRequests(util.GetReleaseNamespace()).UpdateStatus(d.ctx, result, metav1.UpdateOptions{})
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
