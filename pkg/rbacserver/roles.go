package rbacserver

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/pkg/rbacclient"
	"github.com/hobbyfarm/gargantua/pkg/util"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
)

type PreparedRole struct {
	Name string `json:"name"`
	Rules []PreparedRule `json:"rules"`
}

type PreparedRule struct {
	Verbs []string 	`json:"verbs"`
	APIGroups []string `json:"apiGroups"`
	Resources []string `json:"resources"`
}

func (s Server) ListRoles(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().Permission(k8sRbacGroup, roleResourcePlural, rbacclient.VerbList), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list roles")
		return
	}

	listOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%t", rbacManagedLabel, true),
	}

	roles, err := s.kubeClientSet.RbacV1().Roles(util.GetReleaseNamespace()).List(r.Context(), listOptions)
	if err != nil {
		if errors.IsNotFound(err) {
			util.ReturnHTTPMessage(w, r, 404, "notfound", "roles not found")
		} else {
			util.ReturnHTTPMessage(w, r, 500, "internalerror","internal error")
		}
		return
	}

	var preparedRoles = make([]PreparedRole, 0)
	for _, r := range roles.Items {
		pr := s.unmarshalRole(&r)
		preparedRoles = append(preparedRoles, *pr)
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
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().Permission(k8sRbacGroup, roleResourcePlural, rbacclient.VerbGet), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get role")
		return
	}

	role, err := s.getRole(w, r)
	if err != nil {
		return
	}

	preparedRole := s.unmarshalRole(role)

	data, err := json.Marshal(preparedRole)
	if err != nil {
		glog.Errorf("error while marshalling json for role: %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPContent(w, r, 200, "content", data)
}

func (s Server) CreateRole(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().Permission(k8sRbacGroup, roleResourcePlural, rbacclient.VerbCreate), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create role")
		return
	}

	var preparedRole PreparedRole
	err = json.NewDecoder(r.Body).Decode(&preparedRole)
	if err != nil {
		glog.Errorf("error decoding json from create role request: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", "malformed json")
		return
	}

	role, err := s.marshalRole(&preparedRole)
	if err != nil {
		glog.Errorf("invalid role: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", "invalid role")
		return
	}

	role, err = s.kubeClientSet.RbacV1().Roles(util.GetReleaseNamespace()).Create(r.Context(), role, metav1.CreateOptions{})
	if err != nil {
		glog.Errorf("error creating role in kubernetes: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPMessage(w, r, http.StatusOK, "created", "created")
}

func (s Server) UpdateRole(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().Permission(k8sRbacGroup, roleResourcePlural, rbacclient.VerbUpdate), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, http.StatusForbidden, "forbidden", "no access to update role")
		return
	}

	var preparedRole PreparedRole
	err = json.NewDecoder(r.Body).Decode(&preparedRole)
	if err != nil {
		glog.Errorf("error decoding json from create role request: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "badrequest", "malformed json")
		return
	}

	role, err := s.marshalRole(&preparedRole)
	if err != nil {
		glog.Errorf("invalid role: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "badrequest", "invalid role")
		return
	}

	role, err = s.kubeClientSet.RbacV1().Roles(util.GetReleaseNamespace()).Update(r.Context(), role, metav1.UpdateOptions{})
	if err != nil {
		glog.Errorf("error while updating role in kubernetes: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error")
		return
	}

	util.ReturnHTTPMessage(w, r, http.StatusOK, "updated", "updated")
}

func (s Server) DeleteRole(w http.ResponseWriter, r *http.Request) {
	_, err := s.auth.AuthGrant(rbacclient.RbacRequest().Permission(k8sRbacGroup, roleResourcePlural, rbacclient.VerbDelete), w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, http.StatusForbidden, "forbidden", "no access to update role")
		return
	}

	// we want to get the role first as this allows us to run it through the various checks before we attempt deletion
	// most important of which is checking that we have labeled it correctly
	// but it doesn't hurt to check if it exists before
	role, err := s.getRole(w, r)
	if err != nil {
		return
	}

	err = s.kubeClientSet.RbacV1().Roles(util.GetReleaseNamespace()).Delete(r.Context(), role.Name, metav1.DeleteOptions{})
	if err != nil {
		glog.Errorf("error deleting role in kubernetes: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror","internal error")
		return
	}

	util.ReturnHTTPMessage(w, r, http.StatusOK, "deleted", "deleted")
}

func (s Server) getRole(w http.ResponseWriter, r *http.Request) (*rbacv1.Role, error) {
	vars := mux.Vars(r)

	roleId := vars["id"]
	if roleId == "" {
		util.ReturnHTTPMessage(w, r, http.StatusBadRequest, "invalidid", "invalid id")
		return nil, fmt.Errorf("invalid id")
	}

	role, err := s.kubeClientSet.RbacV1().Roles(util.GetReleaseNamespace()).Get(r.Context(), roleId, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "notfound", "role not found")
			return nil, err
		}
		glog.Errorf("kubernetes error while getting role: %v", err)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal server error")
		return nil, err
	}

	if _, ok := role.Labels[rbacManagedLabel]; !ok {
		// this isn't a hobbyfarm role. we don't serve your kind here
		util.ReturnHTTPMessage(w, r, http.StatusForbidden, "forbidden", "role not managed by hobbyfarm")
		return nil, fmt.Errorf("role not managed by hobbyfarm")
	}

	return role, nil
}

func (s Server) sanitizeRules(rules []PreparedRule) error {
	for _, rule := range rules {
		for _, group := range rule.APIGroups {
			if group != "hobbyfarm.io" && group != "rbac.authorization.k8s.io" {
				return fmt.Errorf("invalid api group specified: %v", group)
			}
		}
	}

	return nil
}

func (s Server) unmarshalRole(role *rbacv1.Role) (preparedRole *PreparedRole) {
	preparedRole = &PreparedRole{}
	preparedRole.Name = role.Name

	for _, r := range role.Rules {
		preparedRole.Rules = append(preparedRole.Rules, PreparedRule{
			Resources: r.Resources,
			Verbs: r.Verbs,
			APIGroups: r.APIGroups,
		})
	}

	return
}

func (s Server) marshalRole(preparedRole *PreparedRole) (*rbacv1.Role, error) {
	role := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      preparedRole.Name,
			Namespace: util.GetReleaseNamespace(),
			Labels: map[string]string{
				rbacManagedLabel: "true",
			},
		},
		Rules: []rbacv1.PolicyRule{},
	}

	for _, r := range preparedRole.Rules {
		for _, group := range r.APIGroups {
			if group != "hobbyfarm.io" && group != "rbac.authorization.k8s.io" {
				return nil, fmt.Errorf("invalid api group specified: %v", group)
			}
		}

		role.Rules = append(role.Rules, rbacv1.PolicyRule{
			Verbs: r.Verbs,
			APIGroups: r.APIGroups,
			Resources: r.Resources,
		})
	}

	return &role, nil
}