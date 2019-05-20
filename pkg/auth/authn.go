package auth

import (
	"fmt"
	"github.com/golang/glog"
	"net/http"
)

type Auth struct {
	secret string
}

func NewAuth(inputSecret string) (Auth, error) {

	if inputSecret == "" {
		return Auth{}, fmt.Errorf("secret passed in was empty")
	}
	a := Auth{}

	a.secret = inputSecret

	return a, nil
}

func (a Auth) AuthNFunc(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	email := r.PostFormValue("email")
	accessCode := r.PostFormValue("accessCode")


	glog.V(2).Infof("email and access code were passed in %s %s", email, accessCode)
	glog.V(2).Infof("authnfunc invoked")

}