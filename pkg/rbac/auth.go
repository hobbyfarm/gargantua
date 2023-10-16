package rbac

import (
	"net/http"

	"github.com/golang/glog"
	"github.com/hobbyfarm/gargantua/v3/pkg/microservices"
	"github.com/hobbyfarm/gargantua/v3/protos/authn"
	"github.com/hobbyfarm/gargantua/v3/protos/authr"
	userProto "github.com/hobbyfarm/gargantua/v3/protos/user"
)

func AuthenticateRequest(r *http.Request, caCertPath string) (*userProto.User, error) {
	authnConn, err := microservices.EstablishConnection(microservices.AuthN, caCertPath)
	if err != nil {
		glog.Error("failed connecting to service authn-service")
		return nil, err
	}
	defer authnConn.Close()

	authnClient := authn.NewAuthNClient(authnConn)
	token := r.Header.Get("Authorization")
	return authnClient.AuthN(r.Context(), &authn.AuthNRequest{Token: token})
}

func AuthenticateWS(r *http.Request, caCertPath string) (*userProto.User, error) {
	authnConn, err := microservices.EstablishConnection(microservices.AuthN, caCertPath)
	if err != nil {
		glog.Error("failed connecting to service authn-service")
		return nil, err
	}
	defer authnConn.Close()

	authnClient := authn.NewAuthNClient(authnConn)
	token := "Bearer " + r.URL.Query().Get("auth")
	return authnClient.AuthN(r.Context(), &authn.AuthNRequest{Token: token})
}

func AuthorizeSimple(r *http.Request, caCertPath string, username string, permission *authr.Permission) (*authr.AuthRResponse, error) {
	authrConn, err := microservices.EstablishConnection(microservices.AuthR, caCertPath)
	if err != nil {
		glog.Error("failed connecting to service authr-service")
		return nil, err
	}
	defer authrConn.Close()

	authrClient := authr.NewAuthRClient(authrConn)
	rbacPermissions := []*authr.Permission{
		permission,
	}
	rbacRq := &authr.RbacRequest{
		Permissions: rbacPermissions,
	}
	return authrClient.AuthR(r.Context(), &authr.AuthRRequest{UserName: username, Request: rbacRq})
}

func Authorize(r *http.Request, caCertPath string, username string, permissions []*authr.Permission, operator string) (*authr.AuthRResponse, error) {
	authrConn, err := microservices.EstablishConnection(microservices.AuthR, caCertPath)
	if err != nil {
		glog.Error("failed connecting to service authr-service")
		return nil, err
	}
	defer authrConn.Close()

	authrClient := authr.NewAuthRClient(authrConn)
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
