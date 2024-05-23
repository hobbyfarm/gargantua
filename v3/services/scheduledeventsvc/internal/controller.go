package eventservice

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	settingUtil "github.com/hobbyfarm/gargantua/v3/pkg/setting"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/golang/glog"
	controllers "github.com/hobbyfarm/gargantua/v3/pkg/microservices/controller"
	accesscodepb "github.com/hobbyfarm/gargantua/v3/protos/accesscode"
	dbconfigpb "github.com/hobbyfarm/gargantua/v3/protos/dbconfig"
	environmentpb "github.com/hobbyfarm/gargantua/v3/protos/environment"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	progresspb "github.com/hobbyfarm/gargantua/v3/protos/progress"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
	sessionpb "github.com/hobbyfarm/gargantua/v3/protos/session"
	settingpb "github.com/hobbyfarm/gargantua/v3/protos/setting"
	vmsetpb "github.com/hobbyfarm/gargantua/v3/protos/vmset"
	vmtemplatepb "github.com/hobbyfarm/gargantua/v3/protos/vmtemplate"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
)

type ScheduledEventController struct {
	controllers.RateLimitingWorkqueueController
	controllers.Reconciler
	internalScheduledEventServer *GrpcScheduledEventServer
	accessCodeClient             accesscodepb.AccessCodeSvcClient
	sessionClient                sessionpb.SessionSvcClient
	progressClient               progresspb.ProgressSvcClient
	environmentClient            environmentpb.EnvironmentSvcClient
	dbConfigClient               dbconfigpb.DynamicBindConfigSvcClient
	vmSetClient                  vmsetpb.VMSetSvcClient
	vmTemplateClient             vmtemplatepb.VMTemplateSvcClient
	settingClient                settingpb.SettingSvcClient
}

var baseNameScheduledPrefix string

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
}

func NewScheduledEventController(
	kubeClient *kubernetes.Clientset,
	internalScheduledEventServer *GrpcScheduledEventServer,
	hfInformerFactory hfInformers.SharedInformerFactory,
	acClient accesscodepb.AccessCodeSvcClient,
	dbConfigClient dbconfigpb.DynamicBindConfigSvcClient,
	environmentClient environmentpb.EnvironmentSvcClient,
	progressClient progresspb.ProgressSvcClient,
	sessionClient sessionpb.SessionSvcClient,
	vmSetClient vmsetpb.VMSetSvcClient,
	vmTemplateClient vmtemplatepb.VMTemplateSvcClient,
	settingClient settingpb.SettingSvcClient,
	ctx context.Context,
) (*ScheduledEventController, error) {
	scheduledEventInformer := hfInformerFactory.Hobbyfarm().V1().ScheduledEvents().Informer()
	rateLimitingWorkqueueController := *controllers.NewRateLimitingWorkqueueController(
		ctx,
		scheduledEventInformer,
		kubeClient,
		"scheduledevent-controller",
		time.Minute*30,
		workqueue.NewItemExponentialFailureRateLimiter(ScheduledEventBaseDelay, ScheduledEventMaxDelay),
	)

	scheduledEventController := &ScheduledEventController{
		RateLimitingWorkqueueController: rateLimitingWorkqueueController,
		internalScheduledEventServer:    internalScheduledEventServer,
		accessCodeClient:                acClient,
		dbConfigClient:                  dbConfigClient,
		environmentClient:               environmentClient,
		progressClient:                  progressClient,
		sessionClient:                   sessionClient,
		vmSetClient:                     vmSetClient,
		vmTemplateClient:                vmTemplateClient,
		settingClient:                   settingClient,
	}
	scheduledEventController.SetReconciler(scheduledEventController)

	return scheduledEventController, nil
}

func (sc *ScheduledEventController) Reconcile(objName string) error {
	err := sc.reconcileScheduledEvent(objName)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			sc.GetWorkqueue().Add(objName)
			return err
		}
	}
	//s.seWorkqueue.Forget(obj)
	glog.V(8).Infof("se %s processed by scheduled event controller", objName)

	return nil
}

func (sc *ScheduledEventController) completeScheduledEvent(se *scheduledeventpb.ScheduledEvent) error {
	glog.V(6).Infof("ScheduledEvent %s is done, deleting corresponding VMSets and marking as finished", se.GetId())
	// scheduled event is finished, we need to set the scheduled event to finished and delete the vm's

	err := sc.deleteVMSetsFromScheduledEvent(se)

	if err != nil {
		return err
	}

	err = sc.finishSessionsFromScheduledEvent(se)

	if err != nil {
		return err
	}

	// update the scheduled event and set the various flags accordingly (provisioned, ready, finished)
	_, err = sc.internalScheduledEventServer.UpdateScheduledEventStatus(sc.Context, &scheduledeventpb.UpdateScheduledEventStatusRequest{
		Id:          se.GetId(),
		Provisioned: wrapperspb.Bool(true),
		Ready:       wrapperspb.Bool(false),
		Finished:    wrapperspb.Bool(true),
	})

	if err != nil {
		return err
	}

	return nil // break (return) here because we're done with this SE.
}

func (sc *ScheduledEventController) deleteScheduledEvent(se *scheduledeventpb.ScheduledEvent) error {
	glog.V(6).Infof("ScheduledEvent %s is done and retention time is over, deleting SE finally", se.GetId())

	if !se.GetStatus().GetFinished() {
		return fmt.Errorf("error attempting to delete SE that is not finished")
	}

	// Delete Progress
	err := sc.deleteProgressFromScheduledEvent(se)
	if err != nil {
		return err
	}

	_, err = sc.internalScheduledEventServer.DeleteScheduledEvent(sc.Context, &generalpb.ResourceId{Id: se.GetId()})
	if err != nil {
		return err
	}
	return nil // break (return) here because we're done with this SE.
}

func (sc *ScheduledEventController) deleteVMSetsFromScheduledEvent(se *scheduledeventpb.ScheduledEvent) error {
	// for each vmset that belongs to this to-be-stopped scheduled event, delete that vmset
	_, err := sc.vmSetClient.DeleteCollectionVMSet(sc.Context, &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", hflabels.ScheduledEventLabel, se.Name),
	})
	return err
}

func (sc *ScheduledEventController) deleteProgressFromScheduledEvent(se *scheduledeventpb.ScheduledEvent) error {
	// for each vmset that belongs to this to-be-stopped scheduled event, delete that vmset
	_, err := sc.progressClient.DeleteCollectionProgress(sc.Context, &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", hflabels.ScheduledEventLabel, se.GetId()),
	})
	if err != nil {
		return err
	}

	return nil
}

func (sc *ScheduledEventController) deleteAccessCode(seId string) error {
	// delete the access code for the corresponding ScheduledEvent
	_, err := sc.accessCodeClient.DeleteCollectionAc(sc.Context, &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", hflabels.ScheduledEventLabel, seId),
	})
	return err
}

func (sc *ScheduledEventController) finishSessionsFromScheduledEvent(se *scheduledeventpb.ScheduledEvent) error {
	// get a list of sessions for the user
	sessionList, err := sc.sessionClient.ListSession(sc.Context, &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", hflabels.AccessCodeLabel, se.GetAccessCode()),
	})

	if err != nil {
		glog.Errorf("error listing sessions which are supposed to expire %v", err)
		return fmt.Errorf("error attempting to update")
	}

	now := time.Now().Format(time.UnixDate)

	for _, session := range sessionList.GetSessions() {
		_, err = sc.sessionClient.UpdateSessionStatus(sc.Context, &sessionpb.UpdateSessionStatusRequest{
			Id:             session.GetId(),
			ExpirationTime: now,
			Active:         wrapperspb.Bool(false),
		})

		if err != nil {
			glog.Errorf("error updating session status %v", err)
			return fmt.Errorf("error attempting to update")
		}
	}
	return nil
}

func (sc *ScheduledEventController) provisionScheduledEvent(se *scheduledeventpb.ScheduledEvent) error {
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
	for envId, vmtMap := range se.GetRequiredVms() {
		// TODO: actually check for capacity usage
		// get the environment we're provisioning into (envName)
		// env, err := sc.environmentClient.GetEnvironment(sc.Context, &general.GetRequest{Id: envId})
		// if err != nil {
		// 	glog.Errorf("error retreiving environment %s", err.Error())
		// 	return err
		// }

		// create virtualmachinesets if not on demand
		if !se.GetOnDemand() {
			for templateName, count := range vmtMap.GetVmTemplateCounts() {
				if count > 0 { // only setup vmsets if > 0 VMs are requested, and they aren't ondemand
					//1. Find existing VMset that match this SE and the current environment
					existingVMSetsList, err := sc.vmSetClient.ListVMSet(sc.Context, &generalpb.ListOptions{
						LabelSelector: fmt.Sprintf("%s=%s,%s=%s,virtualmachinetemplate.hobbyfarm.io/%s=true", hflabels.ScheduledEventLabel, se.GetId(), hflabels.EnvironmentLabel, envId, templateName),
					})

					if err != nil || len(existingVMSetsList.GetVmsets()) == 0 { // create new vmset if no existing one was found
						vmsRand := fmt.Sprintf("%s-%08x", baseNameScheduledPrefix, rand.Uint32())
						vmsId := strings.Join([]string{"se", se.Name, "vms", vmsRand}, "-")
						vmSets = append(vmSets, vmsId)
						_, err = sc.vmSetClient.CreateVMSet(sc.Context, &vmsetpb.CreateVMSetRequest{
							Id:                  vmsId,
							Count:               count,
							Environment:         envId,
							VmTemplate:          templateName,
							BaseName:            vmsRand,
							RestrictedBind:      se.GetRestrictedBind(),
							RestrictedBindValue: se.GetRestrictedBindValue(),
							SeName:              se.GetId(),
							SeUid:               se.GetUid(),
							Labels: map[string]string{
								hflabels.EnvironmentLabel:    envId,
								hflabels.ScheduledEventLabel: se.GetId(),
								fmt.Sprintf("virtualmachinetemplate.hobbyfarm.io/%s", templateName): "true",
							},
						})
						if err != nil {
							glog.Error(err)
							return err
						}
					} else { // update existing vmset
						// Todo support multiple VM Sets
						existingVMSet := existingVMSetsList.GetVmsets()[0]
						vmSets = append(vmSets, existingVMSet.GetId())

						_, err = sc.vmSetClient.UpdateVMSet(sc.Context, &vmsetpb.UpdateVMSetRequest{
							Id:             existingVMSet.GetId(),
							Count:          wrapperspb.UInt32(count),
							RestrictedBind: wrapperspb.Bool(se.GetRestrictedBind()),
							Environment:    envId,
						})
						if err != nil {
							glog.Errorf("error updating vmset config %s", err.Error())
							return err
						}
					}
				}
			}
		}

		// Delete existing DynamicBindConfigurations
		_, err := sc.dbConfigClient.DeleteCollectionDynamicBindConfig(sc.Context, &generalpb.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s,environment=%s", hflabels.ScheduledEventLabel, se.GetId(), envId),
		})
		if err != nil {
			return err
		}

		// create the dynamic bind configurations
		_, err = sc.dbConfigClient.CreateDynamicBindConfig(sc.Context, &dbconfigpb.CreateDynamicBindConfigRequest{
			SeName:              se.GetId(),
			SeUid:               se.GetUid(),
			EnvName:             envId,
			BurstCountCapacity:  vmtMap.GetVmTemplateCounts(),
			RestrictedBind:      se.GetRestrictedBind(),
			RestrictedBindValue: se.GetRestrictedBindValue(),
		})
		if err != nil {
			glog.Errorf("error creating dynamic bind configuration %v", err)
		}
	}

	// Delete AccessCode if it exists
	_, err := sc.accessCodeClient.GetAc(sc.Context, &generalpb.GetRequest{
		Id: se.GetAccessCode(),
	})
	if err == nil {
		err = sc.deleteAccessCode(se.GetId())
		if err != nil {
			return err
		}
	}

	err = sc.createAccessCode(se)
	if err != nil {
		return err
	}

	_, err = sc.internalScheduledEventServer.UpdateScheduledEventStatus(sc.Context, &scheduledeventpb.UpdateScheduledEventStatusRequest{
		Id: se.GetId(),
		Vmsets: &scheduledeventpb.VMSetsWrapper{
			Value: vmSets,
		},
		Provisioned: wrapperspb.Bool(true),
		Ready:       wrapperspb.Bool(false),
		Finished:    wrapperspb.Bool(false),
	})
	glog.V(4).Infof("updated result for scheduled event %s", se.GetId())
	if err != nil {
		return err
	}

	return nil
}

func (sc *ScheduledEventController) createAccessCode(se *scheduledeventpb.ScheduledEvent) error {
	_, err := sc.accessCodeClient.CreateAc(sc.Context, &accesscodepb.CreateAcRequest{
		AcName:              se.GetAccessCode(),
		SeName:              se.GetId(),
		SeUid:               se.GetUid(),
		Description:         "Generated by ScheduledEventController",
		Scenarios:           se.GetScenarios(),
		Courses:             se.GetCourses(),
		Expiration:          se.GetEndTime(),
		RestrictedBind:      se.GetRestrictedBind(),
		RestrictedBindValue: se.GetRestrictedBindValue(),
		Printable:           se.GetPrintable(),
	})
	if err != nil {
		return err
	}

	return nil
}

func (sc *ScheduledEventController) verifyScheduledEvent(se *scheduledeventpb.ScheduledEvent) error {
	// check the state of the vmset and mark the sevent as ready if everything is OK
	glog.V(6).Infof("ScheduledEvent %s is in provisioned status, checking status of VMSet Provisioning", se.Name)
	vmsList, err := sc.vmSetClient.ListVMSet(sc.Context, &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", hflabels.ScheduledEventLabel, se.Name),
	})
	if err != nil {
		return err
	}

	for _, vms := range vmsList.GetVmsets() {
		if vms.GetStatus().GetProvisioned() < vms.GetCount() {
			return fmt.Errorf("scheduled event is not ready yet")
		}
	}

	// Validate AccessCode existence and has label set
	ac, err := sc.accessCodeClient.GetAc(sc.Context, &generalpb.GetRequest{Id: se.GetAccessCode()})
	if err != nil {
		err = sc.createAccessCode(se)

		if err != nil {
			return err
		}

	} else if ac.GetLabels()[hflabels.AccessCodeLabel] != ac.GetId() {
		err = sc.deleteAccessCode(se.GetId())
		if err != nil {
			return err
		}

		err = sc.createAccessCode(se)

		if err != nil {
			return err
		}
	}

	_, err = sc.internalScheduledEventServer.UpdateScheduledEventStatus(sc.Context, &scheduledeventpb.UpdateScheduledEventStatusRequest{
		Id:    se.GetId(),
		Ready: wrapperspb.Bool(true),
	})
	if err != nil {
		return err
	}

	return nil
}

func (sc *ScheduledEventController) reconcileScheduledEvent(seName string) error {
	glog.V(4).Infof("reconciling scheduled event %s", seName)

	// fetch the scheduled event

	se, err := sc.internalScheduledEventServer.GetScheduledEvent(sc.Context, &generalpb.GetRequest{Id: seName})
	if err != nil {
		return err
	}

	now := time.Now()

	beginTime, err := time.Parse(time.UnixDate, se.GetStartTime())
	if err != nil {
		return err
	}
	endTime, err := time.Parse(time.UnixDate, se.GetEndTime())

	if err != nil {
		return err
	}

	// this means that the scheduled event has ended (endtime.Before(now)), but the status of the event is not finished
	// and it is still marked as active. this means we need to finish and deactivate the SE.
	if endTime.Before(now) && !se.GetStatus().GetFinished() && se.GetStatus().GetActive() {
		return sc.completeScheduledEvent(se)
	}

	// if this scheduled event has begun (beginTime.Before(now)), and we haven't already provisioned
	// this SE, let's do so
	if beginTime.Before(now) && !se.GetStatus().GetProvisioned() && se.GetStatus().GetActive() {
		return sc.provisionScheduledEvent(se)
	}

	// the SE is ongoing and we should just verify things are good
	if beginTime.Before(now) && se.Status.Provisioned && !se.Status.Finished && se.Status.Active {
		return sc.verifyScheduledEvent(se)
	}

	if endTime.Before(now) && se.Status.Finished {
		// scheduled event is finished and nothing to do
		setting, err := sc.settingClient.GetSettingValue(sc.Context, &generalpb.ResourceId{Id: string(settingUtil.ScheduledEventRetentionTime)})

		if set, ok := setting.GetValue().(*settingpb.SettingValue_Int64Value); err != nil || !ok || setting == nil {
			return fmt.Errorf("error retreiving retention Time setting")
		} else {
			retentionTime := endTime.Add(time.Hour * time.Duration(set.Int64Value))
			if retentionTime.Before(now) {
				// Really finish the ScheduledEvent
				return sc.deleteScheduledEvent(se)
			}
		}
	}

	// The ScheduledEvent is set to OnDemand but still has VMSets
	if se.GetOnDemand() && len(se.GetStatus().GetVmsets()) > 0 {
		vmSets := []string{}
		_, err := sc.internalScheduledEventServer.UpdateScheduledEventStatus(sc.Context, &scheduledeventpb.UpdateScheduledEventStatusRequest{
			Id:     se.GetId(),
			Vmsets: &scheduledeventpb.VMSetsWrapper{Value: vmSets},
		})
		if err != nil {
			return err
		}
		err = sc.deleteVMSetsFromScheduledEvent(se)
		return err
	}

	return nil
}

// @TODO: Integrate this function if it should be used or remove it if not.
func calculateUsedCapacity(env *hfv1.Environment, vmsList *hfv1.VirtualMachineSetList, templates *hfv1.VirtualMachineTemplateList) map[string]int {
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
