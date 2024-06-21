package rbac

import (
	"net/http"

	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	userpb "github.com/hobbyfarm/gargantua/v3/protos/user"
)

func AuthenticateRequest(r *http.Request, authnClient authnpb.AuthNClient) (*userpb.User, error) {
	token := r.Header.Get("Authorization")
	return authnClient.AuthN(r.Context(), &authnpb.AuthNRequest{Token: token})
}

func AuthenticateWS(r *http.Request, authnClient authnpb.AuthNClient) (*userpb.User, error) {
	token := "Bearer " + r.URL.Query().Get("auth")
	return authnClient.AuthN(r.Context(), &authnpb.AuthNRequest{Token: token})
}

func AuthorizeSimple(r *http.Request, authrClient authrpb.AuthRClient, username string, permission *authrpb.Permission) (*authrpb.AuthRResponse, error) {
	rbacPermissions := []*authrpb.Permission{
		permission,
	}
	rbacRq := &authrpb.RbacRequest{
		Permissions: rbacPermissions,
	}
	return authrClient.AuthR(r.Context(), &authrpb.AuthRRequest{UserName: username, Request: rbacRq})
}

func Authorize(r *http.Request, authrClient authrpb.AuthRClient, username string, permissions []*authrpb.Permission, operator string) (*authrpb.AuthRResponse, error) {
	rbacRq := &authrpb.RbacRequest{
		Operator:    operator,
		Permissions: permissions,
	}
	return authrClient.AuthR(r.Context(), &authrpb.AuthRRequest{UserName: username, Request: rbacRq})
}

func Permission(apiGroup string, resource string, verb string) *authrpb.Permission {
	return &authrpb.Permission{
		ApiGroup: apiGroup,
		Resource: resource,
		Verb:     verb,
	}
}

func HobbyfarmPermission(resource string, verb string) *authrpb.Permission {
	return Permission(HobbyfarmGroup, resource, verb)
}

func RbacPermission(resource string, verb string) *authrpb.Permission {
	return Permission(RbacGroup, resource, verb)
}
