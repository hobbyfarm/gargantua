package userserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"golang.org/x/crypto/bcrypt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"net/http"
	"strings"
)

type UserServer struct {
	auth        *authclient.AuthClient
	hfClientSet hfClientset.Interface
	ctx         context.Context
}

func NewUserServer(authClient *authclient.AuthClient, hfClientset hfClientset.Interface, ctx context.Context) (*UserServer, error) {
	s := UserServer{}

	s.hfClientSet = hfClientset
	s.auth = authClient
	s.ctx = ctx

	return &s, nil
}

func (u UserServer) getUser(id string) (hfv1.User, error) {

	empty := hfv1.User{}

	if len(id) == 0 {
		return empty, fmt.Errorf("user id passed in was empty")
	}

	obj, err := u.hfClientSet.HobbyfarmV1().Users().Get(u.ctx, id, metav1.GetOptions{})
	if err != nil {
		return empty, fmt.Errorf("error while retrieving User by id: %s with error: %v", id, err)
	}

	return *obj, nil

}

func (u UserServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/user/list", u.ListFunc).Methods("GET")
	r.HandleFunc("/a/user/{id}", u.GetFunc).Methods("GET")
	r.HandleFunc("/a/user", u.UpdateFunc).Methods("PUT")
	glog.V(2).Infof("set up routes for User server")
}

type PreparedUser struct {
	ID string `json:"id"`
	hfv1.UserSpec
}

func (u UserServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	_, err := u.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get User")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no id passed in")
		return
	}

	user, err := u.getUser(id)

	if err != nil {
		glog.Errorf("error while retrieving user %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no user found")
		return
	}

	preparedUser := PreparedUser{user.Name, user.Spec}

	encodedUser, err := json.Marshal(preparedUser)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedUser)

	glog.V(2).Infof("retrieved user %s", user.Name)
}

func (u UserServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	_, err := u.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list users")
		return
	}

	users, err := u.hfClientSet.HobbyfarmV1().Users().List(u.ctx, metav1.ListOptions{})

	if err != nil {
		glog.Errorf("error while retrieving users %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no users found")
		return
	}

	preparedUsers := []PreparedUser{} // must be declared this way so as to JSON marshal into [] instead of null
	for _, s := range users.Items {
		preparedUsers = append(preparedUsers, PreparedUser{s.Name, s.Spec})
	}

	encodedUsers, err := json.Marshal(preparedUsers)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedUsers)

	glog.V(2).Infof("listed users")
}

func (u UserServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	_, err := u.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update users")
		return
	}

	id := r.PostFormValue("id")
	if id == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ID passed in")
		return
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		user, err := u.hfClientSet.HobbyfarmV1().Users().Get(u.ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return fmt.Errorf("bad")
		}

		email := r.PostFormValue("email")
		password := r.PostFormValue("password")
		accesscodes := r.PostFormValue("accesscodes")
		admin := r.PostFormValue("admin")

		if email != "" {
			user.Spec.Email = email
		}
		if password != "" {
			passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("bad")
			}
			user.Spec.Password = string(passwordHash)
		}

		if accesscodes != "" {
			var acUnmarshaled []string

			err = json.Unmarshal([]byte(accesscodes), &acUnmarshaled)
			if err != nil {
				glog.Errorf("error while unmarshaling steps %v", err)
				return fmt.Errorf("bad")
			}
			user.Spec.AccessCodes = acUnmarshaled
		}

		if admin != "" {
			if strings.ToLower(admin) == "true" {
				user.Spec.Admin = true
			} else {
				user.Spec.Admin = false
			}
		}

		_, updateErr := u.hfClientSet.HobbyfarmV1().Users().Update(u.ctx, user, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to update")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	return
}
