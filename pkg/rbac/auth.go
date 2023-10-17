package rbac

import (
	"net/http"

	"github.com/hobbyfarm/gargantua/v3/protos/authn"
	"github.com/hobbyfarm/gargantua/v3/protos/authr"
	userProto "github.com/hobbyfarm/gargantua/v3/protos/user"
)

func AuthenticateRequest(r *http.Request, authnClient authn.AuthNClient) (*userProto.User, error) {
	token := r.Header.Get("Authorization")
	return authnClient.AuthN(r.Context(), &authn.AuthNRequest{Token: token})
}

func AuthenticateWS(r *http.Request, authnClient authn.AuthNClient) (*userProto.User, error) {
	token := "Bearer " + r.URL.Query().Get("auth")
	return authnClient.AuthN(r.Context(), &authn.AuthNRequest{Token: token})
}

func AuthorizeSimple(r *http.Request, authrClient authr.AuthRClient, username string, permission *authr.Permission) (*authr.AuthRResponse, error) {
	rbacPermissions := []*authr.Permission{
		permission,
	}
	rbacRq := &authr.RbacRequest{
		Permissions: rbacPermissions,
	}
	return authrClient.AuthR(r.Context(), &authr.AuthRRequest{UserName: username, Request: rbacRq})
}

func Authorize(r *http.Request, authrClient authr.AuthRClient, username string, permissions []*authr.Permission, operator string) (*authr.AuthRResponse, error) {
	rbacRq := &authr.RbacRequest{
		Operator:    operator,
		Permissions: permissions,
	}
	return authrClient.AuthR(r.Context(), &authr.AuthRRequest{UserName: username, Request: rbacRq})
}

func Permission(apiGroup string, resource string, verb string) *authr.Permission {
	return &authr.Permission{
		ApiGroup: apiGroup,
		Resource: resource,
		Verb:     verb,
	}
}

func HobbyfarmPermission(resource string, verb string) *authr.Permission {
	return Permission(HobbyfarmGroup, resource, verb)
}

func RbacPermission(resource string, verb string) *authr.Permission {
	return Permission(RbacGroup, resource, verb)
}
