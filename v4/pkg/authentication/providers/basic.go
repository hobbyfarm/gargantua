package providers

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/statuswriter"
	"k8s.io/apimachinery/pkg/api/errors"
	"net/http"
)

type BasicAuthProvider struct {
}

func (ba BasicAuthProvider) HandleLogin() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		up, err := ParseUsernamePasswordAuthRequest(request)

		if err != nil {
			br := errors.NewBadRequest(err.Error())
			statuswriter.WriteError(br, writer)
			return
		}

	}
}
