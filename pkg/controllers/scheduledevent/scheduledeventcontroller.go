package scheduledevent

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
)

type ScheduledEventController struct {
	hfClientSet hfClientset.Interface

	//seWorkqueue workqueue.RateLimitingInterface
	seWorkqueue workqueue.DelayingInterface
	seSynced    cache.InformerSynced
}

var baseNameScheduledPrefix string
var baseNameDynamicPrefix string

const (
	ScheduledEventBaseDelay = 5 * time.Millisecond
	ScheduledEventMaxDelay  = 300 * time.Second
)

func init() {
	bnsp := os.Getenv("HF_BASENAME_SCHEDULED_PREFIX")
	if bnsp == "" {
		baseNameScheduledPrefix = "scheduled"
	} else {
		baseNameScheduledPrefix = bnsp
	}

	bndp := os.Getenv("HF_BASENAME_DYNAMIC_PREFIX")
	if bndp == "" {
		baseNameDynamicPrefix = "dynamic"
	} else {
		baseNameDynamicPrefix = bndp
	}
}

func NewScheduledEventController(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) (*ScheduledEventController, error) {
	seController := ScheduledEventController{}
	seController.hfClientSet = hfClientSet
	seController.seSynced = hfInformerFactory.Hobbyfarm().V1().ScheduledEvents().Informer().HasSynced

	//seController.seWorkqueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ScheduledEvent")
	seController.seWorkqueue = workqueue.NewNamedRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(ScheduledEventBaseDelay, ScheduledEventMaxDelay), "sec-se")
	seInformer := hfInformerFactory.Hobbyfarm().V1().ScheduledEvents().Informer()

	seInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: seController.enqueueSE,
		UpdateFunc: func(old, new interface{}) {
			seController.enqueueSE(new)
		},
		DeleteFunc: seController.enqueueSE,
	}, time.Minute*30)

	return &seController, nil
}

func (s *ScheduledEventController) enqueueSE(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		//utilruntime.HandleError(err)
		return
	}
	glog.V(8).Infof("Enqueueing se %s", key)
	//s.seWorkqueue.AddRateLimited(key)
	s.seWorkqueue.Add(key)
}

func (s *ScheduledEventController) Run(stopCh <-chan struct{}) error {
	defer s.seWorkqueue.ShutDown()

	glog.V(4).Infof("Starting Scheduled Event controller")
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, s.seSynced); !ok {
		return fmt.Errorf("failed to wait for vm, vmc, and ss caches to sync")
	}
	glog.Info("Starting se controller workers")
	go wait.Until(s.runSEWorker, time.Second, stopCh)
	glog.Info("Started se controller workers")
	//if ok := cache.WaitForCacheSync(stopCh, )
	<-stopCh
	return nil
}

func (s *ScheduledEventController) runSEWorker() {
	glog.V(6).Infof("Starting scheduled event worker")
	for s.processNextScheduledEvent() {

	}
}

func (s *ScheduledEventController) processNextScheduledEvent() bool {
	obj, shutdown := s.seWorkqueue.Get()

	if shutdown {
		return false
	}

	err := func() error {
		defer s.seWorkqueue.Done(obj)
		glog.V(8).Infof("processing se in se controller: %v", obj)
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string))
		if err != nil {
			glog.Errorf("error while splitting meta namespace key %v", err)
			return nil
		}

		err = s.reconcileScheduledEvent(objName)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				glog.Error(err)
				s.seWorkqueue.Add(objName)
			}
		}
		//s.seWorkqueue.Forget(obj)
		glog.V(8).Infof("se processed by scheduled event controller %v", objName)

		return nil

	}()

	if err != nil {
		return true
	}

	return true
}

func (s ScheduledEventController) completeScheduledEvent(se *hfv1.ScheduledEvent) error {
	glog.V(6).Infof("ScheduledEvent %s is done, deleting corresponding VMSets and marking as finished", se.Name)
	// scheduled event is finished, we need to set the scheduled event to finished and delete the vm's

	err := s.deleteVMSetsFromScheduledEvent(se)

	if err != nil {
		return err
	}

	// update the scheduled event and set the various flags accordingly (provisioned, ready, finished)
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		seToUpdate, err := s.hfClientSet.HobbyfarmV1().ScheduledEvents().Get(se.Name, metav1.GetOptions{})

		if err != nil {
			return err
		}

		seToUpdate.Status.Provisioned = true
		seToUpdate.Status.Ready = false
		seToUpdate.Status.Finished = true

		_, updateErr := s.hfClientSet.HobbyfarmV1().ScheduledEvents().Update(seToUpdate)
		glog.V(4).Infof("updated result for scheduled event")

		return updateErr
	})

	if retryErr != nil {
		return retryErr
	}

	return nil // break (return) here because we're done with this SE.
}

func (s ScheduledEventController) deleteVMSetsFromScheduledEvent(se *hfv1.ScheduledEvent) error {
	// get a list of the vmsets corresponding to this scheduled event
	vmsList, err := s.hfClientSet.HobbyfarmV1().VirtualMachineSets().List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("scheduledevent=%s", se.Name),
	})
	if err != nil {
		return err
	}

	// for each vmset that belongs to this to-be-stopped scheduled event, delete that vmset
	for _, vms := range vmsList.Items {
		err := s.hfClientSet.HobbyfarmV1().VirtualMachineSets().Delete(vms.Name, &metav1.DeleteOptions{})
		if err != nil {
			glog.Errorf("error deleting virtualmachineset %v", err)
		}
	}
	return nil
}

func (s ScheduledEventController) provisionScheduledEvent(templates *hfv1.VirtualMachineTemplateList, se *hfv1.ScheduledEvent) error {
	glog.V(6).Infof("ScheduledEvent %s is ready to be provisioned", se.Name)
	// start creating resources related to this
	vmSets := []string{}

	/**
	The general flow here is to calculate how much resources (cpu, mem, storage) are currently
	being used, and then compare that to what is needed. If needed > used, we're going to still
	provision (for some reason), but at least we'll tell the user about it
		e.g. --> glog.Errorf("we are overprovisioning this environment %s by CPU...
	*/

	// begin by calculating what is currently being used in the environment
	for envName, vmtMap := range se.Spec.RequiredVirtualMachines {
		// get the environment we're provisioning into (envName)
		env, err := s.hfClientSet.HobbyfarmV1().Environments().Get(envName, metav1.GetOptions{})

		// get all vmsets that are being provisioned into this environment (label selector)
		vmsList, err := s.hfClientSet.HobbyfarmV1().VirtualMachineSets().List(metav1.ListOptions{
			LabelSelector: fmt.Sprintf("environment=%s", envName),
		})
		if err != nil {
			glog.Errorf("error while retrieving virtual machine sets %v", err)
		}

		_, usedCapacity := calculateUsedCapacity(env, vmsList, templates)

		_, neededCapacity := calculateNeededCapacity(env, vmtMap, templates)

		if env.Spec.CapacityMode == hfv1.CapacityModeRaw {
			if env.Spec.Capacity.CPU < (+neededCapacity.CPU) {
				glog.Errorf("we are overprovisioning this environment %s by CPU, capacity is %d but need %d", envName, env.Spec.Capacity.CPU, usedCapacity.CPU+neededCapacity.CPU)
			}
			if env.Spec.Capacity.Memory < (usedCapacity.Memory + neededCapacity.Memory) {
				glog.Errorf("we are overprovisioning this environment %s by Memory, capacity is %d but need %d", envName, env.Spec.Capacity.Memory, usedCapacity.Memory+neededCapacity.Memory)
			}
			if env.Spec.Capacity.Storage < (usedCapacity.Storage + neededCapacity.Storage) {
				glog.Errorf("we are overprovisioning this environment %s by Storage, capacity is %d but need %d", envName, env.Spec.Capacity.Storage, usedCapacity.Storage+neededCapacity.Storage)
			}
		} else if env.Spec.CapacityMode == hfv1.CapacityModeCount {
			// todo: actually check for capacity usage
		}

		// create the virtualmachinesets now

		for templateName, count := range vmtMap {
			if count > 0 && !se.Spec.OnDemand { // only setup vmsets if >0 VMs are requested, and they aren't ondemand
				vmsRand := fmt.Sprintf("%s-%08x", baseNameScheduledPrefix, rand.Uint32())
				vmsName := strings.Join([]string{"se", se.Name, "vms", vmsRand}, "-")
				vmSets = append(vmSets, vmsName)
				vms := &hfv1.VirtualMachineSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: vmsName,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "hobbyfarm.io/v1",
								Kind:       "ScheduledEvent",
								Name:       se.Name,
								UID:        se.UID,
							},
						},
						Labels: map[string]string{
							"environment":    env.Name,
							"scheduledevent": se.Name,
						},
					},
					Spec: hfv1.VirtualMachineSetSpec{
						Count:       count,
						Environment: envName,
						VMTemplate:  templateName,
						BaseName:    vmsRand,
					},
				}
				if se.Spec.RestrictedBind {
					vms.Spec.RestrictedBind = true
					vms.Spec.RestrictedBindValue = se.Spec.RestrictedBindValue
				} else {
					vms.Spec.RestrictedBind = false
				}
				vms, err := s.hfClientSet.HobbyfarmV1().VirtualMachineSets().Create(vms)
				if err != nil {
					glog.Error(err)
				}
			}
		}

		// create the dynamic bind configurations
		dbcRand := fmt.Sprintf("%s-%08x", baseNameDynamicPrefix, rand.Uint32())
		dbcName := strings.Join([]string{"se", se.Name, "dbc", dbcRand}, "-")
		emptyCap := hfv1.CMSStruct{
			CPU:     0,
			Memory:  0,
			Storage: 0,
		}

		bcc := map[string]int{}

		//for t, c := range vmtMap {
		//	if c == 0 || c == -1 {
		//		bcc[t] = 10
		//	} else {
		//		bcc[t] = c
		//	}
		//}

		for t, c := range vmtMap {
			if se.Spec.OnDemand {
				bcc[t] = c
			} else if c == 0 || c == -1 {
				bcc[t] = 10
			} else {
				bcc[t] = c
			}
		}

		dbc := &hfv1.DynamicBindConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: dbcName,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "hobbyfarm.io/v1",
						Kind:       "ScheduledEvent",
						Name:       se.Name,
						UID:        se.UID,
					},
				},
				Labels: map[string]string{
					"environment":    env.Name,
					"scheduledevent": se.Name,
				},
			},
			Spec: hfv1.DynamicBindConfigurationSpec{
				Id:                 dbcName,
				Environment:        envName,
				BaseName:           dbcRand,
				BurstCountCapacity: bcc,
				BurstCapacity:      emptyCap,
			},
		}

		if se.Spec.RestrictedBind {
			dbc.Spec.RestrictedBind = true
			dbc.Spec.RestrictedBindValue = se.Spec.RestrictedBindValue
			dbc.ObjectMeta.Labels["restrictedbind"] = "true"
			dbc.ObjectMeta.Labels["restrictedbindvalue"] = se.Spec.RestrictedBindValue
		} else {
			dbc.ObjectMeta.Labels["restrictedbind"] = "false"
		}

		_, err = s.hfClientSet.HobbyfarmV1().DynamicBindConfigurations().Create(dbc)
		if err != nil {
			glog.Errorf("error creating dynamic bind configuration %v", err)
		}
	}

	ac := &hfv1.AccessCode{
		ObjectMeta: metav1.ObjectMeta{
			Name: se.Spec.AccessCode,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "hobbyfarm.io/v1",
					Kind:       "ScheduledEvent",
					Name:       se.Name,
					UID:        se.UID,
				},
			},
			Labels: map[string]string{
				"scheduledevent": se.Name,
			},
		},
		Spec: hfv1.AccessCodeSpec{
			Code:               se.Spec.AccessCode,
			Description:        "Generated by ScheduledEventController",
			Scenarios:          se.Spec.Scenarios,
			Courses:            se.Spec.Courses,
			VirtualMachineSets: vmSets,
			Expiration:         se.Spec.EndTime,
		},
	}

	if se.Spec.RestrictedBind {
		ac.Spec.RestrictedBind = true
		ac.Spec.RestrictedBindValue = se.Spec.RestrictedBindValue
	} else {
		ac.Spec.RestrictedBind = false
	}

	ac, err := s.hfClientSet.HobbyfarmV1().AccessCodes().Create(ac)
	if err != nil {
		glog.Error(err)
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {

		seToUpdate, err := s.hfClientSet.HobbyfarmV1().ScheduledEvents().Get(se.Name, metav1.GetOptions{})

		if err != nil {
			return err
		}

		seToUpdate.Status.Provisioned = true
		seToUpdate.Status.VirtualMachineSets = vmSets
		seToUpdate.Status.Ready = false
		seToUpdate.Status.Finished = false

		_, updateErr := s.hfClientSet.HobbyfarmV1().ScheduledEvents().Update(seToUpdate)
		glog.V(4).Infof("updated result for scheduled event")

		return updateErr
	})
	if retryErr != nil {
		return retryErr
	}

	return nil

}

func (s ScheduledEventController) verifyScheduledEvent(se *hfv1.ScheduledEvent) error {
	// check the state of the vmset and mark the sevent as ready if everything is OK
	glog.V(6).Infof("ScheduledEvent %s is in provisioned status, checking status of VMSet Provisioning", se.Name)
	vmsList, err := s.hfClientSet.HobbyfarmV1().VirtualMachineSets().List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("scheduledevent=%s", se.Name),
	})
	if err != nil {
		return err
	}

	for _, vms := range vmsList.Items {
		if vms.Status.ProvisionedCount != vms.Spec.Count {
			return fmt.Errorf("scheduled event is not ready yet")
		}
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {

		seToUpdate, err := s.hfClientSet.HobbyfarmV1().ScheduledEvents().Get(se.Name, metav1.GetOptions{})

		if err != nil {
			return err
		}

		seToUpdate.Status.Provisioned = true
		seToUpdate.Status.Ready = true
		seToUpdate.Status.Finished = false

		_, updateErr := s.hfClientSet.HobbyfarmV1().ScheduledEvents().Update(seToUpdate)
		glog.V(4).Infof("updated result for scheduled event")

		return updateErr
	})
	if retryErr != nil {
		return retryErr
	}

	return nil
}

func (s *ScheduledEventController) reconcileScheduledEvent(seName string) error {
	glog.V(4).Infof("reconciling scheduled event %s", seName)

	// fetch the scheduled event
	se, err := s.hfClientSet.HobbyfarmV1().ScheduledEvents().Get(seName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// fetch the list of virtual machine templates
	templates, err := s.hfClientSet.HobbyfarmV1().VirtualMachineTemplates().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	now := time.Now()

	beginTime, err := time.Parse(time.UnixDate, se.Spec.StartTime)
	if err != nil {
		return err
	}
	endTime, err := time.Parse(time.UnixDate, se.Spec.EndTime)

	if err != nil {
		return err
	}

	// this means that the scheduled event has ended (endtime.Before(now)), but the status of the event is not finished
	// and it is still marked as active. this means we need to finish and deactivate the SE.
	if endTime.Before(now) && !se.Status.Finished && se.Status.Active {
		return s.completeScheduledEvent(se)
	}

	// if this scheduled event has begun (beginTime.Before(now)), and we haven't already provisioned
	// this SE, let's do so
	if beginTime.Before(now) && !se.Status.Provisioned && se.Status.Active {
		return s.provisionScheduledEvent(templates, se)
	}

	// the SE is ongoing and we should just verify things are good
	if beginTime.Before(now) && se.Status.Provisioned && !se.Status.Finished && se.Status.Active {
		return s.verifyScheduledEvent(se)
	}

	if endTime.Before(now) && se.Status.Finished {
		// scheduled event is finished and nothing to do
	}

	return nil
}

func calculateUsedCapacity(env *hfv1.Environment, vmsList *hfv1.VirtualMachineSetList, templates *hfv1.VirtualMachineTemplateList) (map[string]int, hfv1.CMSStruct) {
	used := hfv1.CMSStruct{}
	usedCount := map[string]int{}
	for _, vms := range vmsList.Items {
		for _, t := range templates.Items {
			if t.Name == vms.Spec.VMTemplate {
				if env.Spec.CapacityMode == hfv1.CapacityModeRaw {
					used.CPU = used.CPU + (t.Spec.Resources.CPU * vms.Spec.Count)
					used.Memory = used.Memory + (t.Spec.Resources.Memory * vms.Spec.Count)
					used.Storage = used.Storage + (t.Spec.Resources.Storage * vms.Spec.Count)
				} else if env.Spec.CapacityMode == hfv1.CapacityModeCount {
					if countKey, ok := t.Spec.CountMap[env.Spec.Provider]; ok {
						usedCount[countKey] = usedCount[countKey] + vms.Spec.Count
					} else {
						glog.Errorf("count key was not found for virtual machine template %s for provider %s", t.Name, env.Spec.Provider)
					}
				}
			}
		}
	}

	return usedCount, used
}

func calculateNeededCapacity(env *hfv1.Environment, vmtMap map[string]int, templates *hfv1.VirtualMachineTemplateList) (map[string]int, hfv1.CMSStruct) {
	needed := hfv1.CMSStruct{}

	neededCount := map[string]int{}
	for templateName, count := range vmtMap {
		for _, t := range templates.Items {
			if t.Name == templateName {
				if env.Spec.CapacityMode == hfv1.CapacityModeRaw {
					needed.CPU = needed.CPU + (t.Spec.Resources.CPU * count)
					needed.Memory = needed.Memory + (t.Spec.Resources.Memory * count)
					needed.Storage = needed.Storage + (t.Spec.Resources.Storage * count)
				} else if env.Spec.CapacityMode == hfv1.CapacityModeCount {
					if countKey, ok := t.Spec.CountMap[env.Spec.Provider]; ok {
						neededCount[countKey] = neededCount[countKey] + count
					} else {
						glog.Errorf("count key was not found for virtual machine template %s for provider %s", t.Name, env.Spec.Provider)
					}
				}
			}
		}
	}

	return neededCount, needed
}
