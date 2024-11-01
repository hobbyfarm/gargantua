package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/user"
	"io"
	"k8s.io/apimachinery/pkg/api/errors"
	"net/http"
)

type CallbackProvider interface {
	LoginHandler
	CallbackHandler
}

type CredentialedProvider interface {
	CredentialedLoginHandler
}

type TokenValidator func(token string) (*user.User, error)

type CredentialedLoginHandler interface {
	HandleLogin(ctx context.Context, creds *Credentials) (*user.User, *errors.StatusError)
}

type LoginHandler interface {
	HandleLogin() http.HandlerFunc
}

type CallbackHandler interface {
	HandleCallback() http.HandlerFunc
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func ParseCredentials(req *http.Request) (*Credentials, error) {
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	var out = &Credentials{}
	if err := json.Unmarshal(bodyBytes, out); err != nil {
		return nil, err
	}

	if out.Username == "" {
		return nil, fmt.Errorf("missing username field in request body")
	}

	if out.Password == "" {
		return nil, fmt.Errorf("missing password field in request body")
	}

	return out, nil
}
