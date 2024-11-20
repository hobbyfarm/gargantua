package mux

import "net/http"

type Mux interface {
	HandleFunc(path string, handlerFunc http.HandlerFunc)
	HandlePrefix(path string, handler http.Handler)
	Handle(path string, handler http.Handler)
}
