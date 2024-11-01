package authentication

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/providers"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/providers/ldap"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/providers/local"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/token"
	"github.com/hobbyfarm/gargantua/v4/pkg/statuswriter"
	"github.com/hobbyfarm/mink/pkg/openapi"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type Server struct {
	credentialedProviders map[string]providers.CredentialedProvider
	callbackProviders     map[string]providers.CallbackProvider
	kclient               client.Client
	token.TokenGeneratorValidator
}

func RegisterHandlers(kclient client.Client, mux openapi.CanHandle) {
	genericTokenGV := token.NewGenericGeneratorValidator(kclient)

	basicProvider := local.NewProvider(kclient)
	ldapProvider := ldap.NewProvider(kclient)

	s := &Server{
		credentialedProviders: map[string]providers.CredentialedProvider{
			"local": basicProvider,
			"ldap":  ldapProvider,
		},
		TokenGeneratorValidator: genericTokenGV,
		kclient:                 kclient,
	}

	s.RegisterAuthenticators(mux)
}

func (s *Server) RegisterAuthenticators(mux openapi.CanHandle) {
	for k, v := range s.credentialedProviders {
		if clPv, ok := v.(providers.CredentialedProvider); ok {
			mux.Handle("/auth/"+k+"/login", s.handleCredentialedLogin(clPv))
		}
	}

	for k, v := range s.callbackProviders {
		if cv, ok := v.(providers.CallbackProvider); ok {
			mux.Handle("/auth"+k+"/callback", cv.HandleCallback())
		}
	}
}

func (s *Server) handleCredentialedLogin(handler providers.CredentialedProvider) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		creds, err := providers.ParseCredentials(req)
		if err != nil {
			statuswriter.WriteError(errors.NewBadRequest("invalid credentials"), writer)
			return
		}

		user, lErr := handler.HandleLogin(req.Context(), creds)
		if lErr != nil {
			statuswriter.WriteError(lErr, writer)
			return
		}

		// get expiration
		// if we get here, everything is good, user can be authenticated
		// so, now we need to issue them a token
		// how long should that token be valid for? Usually its 12 hours but let's check settings
		set := &v4alpha1.Setting{}
		err = s.kclient.Get(req.Context(), client.ObjectKey{Name: "user-token-expiration"}, set)

		var timeout = time.Now().Add(12 * time.Hour)
		if err == nil {
			anySet, err := set.FromJSON(set.Value)
			if err == nil {
				// err would be non-nil if empty
				intSet := anySet.(int)
				timeout = time.Now().Add(time.Duration(intSet) * time.Hour)
			}
		}

		// gen token for user
		tok, err := s.GenerateToken(user, timeout)
		if err != nil {
			statuswriter.WriteError(errors.NewInternalError(err), writer)
			return
		}

		statuswriter.WriteStatus(&metav1.Status{
			Status:  metav1.StatusSuccess,
			Message: tok,
			Reason:  "login successful",
			Details: nil,
			Code:    http.StatusOK,
		}, writer)
	}
}
