package auth

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

func (a Auth) Setup(r *mux.Router) {

	r.HandleFunc("/auth/authenticate", a.AuthNFunc)
	glog.V(2).Infof("set up route")
}