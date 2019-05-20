package main

import (
	"flag"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/pkg/auth"
	"net/http"
)

func main() {
	//var signal chan struct{}
	//signal = make(chan struct{})
	flag.Parse()
	glog.V(2).Infof("Starting")
	r := mux.NewRouter()

	a, err := auth.NewAuth("test")
	if err != nil {
		glog.Fatal(err)
	}

	a.Setup(r)

	http.ListenAndServe(":80", r)
}

