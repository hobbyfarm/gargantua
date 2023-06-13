package userservice

import (
	"encoding/json"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv2 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v2"
	"github.com/hobbyfarm/gargantua/pkg/util"
	userProto "github.com/hobbyfarm/gargantua/protos/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	resourcePlural = "users"
)

type PreparedUser struct {
	ID string `json:"id"`
	hfv2.UserSpec
}

func (u UserServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := util.AuthenticateRequest(r, u.tlsCaPath)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := util.AuthorizeRequest(r, u.tlsCaPath, impersonatedUserId, "hobbyfarm.io", resourcePlural, "get")
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get User")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	user, err := u.internalUserServer.GetUserById(r.Context(), &userProto.UserId{Id: id})

	if err != nil {
		if s, ok := status.FromError(err); ok {
			details := s.Details()[0].(*userProto.UserId)
			if s.Code() == codes.InvalidArgument {
				util.ReturnHTTPMessage(w, r, 500, "error", "no id passed in")
				return
			}
			glog.Errorf("error while retrieving user %s: %s", details.Id, s.Message())
			util.ReturnHTTPMessage(w, r, 500, "error", "no user found")
		}
		glog.Errorf("error while retrieving user: %s", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no user found")
	}

	preparedUser := PreparedUser{
		ID: user.Id,
		UserSpec: hfv2.UserSpec{
			Email:       user.Email,
			Password:    user.Password,
			AccessCodes: user.AccessCodes,
			Settings:    user.Settings,
		},
	}

	encodedUser, err := json.Marshal(preparedUser)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedUser)

	glog.V(2).Infof("retrieved user %s", user.Id)
}

func (u UserServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	user, err := util.AuthenticateRequest(r, u.tlsCaPath)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := util.AuthorizeRequest(r, u.tlsCaPath, impersonatedUserId, "hobbyfarm.io", resourcePlural, "list")
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get User")
		return
	}

	users, err := u.internalUserServer.ListUser(r.Context(), &emptypb.Empty{})

	if err != nil {
		glog.Errorf("error while retrieving users %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no users found")
		return
	}

	preparedUsers := []PreparedUser{} // must be declared this way so as to JSON marshal into [] instead of null
	for _, s := range users.Users {
		preparedUsers = append(preparedUsers, PreparedUser{
			ID: s.Id,
			UserSpec: hfv2.UserSpec{
				Email:       s.Email,
				Password:    s.Password,
				AccessCodes: s.AccessCodes,
				Settings:    s.Settings,
			},
		})
	}

	encodedUsers, err := json.Marshal(preparedUsers)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedUsers)

	glog.V(2).Infof("listed users")
}

func (u UserServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := util.AuthenticateRequest(r, u.tlsCaPath)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := util.AuthorizeRequest(r, u.tlsCaPath, impersonatedUserId, "hobbyfarm.io", resourcePlural, "update")
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get User")
		return
	}

	id := r.PostFormValue("id")
	email := r.PostFormValue("email")
	password := r.PostFormValue("password")
	accesscodes := r.PostFormValue("accesscodes")
	var acUnmarshaled []string
	if accesscodes != "" {
		err = json.Unmarshal([]byte(accesscodes), &acUnmarshaled)
		if err != nil {
			glog.Errorf("error while unmarshaling steps %v", err)
			util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to update")
		}
	}

	_, err = u.internalUserServer.UpdateUser(r.Context(), &userProto.User{Id: id, Email: email, Password: password, AccessCodes: acUnmarshaled})

	if err != nil {
		if s, ok := status.FromError(err); ok {
			details := s.Details()[0].(*userProto.User)
			if s.Code() == codes.InvalidArgument {
				util.ReturnHTTPMessage(w, r, 400, "badrequest", "no ID passed in")
				return
			}
			glog.Errorf("error while updating user %s: %s", details.Id, s.Message())
			util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to update")
		}
		glog.Errorf("error while updating user: %s", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to update")
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	return
}

func (u UserServer) DeleteFunc(w http.ResponseWriter, r *http.Request) {
	// criteria to delete user:
	// 1. must not have an active session
	// that's about it.

	user, err := util.AuthenticateRequest(r, u.tlsCaPath)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := util.AuthorizeRequest(r, u.tlsCaPath, impersonatedUserId, "hobbyfarm.io", resourcePlural, "get")
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get User")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]

	if len(id) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "error", "no id passed in")
		return
	}

	_, err = u.internalUserServer.DeleteUser(r.Context(), &userProto.UserId{Id: id})

	if err != nil {
		if s, ok := status.FromError(err); ok {
			details := s.Details()[0].(*userProto.UserId)
			if s.Code() == codes.InvalidArgument {
				util.ReturnHTTPMessage(w, r, 400, "error", "no id passed in")
				return
			}
			glog.Errorf("error deleting user %s: %s", details.Id, s.Message())
			util.ReturnHTTPMessage(w, r, 500, "error", s.Message())
		}
		glog.Errorf("error deleting user: %s", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error deleting user")
	}

	util.ReturnHTTPMessage(w, r, 200, "success", "user deleted")
}
