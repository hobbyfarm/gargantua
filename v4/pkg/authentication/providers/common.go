package providers

import (
	"encoding/json"
	"fmt"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/user"
	"io"
	"net/http"
)

const Unauthorized = "unauthorized"

type TokenValidator func(token string) (*user.User, error)

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
