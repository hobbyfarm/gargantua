package userservice

import (
	"encoding/json"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	rbacProto "github.com/hobbyfarm/gargantua/v3/protos/rbac"
	userProto "github.com/hobbyfarm/gargantua/v3/protos/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	resourcePlural = rbac.ResourcePluralUser
)

type PreparedUser struct {
	ID          string   `json:"id"`
	Email       string   `json:"email"`
	AccessCodes []string `json:"access_codes"`
}

type PreparedSubject struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type PreparedRoleBinding struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	Subjects []PreparedSubject
}

func (u UserServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := rbac.AuthenticateRequest(r, u.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, u.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbGet))
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

	accessCodes := user.GetAccessCodes()
	// If "accessCodes" variable is nil -> convert it to an empty slice
	if accessCodes == nil {
		accessCodes = []string{}
	}
	preparedUser := PreparedUser{
		ID:          user.GetId(),
		Email:       user.GetEmail(),
		AccessCodes: accessCodes,
	}

	encodedUser, err := json.Marshal(preparedUser)
	if err != nil {
		glog.Errorf("error while marshalling json for user: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedUser)

	glog.V(2).Infof("retrieved user %s", user.Id)
}

func (u UserServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, u.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, u.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbList))
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
		accessCodes := s.GetAccessCodes()
		// If "accessCodes" variable is nil -> convert it to an empty slice
		if accessCodes == nil {
			accessCodes = []string{}
		}
		preparedUsers = append(preparedUsers, PreparedUser{
			ID:          s.GetId(),
			Email:       s.GetEmail(),
			AccessCodes: accessCodes,
		})
	}

	encodedUsers, err := json.Marshal(preparedUsers)
	if err != nil {
		glog.Error(err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "internal error")
		return
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedUsers)

	glog.V(2).Infof("listed users")
}

func (u UserServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac.AuthenticateRequest(r, u.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, u.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbUpdate))
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
}

func (u UserServer) DeleteFunc(w http.ResponseWriter, r *http.Request) {
	// criteria to delete user:
	// 1. must not have an active session
	// that's about it.

	user, err := rbac.AuthenticateRequest(r, u.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, u.authrClient, impersonatedUserId, rbac.HobbyfarmPermission(resourcePlural, rbac.VerbDelete))
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

func (u UserServer) ListRoleBindingsForUser(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := rbac.AuthenticateRequest(r, u.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, u.authrClient, impersonatedUserId, rbac.RbacPermission(rbac.ResourcePluralRolebinding, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list rolebindings")
		return
	}

	vars := mux.Vars(r)

	user := vars["user"]

	bindings, err := u.rbacClient.GetHobbyfarmRoleBindings(r.Context(), &userProto.UserId{
		Id: user,
	})

	if err != nil {
		glog.Errorf("error getting hobbyfarm rolebindings for user %s: %v", user, err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "internal error")
		return
	}

	preparedRoleBindings := []PreparedRoleBinding{}
	for _, rb := range bindings.GetRolebindings() {
		preparedRoleBindings = append(preparedRoleBindings, u.prepareRoleBinding(rb))
	}

	data, err := json.Marshal(preparedRoleBindings)
	if err != nil {
		glog.Errorf("error while marshalling json for rolebindings: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPContent(w, r, http.StatusOK, "content", data)
}

func (s UserServer) prepareRoleBinding(roleBinding *rbacProto.RoleBinding) PreparedRoleBinding {
	prb := PreparedRoleBinding{
		Name:     roleBinding.GetName(),
		Role:     roleBinding.GetRole(),
		Subjects: []PreparedSubject{},
	}

	for _, s := range roleBinding.GetSubjects() {
		prb.Subjects = append(prb.Subjects, PreparedSubject{
			Kind: s.GetKind(),
			Name: s.GetName(),
		})
	}

	return prb
}
