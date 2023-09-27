package accesscode

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

type AccessCodeClient struct {
	hfClientSet hfClientset.Interface
	ctx         context.Context
}

func NewAccessCodeClient(hfClientset hfClientset.Interface, ctx context.Context) (*AccessCodeClient, error) {
	acc := AccessCodeClient{}
	acc.hfClientSet = hfClientset
	acc.ctx = ctx
	return &acc, nil
}

func (acc AccessCodeClient) GetSomething(code string) error {
	return nil
}

func (acc AccessCodeClient) GetAccessCodesWithOTACs(codes []string) ([]hfv1.AccessCode, error) {
	otacReq, err := labels.NewRequirement(util.OneTimeAccessCodeLabel, selection.In, codes)

	selector := labels.NewSelector()
	selector = selector.Add(*otacReq)

	// First get the oneTimeAccessCodes
	otacList, err := acc.hfClientSet.HobbyfarmV1().OneTimeAccessCodes(util.GetReleaseNamespace()).List(acc.ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})

	if err != nil {
		return nil, fmt.Errorf("error while retrieving one time access codes %v", err)
	}

	//Append the value of onetime access codes to the list
	for _, otac := range otacList.Items {
		se, err := acc.hfClientSet.HobbyfarmV1().ScheduledEvents(util.GetReleaseNamespace()).Get(acc.ctx, otac.Labels[util.ScheduledEventLabel], metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error while retrieving one time access codes %v", err)
		}
		codes = append(codes, se.Spec.AccessCode)
	}

	accessCodes, err := acc.GetAccessCodes(codes)
	return accessCodes, err
}

func (acc AccessCodeClient) GetAccessCodes(codes []string) ([]hfv1.AccessCode, error) {
	if len(codes) == 0 {
		return nil, fmt.Errorf("code list passed in was less than 0")
	}

	acReq, err := labels.NewRequirement(util.AccessCodeLabel, selection.In, codes)

	selector := labels.NewSelector()
	selector = selector.Add(*acReq)

	accessCodeList, err := acc.hfClientSet.HobbyfarmV1().AccessCodes(util.GetReleaseNamespace()).List(acc.ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})

	if err != nil {
		return nil, fmt.Errorf("error while retrieving access codes %v", err)
	}

	var accessCodes []hfv1.AccessCode

	for _, accessCode := range accessCodeList.Items {

		if accessCode.Spec.Expiration != "" {
			expiration, err := time.Parse(time.UnixDate, accessCode.Spec.Expiration)

			if err != nil {
				return nil, fmt.Errorf("error while parsing expiration time for access code %s %v", accessCode.Name, err)
			}

			if time.Now().After(expiration) { // if the access code is expired don't return any scenarios
				glog.V(4).Infof("access code %s was expired at %s", accessCode.Name, accessCode.Spec.Expiration)
				continue
			}
		}

		accessCodes = append(accessCodes, accessCode)
	}

	return accessCodes, nil

}

func (acc AccessCodeClient) GetAccessCodeWithOTACs(code string) (hfv1.AccessCode, error) {
	if len(code) == 0 {
		return hfv1.AccessCode{}, fmt.Errorf("code was empty")
	}

	accessCodes, err := acc.GetAccessCodesWithOTACs([]string{code})

	if err != nil {
		return hfv1.AccessCode{}, fmt.Errorf("access code (%s) not found: %v", code, err)
	}

	if len(accessCodes) != 1 {
		return hfv1.AccessCode{}, fmt.Errorf("insane result found")
	}

	return accessCodes[0], nil
}

func (acc AccessCodeClient) GetAccessCode(code string) (hfv1.AccessCode, error) {
	if len(code) == 0 {
		return hfv1.AccessCode{}, fmt.Errorf("code was empty")
	}

	accessCodes, err := acc.GetAccessCodes([]string{code})

	if err != nil {
		return hfv1.AccessCode{}, fmt.Errorf("access code (%s) not found: %v", code, err)
	}

	if len(accessCodes) != 1 {
		return hfv1.AccessCode{}, fmt.Errorf("insane result found")
	}

	return accessCodes[0], nil
}

func (acc AccessCodeClient) GetScenarioIds(code string) ([]string, error) {
	var ids []string

	if len(code) == 0 {
		return ids, fmt.Errorf("code was empty")
	}

	accessCode, err := acc.GetAccessCodeWithOTACs(code)

	if err != nil {
		return ids, fmt.Errorf("error finding access code %s: %v", code, err)
	}

	return accessCode.Spec.Scenarios, nil
}

func (acc AccessCodeClient) GetCourseIds(code string) ([]string, error) {
	var ids []string

	if len(code) == 0 {
		return ids, fmt.Errorf("code was empty")
	}

	accessCode, err := acc.GetAccessCodeWithOTACs(code)

	if err != nil {
		return ids, fmt.Errorf("error finding access code %s: %v", code, err)
	}

	return accessCode.Spec.Courses, nil
}

func (acc AccessCodeClient) GetClosestAccessCode(userID string, scenarioOrCourseId string) (string, error) {
	// basically let's get all of the access codes, sort them by expiration, and start going down the list looking for access codes.

	user, err := acc.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Get(acc.ctx, userID, metav1.GetOptions{}) // @TODO: FIX THIS TO NOT DIRECTLY CALL USER

	if err != nil {
		return "", fmt.Errorf("error retrieving user: %v", err)
	}

	rawAccessCodes, err := acc.GetAccessCodesWithOTACs(user.Spec.AccessCodes)

	if err != nil {
		return "", fmt.Errorf("access codes were not found %v", err)
	}

	var accessCodes []hfv1.AccessCode
	for _, code := range rawAccessCodes {
		for _, s := range code.Spec.Scenarios {
			if s == scenarioOrCourseId {
				accessCodes = append(accessCodes, code)
				break
			}
		}

		for _, c := range code.Spec.Courses {
			if c == scenarioOrCourseId {
				accessCodes = append(accessCodes, code)
				break
			}
		}
	}

	if len(accessCodes) == 0 {
		return "", fmt.Errorf("access codes were not found for user %s with scenario or course id %s", userID, scenarioOrCourseId)
	}

	sort.Slice(accessCodes, func(i, j int) bool {
		if accessCodes[i].Spec.Expiration == "" || accessCodes[j].Spec.Expiration == "" {
			if accessCodes[i].Spec.Expiration == "" {
				return false
			}
			if accessCodes[j].Spec.Expiration == "" {
				return true
			}
		}
		iExp, err := time.Parse(time.UnixDate, accessCodes[i].Spec.Expiration)
		if err != nil {
			return false
		}
		jExp, err := time.Parse(time.UnixDate, accessCodes[j].Spec.Expiration)
		if err != nil {
			return true
		}
		return iExp.Before(jExp)
	})

	if glog.V(6) {
		var accessCodesList []string
		for _, ac := range accessCodes {
			accessCodesList = append(accessCodesList, ac.Spec.Code)
		}
		glog.Infof("Access code list was %v", accessCodesList)
	}

	return accessCodes[0].Name, nil
}
