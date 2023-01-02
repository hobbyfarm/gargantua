package scheduledevent

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	"github.com/hobbyfarm/gargantua/pkg/util"
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
	ctx         context.Context
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

func NewScheduledEventController(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context) (*ScheduledEventController, error) {
	seController := ScheduledEventController{}
	seController.ctx = ctx
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

	err = s.finishSessionsFromScheduledEvent(se)

	if err != nil {
		return err
	}

	err = s.deleteProgressFromScheduledEvent(se)

	if err != nil {
		return err
	}

	// update the scheduled event and set the various flags accordingly (provisioned, ready, finished)
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		seToUpdate, err := s.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).Get(s.ctx, se.Name, metav1.GetOptions{})

		if err != nil {
			return err
		}

		seToUpdate.Status.Provisioned = true
		seToUpdate.Status.Ready = false
		seToUpdate.Status.Finished = true

		_, updateErr := s.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).UpdateStatus(s.ctx, seToUpdate, metav1.UpdateOptions{})
		glog.V(4).Infof("updated result for scheduled event")

		return updateErr
	})

	if retryErr != nil {
		return retryErr
	}

	return nil // break (return) here because we're done with this SE.
}

func (s ScheduledEventController) deleteVMSetsFromScheduledEvent(se *hfv1.ScheduledEvent) error {
	// for each vmset that belongs to this to-be-stopped scheduled event, delete that vmset
	err := s.hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()).DeleteCollection(s.ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", util.ScheduledEventLabel, se.Name),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s ScheduledEventController) deleteProgressFromScheduledEvent(se *hfv1.ScheduledEvent) error {
	// for each vmset that belongs to this to-be-stopped scheduled event, delete that vmset
	err := s.hfClientSet.HobbyfarmV1().Progresses(util.GetReleaseNamespace()).DeleteCollection(s.ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", util.ScheduledEventLabel, se.Name),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s ScheduledEventController) finishSessionsFromScheduledEvent(se *hfv1.ScheduledEvent) error {
	// get a list of sessions for the user
	sessionList, err := s.hfClientSet.HobbyfarmV1().Sessions(util.GetReleaseNamespace()).List(s.ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", util.AccessCodeLabel, se.Spec.AccessCode),
	})

	now := time.Now().Format(time.UnixDate)

	for _, session := range sessionList.Items {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			result, getErr := s.hfClientSet.HobbyfarmV1().Sessions(util.GetReleaseNamespace()).Get(s.ctx, session.Spec.Id, metav1.GetOptions{})
			if getErr != nil {
				return fmt.Errorf("error retrieving latest version of session %s: %v", session.Spec.Id, getErr)
			}

			result.Status.ExpirationTime = now
			result.Status.Active = false
			result.Status.Finished = false

			_, updateErr := s.hfClientSet.HobbyfarmV1().Sessions(util.GetReleaseNamespace()).UpdateStatus(s.ctx, result, metav1.UpdateOptions{})
			glog.V(4).Infof("updated result for session")

			return updateErr
		})

		if retryErr != nil {
			glog.Errorf("error updating session %v", err)
			return fmt.Errorf("error attempting to update")
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
		env, err := s.hfClientSet.HobbyfarmV1().Environments(util.GetReleaseNamespace()).Get(s.ctx, envName, metav1.GetOptions{})
		if err != nil {
			glog.Errorf("error retreiving environment %s", err.Error())
			return err
		}

		// TODO: actually check for capacity usage

		// create virtualmachinesets if not on demand
		if !se.Spec.OnDemand {
			for templateName, count := range vmtMap {
				if count > 0 { // only setup vmsets if >0 VMs are requested, and they aren't ondemand
					//1. Find existing VMset that match this SE and the current environment
					existingVMSets, err := s.hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()).List(s.ctx, metav1.ListOptions{
						LabelSelector: fmt.Sprintf("%s=%s,%s=%s,virtualmachinetemplate.hobbyfarm.io/%s=true", util.ScheduledEventLabel, se.Name, util.EnvironmentLabel, envName, templateName),
					})
	
					if err != nil || len(existingVMSets.Items) == 0 { // create new vmset if no existing one was found
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
									util.EnvironmentLabel:    env.Name,
									util.ScheduledEventLabel: se.Name,
									fmt.Sprintf("virtualmachinetemplate.hobbyfarm.io/%s", templateName): "true",
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
						_, err = s.hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()).Create(s.ctx, vms, metav1.CreateOptions{})
						if err != nil {
							glog.Error(err)
							return err
						}
					} else { // update existing vmset
						// Todo support multiple VM Sets
						existingVMSet := existingVMSets.Items[0]
						vmSets = append(vmSets, existingVMSet.Name)
	
						existingVMSet.Labels[util.EnvironmentLabel] = env.Name
						existingVMSet.Spec.Count = count
						if se.Spec.RestrictedBind {
							existingVMSet.Spec.RestrictedBind = true
						} else {
							existingVMSet.Spec.RestrictedBind = false
						}
						_, err = s.hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()).Update(s.ctx, &existingVMSet, metav1.UpdateOptions{})
						if err != nil {
							glog.Errorf("error updating vmset config %s", err.Error())
							return err
						}
					}
				}
			}
		}
		
		// create the dynamic bind configurations
		dbcRand := fmt.Sprintf("%s-%08x", baseNameDynamicPrefix, rand.Uint32())
		dbcName := strings.Join([]string{"se", se.Name, "dbc", dbcRand}, "-")

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
					util.EnvironmentLabel:       env.Name,
					util.ScheduledEventLabel: se.Name,
				},
			},
			Spec: hfv1.DynamicBindConfigurationSpec{
				Id:                 dbcName,
				Environment:        envName,
				BaseName:           dbcRand,
				BurstCountCapacity: vmtMap,
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

		_, err = s.hfClientSet.HobbyfarmV1().DynamicBindConfigurations(util.GetReleaseNamespace()).Create(s.ctx, dbc, metav1.CreateOptions{})
		if err != nil {
			glog.Errorf("error creating dynamic bind configuration %v", err)
		}
	}

	err := s.createAccessCode(se)
	if err != nil {
		return err
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {

		seToUpdate, err := s.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).Get(s.ctx, se.Name, metav1.GetOptions{})

		if err != nil {
			return err
		}

		seToUpdate.Status.Provisioned = true
		seToUpdate.Status.VirtualMachineSets = vmSets
		seToUpdate.Status.Ready = false
		seToUpdate.Status.Finished = false

		_, updateErr := s.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).UpdateStatus(s.ctx, seToUpdate, metav1.UpdateOptions{})
		glog.V(4).Infof("updated result for scheduled event")

		return updateErr
	})
	if retryErr != nil {
		return retryErr
	}

	return nil
}

func (s ScheduledEventController) createAccessCode(se *hfv1.ScheduledEvent) error {
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
				util.ScheduledEventLabel: se.Name,
				util.AccessCodeLabel: se.Spec.AccessCode,
			},
		},
		Spec: hfv1.AccessCodeSpec{
			Code:               se.Spec.AccessCode,
			Description:        "Generated by ScheduledEventController",
			Scenarios:          se.Spec.Scenarios,
			Courses:            se.Spec.Courses,
			Expiration:         se.Spec.EndTime,
		},
	}

	if se.Spec.RestrictedBind {
		ac.Spec.RestrictedBind = true
		ac.Spec.RestrictedBindValue = se.Spec.RestrictedBindValue
	} else {
		ac.Spec.RestrictedBind = false
	}

	if se.Spec.Printable {
		ac.Spec.Printable = true
	} else {
		ac.Spec.Printable = false
	}


	ac, err := s.hfClientSet.HobbyfarmV1().AccessCodes(util.GetReleaseNamespace()).Create(s.ctx, ac, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (s ScheduledEventController) verifyScheduledEvent(se *hfv1.ScheduledEvent) error {
	// check the state of the vmset and mark the sevent as ready if everything is OK
	glog.V(6).Infof("ScheduledEvent %s is in provisioned status, checking status of VMSet Provisioning", se.Name)
	vmsList, err := s.hfClientSet.HobbyfarmV1().VirtualMachineSets(util.GetReleaseNamespace()).List(s.ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", util.ScheduledEventLabel, se.Name),
	})
	if err != nil {
		return err
	}

	for _, vms := range vmsList.Items {
		if vms.Status.ProvisionedCount < vms.Spec.Count {
			return fmt.Errorf("scheduled event is not ready yet")
		}
	}

	// Validate AccessCode existence and has label set
	ac, err := s.hfClientSet.HobbyfarmV1().AccessCodes(util.GetReleaseNamespace()).Get(s.ctx, se.Spec.AccessCode, metav1.GetOptions{})
	if err != nil {
		err = s.createAccessCode(se)

		if err != nil {
			return err
		}

	} else if ac.Labels[util.AccessCodeLabel] != ac.Spec.Code {
		err = s.hfClientSet.HobbyfarmV1().AccessCodes(util.GetReleaseNamespace()).DeleteCollection(s.ctx, metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", util.ScheduledEventLabel, se.Name),
		})
		if err != nil {
			return err
		}

		err = s.createAccessCode(se)

		if err != nil {
			return err
		}
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {

		seToUpdate, err := s.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).Get(s.ctx, se.Name, metav1.GetOptions{})

		if err != nil {
			return err
		}

		seToUpdate.Status.Provisioned = true
		seToUpdate.Status.Ready = true
		seToUpdate.Status.Finished = false

		_, updateErr := s.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).UpdateStatus(s.ctx, seToUpdate, metav1.UpdateOptions{})
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
	se, err := s.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).Get(s.ctx, seName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// fetch the list of virtual machine templates
	templates, err := s.hfClientSet.HobbyfarmV1().VirtualMachineTemplates(util.GetReleaseNamespace()).List(s.ctx, metav1.ListOptions{})
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

	// The ScheduledEvent is set to OnDemand but still has VMSets
	if (se.Spec.OnDemand && len(se.Status.VirtualMachineSets) > 0){
		vmSets := []string{}
		se.Status.VirtualMachineSets = vmSets
		_, updateErr := s.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).UpdateStatus(s.ctx, se, metav1.UpdateOptions{})
		s.deleteVMSetsFromScheduledEvent(se)
		return updateErr
	}

	return nil
}

func calculateUsedCapacity(env *hfv1.Environment, vmsList *hfv1.VirtualMachineSetList, templates *hfv1.VirtualMachineTemplateList) (map[string]int) {
	usedCount := map[string]int{}
	for _, vms := range vmsList.Items {
		for _, t := range templates.Items {
			if t.Name == vms.Spec.VMTemplate {
				usedCount[t.Name] = usedCount[t.Name] + vms.Spec.Count
			}
		}
	}
	return usedCount
}