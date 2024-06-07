package sessionservice

import (
	"context"
	"fmt"
	"time"

	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/golang/glog"
	controllers "github.com/hobbyfarm/gargantua/v3/pkg/microservices/controller"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	progresspb "github.com/hobbyfarm/gargantua/v3/protos/progress"
	sessionpb "github.com/hobbyfarm/gargantua/v3/protos/session"
	vmpb "github.com/hobbyfarm/gargantua/v3/protos/vm"
	vmclaimpb "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"
	"k8s.io/client-go/kubernetes"
)

type SessionController struct {
	controllers.DelayingWorkqueueController
	controllers.Reconciler
	internalSessioServer *GrpcSessionServer
	progressClient       progresspb.ProgressSvcClient
	vmClient             vmpb.VMSvcClient
	vmClaimClient        vmclaimpb.VMClaimSvcClient
}

func NewSessionController(
	kubeClient *kubernetes.Clientset,
	internalSessionServer *GrpcSessionServer,
	hfInformerFactory hfInformers.SharedInformerFactory,
	progressClient progresspb.ProgressSvcClient,
	vmClient vmpb.VMSvcClient,
	vmClaimClient vmclaimpb.VMClaimSvcClient,
	ctx context.Context,
) (*SessionController, error) {
	sessionInformer := hfInformerFactory.Hobbyfarm().V1().Sessions().Informer()
	delayingWorkqueueController := *controllers.NewDelayingWorkqueueController(
		ctx,
		sessionInformer,
		kubeClient,
		"session-controller",
		time.Minute*30,
		nil,
	)

	sessionController := &SessionController{
		DelayingWorkqueueController: delayingWorkqueueController,
		internalSessioServer:        internalSessionServer,
		progressClient:              progressClient,
		vmClient:                    vmClient,
		vmClaimClient:               vmClaimClient,
	}
	sessionController.SetReconciler(sessionController)
	sessionController.SetWorkScheduler(sessionController)

	return sessionController, nil
}

func (s *SessionController) Reconcile(objName string) error {
	err := s.reconcileSession(objName)

	if err != nil {
		glog.Error(err)
	}
	//s.ssWorkqueue.Forget(obj)
	glog.V(8).Infof("ss processed by session controller %v", objName)

	return nil
}

func (s *SessionController) reconcileSession(ssName string) error {
	glog.V(4).Infof("reconciling session %s", ssName)

	ss, err := s.internalSessioServer.GetSession(s.Context, &generalpb.GetRequest{
		Id:            ssName,
		LoadFromCache: true,
	})

	if err != nil {
		return err
	}

	now := time.Now()

	expires, err := time.Parse(time.UnixDate, ss.GetStatus().GetExpirationTime())

	if err != nil {
		return err
	}

	timeUntilExpires := expires.Sub(now)

	// clean up sessions if they are finished
	if ss.GetStatus().GetFinished() {
		glog.V(6).Infof("deleted finished session  %s", ss.GetId())

		// now that the vmclaims are deleted, go ahead and delete the session
		_, err = s.internalSessioServer.DeleteSession(s.Context, &generalpb.ResourceId{Id: ss.GetId()})

		if err != nil {
			return fmt.Errorf("error deleting session %s: %v", ss.GetId(), err)
		}

		glog.V(6).Infof("deleted old session %s", ss.GetId())

		s.FinishProgress(ss.GetId(), ss.GetUser())

		return nil
	}

	if expires.Before(now) && !ss.GetStatus().GetFinished() {
		// we need to set the session to finished and delete the vm's
		if ss.Status.Active && ss.Status.Paused && ss.Status.PausedTime != "" {
			pausedExpiration, err := time.Parse(time.UnixDate, ss.Status.PausedTime)
			if err != nil {
				glog.Error(err)
			}

			if pausedExpiration.After(now) {
				glog.V(4).Infof("Session %s was paused, and the pause expiration is after now, skipping clean up.", ss.GetId())
				return nil
			}

			glog.V(4).Infof("Session %s was paused, but the pause expiration was before now, so cleaning up.", ss.GetId())
		}
		for _, vmc := range ss.GetVmClaim() {
			vmcObj, err := s.vmClaimClient.GetVMClaim(s.Context, &generalpb.GetRequest{
				Id:            vmc,
				LoadFromCache: true,
			})

			if err != nil {
				break
			}

			for _, vm := range vmcObj.GetVms() {
				if len(vm.GetVmId()) == 0 {
					// VM was not even provisioned / assigned yet.
					continue
				}
				taintErr := s.taintVM(vm.GetVmId())
				if taintErr != nil {
					glog.Error(taintErr)
				}
			}

			taintErr := s.taintVMC(vmcObj.GetId())
			if taintErr != nil {
				glog.Error(taintErr)
			}
		}

		_, err = s.internalSessioServer.UpdateSessionStatus(s.Context, &sessionpb.UpdateSessionStatusRequest{
			Id:       ssName,
			Finished: wrapperspb.Bool(true),
			Active:   wrapperspb.Bool(false),
		})
		glog.V(4).Infof("updated result for session")
		if err != nil {
			return err
		}
	} else if expires.Before(now) && ss.GetStatus().GetFinished() {
		glog.V(8).Infof("session %s is finished and expired before now", ssName)
	} else {
		glog.V(8).Infof("adding session %s to workqueue after %s", ssName, timeUntilExpires.String())
		ssWorkqueue, err := s.GetDelayingWorkqueue()
		if err != nil {
			return fmt.Errorf("unable to requeue session: %v", err)
		}
		ssWorkqueue.AddAfter(ssName, timeUntilExpires)
		glog.V(8).Infof("added session %s to workqueue", ssName)
	}

	return nil
}

func (s *SessionController) taintVM(vmName string) error {
	glog.V(5).Infof("tainting VM %s", vmName)
	_, err := s.vmClient.UpdateVMStatus(s.Context, &vmpb.UpdateVMStatusRequest{
		Id:      vmName,
		Tainted: wrapperspb.Bool(true),
	})
	glog.V(4).Infof("updated result for vm")

	return err
}

func (s *SessionController) taintVMC(vmcName string) error {
	glog.V(5).Infof("tainting VMC %s", vmcName)
	_, err := s.vmClaimClient.UpdateVMClaimStatus(s.Context, &vmclaimpb.UpdateVMClaimStatusRequest{
		Id:      vmcName,
		Tainted: wrapperspb.Bool(true),
	})
	glog.V(4).Infof("updated result for vmc")

	return err
}

func (s *SessionController) FinishProgress(sessionId string, userId string) {
	now := time.Now()

	progressList, err := s.progressClient.ListProgress(s.Context, &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=%s,finished=false", hflabels.SessionLabel, sessionId, hflabels.UserLabel, userId),
	})

	if err != nil {
		glog.Errorf("error while retrieving progress %v", err)
		return
	}

	for _, p := range progressList.GetProgresses() {
		_, err = s.progressClient.UpdateProgress(s.Context, &progresspb.UpdateProgressRequest{
			Id:         p.GetId(),
			LastUpdate: now.Format(time.UnixDate),
			Finished:   "true",
		})
		glog.V(4).Infof("updated progress with ID %s", p.GetId())
		if err != nil {
			glog.Errorf("error finishing progress %v", err)
			return
		}
	}
}
