package accesscodeserver

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

type AdminAccessCodeServer struct {
	auth        *authclient.AuthClient
	hfClientSet *hfClientset.Clientset
}

func NewAdminAccessCodeServer(authClient *authclient.AuthClient, hfClientset *hfClientset.Clientset) (*AdminAccessCodeServer, error) {
	s := AdminAccessCodeServer{}

	s.hfClientSet = hfClientset
	s.auth = authClient

	return &s, nil
}

func (a AdminAccessCodeServer) getAccessCode(id string) (hfv1.AccessCode, error) {

	empty := hfv1.AccessCode{}

	if len(id) == 0 {
		return empty, fmt.Errorf("AccessCode id passed in was empty")
	}

	obj, err := a.hfClientSet.HobbyfarmV1().AccessCodes().Get(id, metav1.GetOptions{})
	if err != nil {
		return empty, fmt.Errorf("error while retrieving AccessCode by id: %s with error: %v", id, err)
	}

	return *obj, nil

}

func (a AdminAccessCodeServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/accesscodes", a.ListFunc).Methods("GET")
	r.HandleFunc("/a/accesscodes", a.CreateFunc).Methods("POST")
	r.HandleFunc("/a/accesscodes/{id}", a.GetFunc).Methods("GET")
	r.HandleFunc("/a/accesscodes/{id}", a.DeleteFunc).Methods("DELETE")
	r.HandleFunc("/a/accesscodes/{id}", a.UpdateFunc).Methods("PUT")
	glog.V(2).Infof("set up routes for AccessCode server")
}

type PreparedAccessCode struct {
	ID string `json:"id"`
	hfv1.AccessCodeSpec
}

func (a AdminAccessCodeServer) CreateFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create accesscodes")
		return
	}

	code := r.PostFormValue("code")
	if code == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no code passed in for access code creation")
		return
	}

	accessCode := &hfv1.AccessCode{}

	hasher := sha256.New()
	hasher.Write([]byte(code))
	sha := base32.StdEncoding.WithPadding(-1).EncodeToString(hasher.Sum(nil))[:10]
	accessCode.Name = "s-" + strings.ToLower(sha)
	accessCode.Spec.Code = code

	description := r.PostFormValue("description")
	if description != "" {
		accessCode.Spec.Description = description
	}

	expiration := r.PostFormValue("expiration")
	if expiration != "" {
		accessCode.Spec.Expiration = expiration
	}

	maxUsers := r.PostFormValue("max_users")
	if maxUsers != "" {
		accessCode.Spec.MaxUsers, err = strconv.Atoi(maxUsers)
		if err != nil {
			glog.Errorf("error while converting max users (%s) string to int: %v", maxUsers, err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}

	}

	allowedDomains := []string{}
	rawDomains := r.PostFormValue("allowed_domains")
	if rawDomains != "" {
		err = json.Unmarshal([]byte(rawDomains), &allowedDomains)
		if err != nil {
			glog.Errorf("error while unmarshaling allowed domains %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
	}
	if allowedDomains != nil {
		accessCode.Spec.AllowedDomains = allowedDomains
	}

	scenarios := []string{}
	rawScenarios := r.PostFormValue("scenarios")
	if rawScenarios != "" {
		err = json.Unmarshal([]byte(rawScenarios), &scenarios)
		if err != nil {
			glog.Errorf("error while unmarshaling scenarios %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
	}
	if scenarios != nil {
		accessCode.Spec.Scenarios = scenarios
	}

	courses := []string{}
	rawCourses := r.PostFormValue("courses")
	if rawCourses != "" {
		err = json.Unmarshal([]byte(rawCourses), &courses)
		if err != nil {
			glog.Errorf("error while unmarshaling courses %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			return
		}
	}
	if courses != nil {
		accessCode.Spec.Courses = courses
	}

	accessCode, err = a.hfClientSet.HobbyfarmV1().AccessCodes().Create(accessCode)
	if err != nil {
		glog.Errorf("error creating access code %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating access code")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", accessCode.Name)
	return
}

func (a AdminAccessCodeServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get AccessCode")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no id passed in")
		return
	}

	ac, err := a.getAccessCode(id)

	if err != nil {
		glog.Errorf("error while retrieving accesscode %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no accesscode found")
		return
	}

	preparedAccessCode := PreparedAccessCode{ac.Name, ac.Spec}

	encodedAccessCode, err := json.Marshal(preparedAccessCode)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedAccessCode)

	glog.V(2).Infof("retrieved accesscode %s", ac.Name)
}

func (a AdminAccessCodeServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list accesscodes")
		return
	}

	ac, err := a.hfClientSet.HobbyfarmV1().AccessCodes().List(metav1.ListOptions{})

	if err != nil {
		glog.Errorf("error while retrieving accesscodes %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no accesscodes found")
		return
	}

	preparedAccessCodes := []PreparedAccessCode{}
	for _, s := range ac.Items {
		preparedAccessCodes = append(preparedAccessCodes, PreparedAccessCode{s.Name, s.Spec})
	}

	encodedAccessCodes, err := json.Marshal(preparedAccessCodes)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedAccessCodes)

	glog.V(2).Infof("listed accesscodes")
}

func (a AdminAccessCodeServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update accesscodes")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ac, err := a.hfClientSet.HobbyfarmV1().AccessCodes().Get(id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return fmt.Errorf("bad")
		}

		code := r.PostFormValue("code")
		if code != "" {
			ac.Spec.Code = code
		}
		description := r.PostFormValue("description")
		if description != "" {
			ac.Spec.Description = description
		}
		expiration := r.PostFormValue("expiration")
		if expiration == "null" {
			ac.Spec.Expiration = ""
		} else if expiration != "" {
			ac.Spec.Expiration = expiration
		}
		restrictedBindValue := r.PostFormValue("restricted_bind_value")
		if restrictedBindValue != "" {
			ac.Spec.RestrictedBind = true
			ac.Spec.RestrictedBindValue = restrictedBindValue
		} else {
			ac.Spec.RestrictedBind = false
			ac.Spec.RestrictedBindValue = ""
		}
		allowedDomains := r.PostFormValue("allowed_domains")
		if allowedDomains != "" {
			err := json.Unmarshal([]byte(allowedDomains), &ac.Spec.AllowedDomains)
			if err != nil {
				glog.Errorf("unable to unmarshall allowed domains json array: %v", err)
				util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to update")
			}
		}
		maxUsers := r.PostFormValue("max_users")
		if maxUsers != "" {
			ac.Spec.MaxUsers, err = strconv.Atoi(maxUsers)
			if err != nil {
				glog.Errorf("unable to convert max users to int: %v", err)
				util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to update")
			}
		}

		scenarios := []string{}
		rawScenarios := r.PostFormValue("scenarios")
		if rawScenarios != "" {
			err = json.Unmarshal([]byte(rawScenarios), &scenarios)
			if err != nil {
				glog.Errorf("error while unmarshaling scenarios %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			}
		}
		if scenarios != nil {
			ac.Spec.Scenarios = scenarios
		}

		courses := []string{}
		rawCourses := r.PostFormValue("courses")
		if rawCourses != "" {
			err = json.Unmarshal([]byte(rawCourses), &courses)
			if err != nil {
				glog.Errorf("error while unmarshaling courses %v", err)
				util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing")
			}
		}
		if courses != nil {
			ac.Spec.Courses = courses
		}

		_, updateErr := a.hfClientSet.HobbyfarmV1().AccessCodes().Update(ac)
		return updateErr
	})

	if retryErr != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to update")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	return
}

func (a AdminAccessCodeServer) DeleteFunc(w http.ResponseWriter, r *http.Request) {
	_, err := a.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to delete accesscodes")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		deleteErr := a.hfClientSet.HobbyfarmV1().AccessCodes().Delete(id, &metav1.DeleteOptions{})
		if deleteErr != nil {
			glog.Error(err)
			return fmt.Errorf("unable to delete access code: %s", id)
		}

		return deleteErr
	})

	if retryErr != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to delete access code")
		return
	}

	glog.Info("access code deleted: %s", id)
	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	return
}
