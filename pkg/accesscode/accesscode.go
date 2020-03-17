package accesscode

import (
	"fmt"
	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
	"time"
)

type AccessCodeClient struct {
	hfClientSet *hfClientset.Clientset
}

func NewAccessCodeClient(hfClientset *hfClientset.Clientset) (*AccessCodeClient, error) {
	acc := AccessCodeClient{}
	acc.hfClientSet = hfClientset
	return &acc, nil
}

func (acc AccessCodeClient) GetSomething(code string) error {
	return nil
}

func (acc AccessCodeClient) GetAccessCodes(codes []string, expiredOk bool) ([]hfv1.AccessCode, error) {
	if len(codes) == 0 {
		return nil, fmt.Errorf("code list passed in was less than 0")
	}

	accessCodeList, err := acc.hfClientSet.HobbyfarmV1().AccessCodes().List(metav1.ListOptions{})

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

		if !expiredOk {
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
		}
		accessCodes = append(accessCodes, accessCode)
	}

	return accessCodes, nil

}

func (acc AccessCodeClient) GetAccessCode(code string, expiredOk bool) (hfv1.AccessCode, error) {
	if len(code) == 0 {
		return hfv1.AccessCode{}, fmt.Errorf("code was empty")
	}

	accessCodes, err := acc.GetAccessCodes([]string{code}, false)

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

	accessCode, err := acc.GetAccessCode(code, false)

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

	accessCode, err := acc.GetAccessCode(code, false)

	if err != nil {
		return ids, fmt.Errorf("error finding access code %s: %v", code, err)
	}

	return accessCode.Spec.Courses, nil
}

func (acc AccessCodeClient) GetClosestAccessCode(userID string, scenario string) (string, error) {
	// basically let's get all of the access codes, sort them by expiration, and start going down the list looking for access codes.

	user, err := acc.hfClientSet.HobbyfarmV1().Users().Get(userID, metav1.GetOptions{}) // @TODO: FIX THIS TO NOT DIRECTLY CALL USER

	if err != nil {
		return "", fmt.Errorf("error retrieving user: %v", err)
	}

	accessCodes, err := acc.GetAccessCodes(user.Spec.AccessCodes, false)

	if err != nil {
		return "", fmt.Errorf("access codes were not found %v", err)
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
