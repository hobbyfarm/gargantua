package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type LoginHandler interface {
	HandleLogin() http.HandlerFunc
}

type UsernamePasswordAuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func ParseUsernamePasswordAuthRequest(req *http.Request) (*UsernamePasswordAuthRequest, error) {
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	var out = &UsernamePasswordAuthRequest{}
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
