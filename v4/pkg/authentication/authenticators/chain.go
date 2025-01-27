package authenticators

import (
	"fmt"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"net/http"
)

type ChainAuthenticator struct {
	authenticators []authenticator.Request
}

func NewChainAuthenticator(authenticators ...authenticator.Request) *ChainAuthenticator {
	return &ChainAuthenticator{authenticators: authenticators}
}

func (ca ChainAuthenticator) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {
	for _, a := range ca.authenticators {
		res, ok, err := a.AuthenticateRequest(req)
		if !ok || err != nil {
			// try the next one
			continue
		} else {
			return res, ok, err
		}
	}

	return nil, false, fmt.Errorf("authentication failed")
}
