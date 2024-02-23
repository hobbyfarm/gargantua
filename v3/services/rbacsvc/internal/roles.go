package rbac

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"github.com/hobbyfarm/gargantua/v3/protos/general"
	rbacProto "github.com/hobbyfarm/gargantua/v3/protos/rbac"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PreparedRole struct {
	Name  string         `json:"name"`
	Rules []PreparedRule `json:"rules"`
}

type PreparedRule struct {
	Verbs     []string `json:"verbs"`
	APIGroups []string `json:"apiGroups"`
	Resources []string `json:"resources"`
}

func (s Server) ListRoles(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.RbacPermission(rbac.ResourcePluralRole, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list roles")
		return
	}

	labelSelector := fmt.Sprintf("%s=%t", util.RBACManagedLabel, true)
	roles, err := s.internalRbacServer.ListRole(r.Context(), &general.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		if s, ok := status.FromError(err); ok {
			switch s.Code() {
			case codes.NotFound:
				util.ReturnHTTPMessage(w, r, http.StatusNotFound, "notfound", s.Message())
				return
			}
		}
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}

	preparedRoles := []PreparedRole{}
	for _, r := range roles.GetRoles() {
		preparedRoles = append(preparedRoles, s.prepareRole(r))
	}

	data, err := json.Marshal(preparedRoles)
	if err != nil {
		glog.Errorf("error while marshalling json for roles: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPContent(w, r, 200, "content", data)
}

func (s Server) GetRole(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.RbacPermission(rbac.ResourcePluralRole, rbac.VerbGet))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get role")
		return
	}

	vars := mux.Vars(r)
	roleId := vars["id"]

	preparedRole, err := s.internalRbacServer.GetRole(r.Context(), &general.GetRequest{Id: roleId})
	if err != nil {
		if s, ok := status.FromError(err); ok {
			switch s.Code() {
			case codes.InvalidArgument:
				util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", s.Message())
				return
			case codes.NotFound:
				util.ReturnHTTPMessage(w, r, http.StatusNotFound, "notfound", s.Message())
				return
			case codes.PermissionDenied:
				util.ReturnHTTPMessage(w, r, http.StatusForbidden, "forbidden", s.Message())
				return
			}
		}
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}

	data, err := util.GetProtoMarshaller().Marshal(preparedRole)
	if err != nil {
		glog.Errorf("error while marshalling json for role: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPContent(w, r, 200, "content", data)
}

func (s Server) CreateRole(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.RbacPermission(rbac.ResourcePluralRole, rbac.VerbCreate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create role")
		return
	}

	var preparedRole *rbacProto.Role
	err = json.NewDecoder(r.Body).Decode(&preparedRole)
	if err != nil {
		glog.Errorf("error decoding json from create role request: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", "malformed json")
		return
	}

	_, err = s.internalRbacServer.CreateRole(r.Context(), preparedRole)
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.InvalidArgument {
				util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", "invalid role")
				return
			}
		}
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPMessage(w, r, http.StatusOK, "created", "created")
}

func (s Server) UpdateRole(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.RbacPermission(rbac.ResourcePluralRole, rbac.VerbUpdate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, http.StatusForbidden, "forbidden", "no access to update role")
		return
	}

	var preparedRole *rbacProto.Role
	err = json.NewDecoder(r.Body).Decode(&preparedRole)
	if err != nil {
		glog.Errorf("error decoding json from update role request: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", "malformed json")
		return
	}

	_, err = s.internalRbacServer.UpdateRole(r.Context(), preparedRole)
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.InvalidArgument {
				util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", "invalid role")
				return
			}
		}
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPMessage(w, r, http.StatusOK, "updated", "updated")
}

func (s Server) DeleteRole(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.RbacPermission(rbac.ResourcePluralRole, rbac.VerbDelete))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, http.StatusForbidden, "forbidden", "no access to delete role")
		return
	}

	vars := mux.Vars(r)
	roleId := vars["id"]

	_, err = s.internalRbacServer.DeleteRole(r.Context(), &general.ResourceId{Id: roleId})
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.InvalidArgument {
				util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", "invalid role")
				return
			}
		}
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPMessage(w, r, http.StatusOK, "deleted", "deleted")
}

func (s Server) prepareRole(role *rbacProto.Role) (preparedRole PreparedRole) {
	pr := PreparedRole{
		Name:  role.GetName(),
		Rules: []PreparedRule{},
	}

	for _, r := range role.GetRules() {
		pr.Rules = append(preparedRole.Rules, PreparedRule{
			Resources: r.GetResources(),
			Verbs:     r.GetVerbs(),
			APIGroups: r.GetApiGroups(),
		})
	}

	return pr
}
