package accesscode

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
	"time"
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

func (acc AccessCodeClient) GetAccessCodes(codes []string) ([]hfv1.AccessCode, error) {
	if len(codes) == 0 {
		return nil, fmt.Errorf("code list passed in was less than 0")
	}

	accessCodeList, err := acc.hfClientSet.HobbyfarmV1().AccessCodes(util.GetReleaseNamespace()).List(acc.ctx, metav1.ListOptions{})

	if err != nil {
		return nil, fmt.Errorf("error while retrieving access codes %v", err)
	}

	var accessCodes []hfv1.AccessCode

	for _, code := range codes {
		found := false
		var accessCode hfv1.AccessCode
		for _, ac := range accessCodeList.Items {
			if ac.Spec.Code == code {
				found = true
				accessCode = ac
				break
			}
		}

		if !found {
			//return nil, fmt.Errorf("access code not found")
			glog.V(4).Infof("access code %s seems to be invalid", code)
			continue
		}

		if accessCode.Spec.Expiration != "" {
			expiration, err := time.Parse(time.UnixDate, accessCode.Spec.Expiration)

			if err != nil {
				return nil, fmt.Errorf("error while parsing expiration time for access code %s %v", code, err)
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

	accessCode, err := acc.GetAccessCode(code)

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

	accessCode, err := acc.GetAccessCode(code)

	if err != nil {
		return ids, fmt.Errorf("error finding access code %s: %v", code, err)
	}

	return accessCode.Spec.Courses, nil
}

func (acc AccessCodeClient) GetClosestAccessCode(userID string, scenarioOrCourseId string) (string, error) {
	// basically let's get all of the access codes, sort them by expiration, and start going down the list looking for access codes.

	user, err := acc.hfClientSet.HobbyfarmV1().Users(util.GetReleaseNamespace()).Get(acc.ctx, userID, metav1.GetOptions{}) // @TODO: FIX THIS TO NOT DIRECTLY CALL USER

	if err != nil {
		return "", fmt.Errorf("error retrieving user: %v", err)
	}

	rawAccessCodes, err := acc.GetAccessCodes(user.Spec.AccessCodes)

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
