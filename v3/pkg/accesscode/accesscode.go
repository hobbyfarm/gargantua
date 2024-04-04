package accesscode

import (
	"context"
	"fmt"
	"time"

	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	util2 "github.com/hobbyfarm/gargantua/v3/pkg/util"

	"github.com/golang/glog"
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

func (acc AccessCodeClient) GetAccessCodesWithOTACs(codes []string) ([]hfv1.AccessCode, error) {
	otacReq, err := labels.NewRequirement(util2.OneTimeAccessCodeLabel, selection.In, codes)

	selector := labels.NewSelector()
	selector = selector.Add(*otacReq)

	// First get the oneTimeAccessCodes
	otacList, err := acc.hfClientSet.HobbyfarmV1().OneTimeAccessCodes(util2.GetReleaseNamespace()).List(acc.ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})

	if err != nil {
		return nil, fmt.Errorf("error while retrieving one time access codes %v", err)
	}

	//Append the value of onetime access codes to the list
	for _, otac := range otacList.Items {
		se, err := acc.hfClientSet.HobbyfarmV1().ScheduledEvents(util2.GetReleaseNamespace()).Get(acc.ctx, otac.Labels[util2.ScheduledEventLabel], metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error while retrieving one time access codes %v", err)
		}
		if otac.Spec.MaxDuration != "" {
			otac.Spec.MaxDuration, err = util2.GetDurationWithDays(otac.Spec.MaxDuration)

			maxDuration, err := time.ParseDuration(otac.Spec.MaxDuration)
			if err != nil {
				glog.V(4).Infof("Error parsing OTAC %s MaxDuration '%s': %s", otac.Name, otac.Spec.MaxDuration, err)
				continue
			}
			redeemedTimestamp, err := time.Parse(time.UnixDate, otac.Spec.RedeemedTimestamp)

			if err != nil {
				return nil, fmt.Errorf("error while parsing redeemedTimestamp time for OTAC %s: %v", otac.Name, err)
			}

			if time.Now().After(redeemedTimestamp.Add(maxDuration)) { // if the access code is expired don't return any scenarios
				glog.V(4).Infof("OTAC %s reached MaxDuration of %s", otac.Name, otac.Spec.MaxDuration)
				continue
			}
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

	acReq, err := labels.NewRequirement(util2.AccessCodeLabel, selection.In, codes)

	selector := labels.NewSelector()
	selector = selector.Add(*acReq)

	accessCodeList, err := acc.hfClientSet.HobbyfarmV1().AccessCodes(util2.GetReleaseNamespace()).List(acc.ctx, metav1.ListOptions{
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
