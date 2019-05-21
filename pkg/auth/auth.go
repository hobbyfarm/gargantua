package auth

import (
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	"k8s.io/client-go/tools/cache"
	"net/http"
)

const (
	userNameIndex = "authn.hobbyfarm.io/user-email-index"
)

type Auth struct {
	secret string

	userIndexer cache.Indexer

}

func NewAuth(hfInformerFactory hfInformers.SharedInformerFactory) (Auth, error) {
	a := Auth{}
	inf := hfInformerFactory.Hobbyfarm().V1().Users().Informer()
	indexers := map[string]cache.IndexFunc{userNameIndex: userNameIndexer}
	inf.AddIndexers(indexers)
	a.userIndexer = inf.GetIndexer()
	return a, nil
}


func (a Auth) AuthNFunc(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	email := r.PostFormValue("email")
	accessCode := r.PostFormValue("accessCode")


	glog.V(2).Infof("email and access code were passed in %s %s", email, accessCode)
	glog.V(2).Infof("authnfunc invoked")

}


func (a Auth) test(w http.ResponseWriter, r *http.Request) {
	obj, err := a.userIndexer.ByIndex(userNameIndex, "chris.kim@rancher.com")
	if err != nil {
		glog.Fatal(err)
	}

	if len(obj) < 1 {
		glog.Errorf("did not find user")
		return
	}
	user, ok := obj[0].(*hfv1.User)

	if ok {
		glog.V(2).Infof("Found user! the password was : %s", user.Spec.Password)
	}
}
func (a Auth) Setup(r *mux.Router) {
	r.HandleFunc("/auth/authenticate", a.AuthNFunc)
	r.HandleFunc("/auth/test", a.test)
	glog.V(2).Infof("set up route")
}

func userNameIndexer(obj interface{}) ([]string, error) {
	user, ok := obj.(*hfv1.User)
	if !ok {
		return []string{}, nil
	}
	return []string{user.Spec.Email}, nil
}