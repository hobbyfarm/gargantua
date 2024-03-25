package authserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/hobbyfarm/gargantua/v3/pkg/accesscode"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	util2 "github.com/hobbyfarm/gargantua/v3/pkg/util"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/v3/protos/authn"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

type AuthServer struct {
	authClient       authn.AuthNClient
	hfClientSet      hfClientset.Interface
	accessCodeClient *accesscode.AccessCodeClient
	ctx              context.Context
}

func NewAuthServer(authClient authn.AuthNClient, hfClientSet hfClientset.Interface, ctx context.Context, acClient *accesscode.AccessCodeClient) (AuthServer, error) {
	a := AuthServer{}
	a.authClient = authClient
	a.hfClientSet = hfClientSet
	a.ctx = ctx
	a.accessCodeClient = acClient
	return a, nil
}

func (a AuthServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/auth/scheduledevents", a.ListScheduledEventsFunc).Methods("GET")
	glog.V(2).Infof("set up route")
}

type PreparedScheduledEvent struct {
	Id          string `json:"id"`
	Description string `json:"description"`
	Name        string `json:"name"`
	EndDate     string `json:"end_timestamp"`
}

func (a AuthServer) ListScheduledEventsFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, a.authClient)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list suitable scheduledevents")
		return
	}

	// This holds a map of AC -> SE
	accessCodeScheduledEvent := make(map[string]PreparedScheduledEvent)

	// First we add ScheduledEvents based on OneTimeAccessCodes
	otacReq, _ := labels.NewRequirement(hflabels.OneTimeAccessCodeLabel, selection.In, user.GetAccessCodes())
	selector := labels.NewSelector()
	selector = selector.Add(*otacReq)

	otacList, err := a.hfClientSet.HobbyfarmV1().OneTimeAccessCodes(util2.GetReleaseNamespace()).List(a.ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})

	if err == nil {
		for _, otac := range otacList.Items {
			se, err := a.hfClientSet.HobbyfarmV1().ScheduledEvents(util2.GetReleaseNamespace()).Get(a.ctx, otac.Labels[hflabels.ScheduledEventLabel], metav1.GetOptions{})
			if err != nil {
				continue
			}
			endTime := se.Spec.EndTime

			// If OTAC specifies a max Duration we need to calculate the EndTime correctly
			if otac.Spec.MaxDuration != "" {
				otacEndTime, err := time.Parse(time.UnixDate, otac.Spec.RedeemedTimestamp)
				if err != nil {
					continue
				}
				otacDurationWithDays, err := util2.GetDurationWithDays(otac.Spec.MaxDuration)
				otacDuration, err := time.ParseDuration(otacDurationWithDays)
				if err != nil {
					continue
				}
				otacEndTime = otacEndTime.Add(otacDuration)
				endTime = otacEndTime.Format(time.UnixDate)
			}

			accessCodeScheduledEvent[otac.Name] = PreparedScheduledEvent{se.Name, se.Spec.Description, se.Spec.Name, endTime}
		}
	}

	// Afterwards we retreive the normal AccessCodes
	accessCodes, _ := a.accessCodeClient.GetAccessCodes(user.GetAccessCodes())

	//Getting single SEs should be faster than listing all of them and iterating them in O(n^2), in most cases users only have a hand full of accessCodes.
	for _, ac := range accessCodes {
		se, err := a.hfClientSet.HobbyfarmV1().ScheduledEvents(util2.GetReleaseNamespace()).Get(a.ctx, ac.Labels[hflabels.ScheduledEventLabel], metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			continue
		}
		accessCodeScheduledEvent[ac.Spec.Code] = PreparedScheduledEvent{se.Name, se.Spec.Description, se.Spec.Name, se.Spec.EndTime}
	}

	encodedMap, err := json.Marshal(accessCodeScheduledEvent)
	if err != nil {
		glog.Error(err)
	}
	util2.ReturnHTTPContent(w, r, 200, "success", encodedMap)
}
