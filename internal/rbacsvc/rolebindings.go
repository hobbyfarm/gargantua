package rbac

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/pkg/rbac"
	"github.com/hobbyfarm/gargantua/pkg/util"
	userProto "github.com/hobbyfarm/gargantua/protos/user"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (s Server) ListRoleBindingsForUser(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := rbac.AuthenticateRequest(r, s.tlsCA)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.tlsCA, impersonatedUserId, rbac.RbacPermission(roleBindingResourcePlural, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list rolebindings")
		return
	}

	vars := mux.Vars(r)

	user := vars["user"]

	bindings, err := s.internalRbacServer.GetHobbyfarmRoleBindings(r.Context(), &userProto.UserId{
		Id: user,
	})

	if err != nil {
		glog.Errorf("error getting hobbyfarm rolebindings for user %s: %v", user, err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "internal error")
		return
	}

	data, err := json.Marshal(bindings.GetRolebindings())
	if err != nil {
		glog.Errorf("error while marshalling json for rolebindings: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPContent(w, r, http.StatusOK, "content", data)
}

func (s Server) ListRoleBindings(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := rbac.AuthenticateRequest(r, s.tlsCA)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.tlsCA, impersonatedUserId, rbac.RbacPermission(roleBindingResourcePlural, rbac.VerbList))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list rolebindings")
		return
	}

	listOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%t", util.RBACManagedLabel, true),
	}

	roleBindings, err := s.kubeClientSet.RbacV1().RoleBindings(util.GetReleaseNamespace()).List(r.Context(), listOptions)
	if err != nil {
		if errors.IsNotFound(err) {
			util.ReturnHTTPMessage(w, r, 404, "notfound", "rolebindings not found")
		} else {
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "internal error")
		}
		return
	}

	var preparedRoleBindings = make([]PreparedRoleBinding, 0)

	for _, r := range roleBindings.Items {
		prb := s.prepareRoleBinding(r)
		preparedRoleBindings = append(preparedRoleBindings, prb)
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
	authenticatedUser, err := rbac.AuthenticateRequest(r, s.tlsCA)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.tlsCA, impersonatedUserId, rbac.RbacPermission(roleBindingResourcePlural, rbac.VerbGet))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, http.StatusForbidden, "forbidden", "no access to get rolebinding")
		return
	}

	roleBinding, err := s.getRoleBinding(w, r)
	if err != nil {
		return // return message already handled in getRoleBinding()
	}

	preparedRoleBinding := s.prepareRoleBinding(*roleBinding)

	data, err := json.Marshal(preparedRoleBinding)
	if err != nil {
		glog.Errorf("error while marshalling json for rolebinding: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPContent(w, r, http.StatusOK, "content", data)
}

func (s Server) CreateRoleBinding(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := rbac.AuthenticateRequest(r, s.tlsCA)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.tlsCA, impersonatedUserId, rbac.RbacPermission(roleBindingResourcePlural, rbac.VerbCreate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, http.StatusForbidden, "forbidden", "no access to create rolebinding")
		return
	}

	var preparedRoleBinding PreparedRoleBinding
	err = json.NewDecoder(r.Body).Decode(&preparedRoleBinding)
	if err != nil {
		glog.Errorf("error decoding json from create rolebinding request: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", "malformed json")
		return
	}

	roleBinding, err := s.unmarshalRoleBinding(r.Context(), &preparedRoleBinding)
	if err != nil {
		glog.Errorf("error during role binding validation: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", err.Error())
		return
	}

	roleBinding, err = s.kubeClientSet.RbacV1().RoleBindings(util.GetReleaseNamespace()).Create(r.Context(), roleBinding, metav1.CreateOptions{})
	if err != nil {
		glog.Errorf("error creating rolebinding in kubernetes: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPMessage(w, r, http.StatusOK, "created", "created")
}

func (s Server) UpdateRoleBinding(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := rbac.AuthenticateRequest(r, s.tlsCA)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.tlsCA, impersonatedUserId, rbac.RbacPermission(roleBindingResourcePlural, rbac.VerbUpdate))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, http.StatusForbidden, "forbidden", "no access to update rolebinding")
		return
	}

	var preparedRoleBinding PreparedRoleBinding
	err = json.NewDecoder(r.Body).Decode(&preparedRoleBinding)
	if err != nil {
		glog.Errorf("error decoding json from update rolebinding request: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", "malformed json")
		return
	}

	inputRoleBinding, err := s.unmarshalRoleBinding(r.Context(), &preparedRoleBinding)
	if err != nil {
		glog.Errorf("error during role binding validation: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", err.Error())
		return
	}

	// get the rolebinding from kubernetes
	k8sRoleBinding, err := s.getRoleBinding(w, r)
	if err != nil {
		return
	}

	k8sRoleBinding.RoleRef = inputRoleBinding.RoleRef
	k8sRoleBinding.Subjects = inputRoleBinding.Subjects

	k8sRoleBinding, err = s.kubeClientSet.RbacV1().RoleBindings(util.GetReleaseNamespace()).Update(r.Context(), k8sRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		glog.Errorf("errro while updating rolebinding in kubernetes: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPMessage(w, r, http.StatusOK, "updated", "updated")
}

func (s Server) DeleteRoleBinding(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, err := rbac.AuthenticateRequest(r, s.tlsCA)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := authenticatedUser.GetId()
	authrResponse, err := rbac.AuthorizeSimple(r, s.tlsCA, impersonatedUserId, rbac.RbacPermission(roleBindingResourcePlural, rbac.VerbDelete))
	if err != nil || !authrResponse.Success {
		util.ReturnHTTPMessage(w, r, http.StatusForbidden, "forbidden", "no access to delete rolebinding")
		return
	}

	roleBinding, err := s.getRoleBinding(w, r)
	if err != nil {
		return
	}

	err = s.kubeClientSet.RbacV1().RoleBindings(util.GetReleaseNamespace()).Delete(r.Context(), roleBinding.Name, metav1.DeleteOptions{})
	if err != nil {
		glog.Errorf("error deleting rolebinding in kubernetes: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPMessage(w, r, http.StatusOK, "deleted", "deleted")
}

func (s Server) getRoleBinding(w http.ResponseWriter, r *http.Request) (*rbacv1.RoleBinding, error) {
	vars := mux.Vars(r)

	roleBindingId := vars["id"]
	if roleBindingId == "" {
		util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "invalidid", "invalid id")
		return nil, fmt.Errorf("invalid id")
	}

	roleBinding, err := s.kubeClientSet.RbacV1().RoleBindings(util.GetReleaseNamespace()).Get(
		r.Context(), roleBindingId, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "notfound", "rolebinding not found")
		} else {
			glog.Errorf("kubernetes error while getting rolebinding: %v", err)
			util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		}
		return nil, err
	}

	if _, ok := roleBinding.Labels[util.RBACManagedLabel]; !ok {
		util.ReturnHTTPMessage(w, r, http.StatusForbidden, "forbidden", "rolebinding not managed by hobbyfarm")
		return nil, fmt.Errorf("rolebinding not managed by hobbyfarm")
	}

	return roleBinding, nil
}

func (s Server) prepareRoleBinding(roleBinding rbacv1.RoleBinding) PreparedRoleBinding {
	prb := PreparedRoleBinding{
		Name:     roleBinding.Name,
		Role:     roleBinding.RoleRef.Name,
		Subjects: []PreparedSubject{},
	}

	for _, s := range roleBinding.Subjects {
		prb.Subjects = append(prb.Subjects, PreparedSubject{
			Kind: s.Kind,
			Name: s.Name,
		})
	}

	return prb
}

func (s Server) unmarshalRoleBinding(ctx context.Context, preparedRoleBinding *PreparedRoleBinding) (*rbacv1.RoleBinding, error) {
	// first validation, the role it is referencing has to exist
	role, err := s.kubeClientSet.RbacV1().Roles(util.GetReleaseNamespace()).Get(ctx, preparedRoleBinding.Role, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("invalid role ref")
	}

	rb := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      preparedRoleBinding.Name,
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				util.RBACManagedLabel: "true",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "rbac.authorization.k8s.io/v1",
					Kind:       "Role",
					Name:       role.Name,
					UID:        role.UID,
				},
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: k8sRbacGroup,
			Name:     preparedRoleBinding.Role,
			Kind:     "Role",
		},
		Subjects: []rbacv1.Subject{},
	}

	for _, s := range preparedRoleBinding.Subjects {
		if s.Kind != "Group" && s.Kind != "User" {
			return nil, fmt.Errorf("invalid subject kind")
		}

		rb.Subjects = append(rb.Subjects, rbacv1.Subject{
			Kind:     s.Kind,
			Name:     s.Name,
			APIGroup: k8sRbacGroup,
		})
	}

	return &rb, nil
}
