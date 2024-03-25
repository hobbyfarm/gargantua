package rbac

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"github.com/hobbyfarm/gargantua/v3/protos/general"
	rbacProto "github.com/hobbyfarm/gargantua/v3/protos/rbac"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PreparedRoleBinding struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	Subjects []PreparedSubject
}

type PreparedSubject struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

func (s Server) ListRoleBindings(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.RbacPermission(rbac.ResourcePluralRolebinding, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list rolebindings")
		return
	}

	labelSelector := fmt.Sprintf("%s=%t", hflabels.RBACManagedLabel, true)
	bindings, err := s.internalRbacServer.ListRolebinding(r.Context(), &general.ListOptions{LabelSelector: labelSelector})
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

	preparedRoleBindings := []PreparedRoleBinding{}
	for _, rb := range bindings.GetRolebindings() {
		preparedRoleBindings = append(preparedRoleBindings, s.prepareRoleBinding(rb))
	}

	data, err := json.Marshal(preparedRoleBindings)
	if err != nil {
		glog.Errorf("error while marshalling json for rolebindings: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPContent(w, r, http.StatusOK, "content", data)
}

func (s Server) GetRoleBinding(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.RbacPermission(rbac.ResourcePluralRolebinding, rbac.VerbGet))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, http.StatusForbidden, "forbidden", "no access to get rolebinding")
		return
	}

	vars := mux.Vars(r)
	rolebindingId := vars["id"]

	preparedRoleBinding, err := s.internalRbacServer.GetRolebinding(r.Context(), &general.GetRequest{Id: rolebindingId})
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

	data, err := util.GetProtoMarshaller().Marshal(preparedRoleBinding)
	if err != nil {
		glog.Errorf("error while marshalling json for rolebinding: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPContent(w, r, http.StatusOK, "content", data)
}

func (s Server) CreateRoleBinding(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.RbacPermission(rbac.ResourcePluralRolebinding, rbac.VerbCreate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, http.StatusForbidden, "forbidden", "no access to create rolebinding")
		return
	}

	var preparedRoleBinding *rbacProto.RoleBinding
	err = json.NewDecoder(r.Body).Decode(&preparedRoleBinding)
	if err != nil {
		glog.Errorf("error decoding json from create rolebinding request: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", "malformed json")
		return
	}

	_, err = s.internalRbacServer.CreateRolebinding(r.Context(), preparedRoleBinding)
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.InvalidArgument {
				util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", "invalid rolebinding")
				return
			}
		}
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPMessage(w, r, http.StatusOK, "created", "created")
}

func (s Server) UpdateRoleBinding(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.RbacPermission(rbac.ResourcePluralRolebinding, rbac.VerbUpdate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, http.StatusForbidden, "forbidden", "no access to update rolebinding")
		return
	}

	var preparedRoleBinding *rbacProto.RoleBinding
	err = json.NewDecoder(r.Body).Decode(&preparedRoleBinding)
	if err != nil {
		glog.Errorf("error decoding json from update rolebinding request: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", "malformed json")
		return
	}

	_, err = s.internalRbacServer.UpdateRolebinding(r.Context(), preparedRoleBinding)
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.InvalidArgument {
				util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", "invalid rolebinding")
				return
			}
		}
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPMessage(w, r, http.StatusOK, "updated", "updated")
}

func (s Server) DeleteRoleBinding(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := rbac.AuthenticateRequest(r, s.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.authrClient, impersonatedUserId, rbac.RbacPermission(rbac.ResourcePluralRolebinding, rbac.VerbDelete))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, http.StatusForbidden, "forbidden", "no access to delete rolebinding")
		return
	}

	vars := mux.Vars(r)
	rolebindingId := vars["id"]

	_, err = s.internalRbacServer.DeleteRolebinding(r.Context(), &general.ResourceId{Id: rolebindingId})
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.InvalidArgument {
				util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", "invalid rolebinding")
				return
			}
		}
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPMessage(w, r, http.StatusOK, "deleted", "deleted")
}

func (s Server) prepareRoleBinding(roleBinding *rbacProto.RoleBinding) PreparedRoleBinding {
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
