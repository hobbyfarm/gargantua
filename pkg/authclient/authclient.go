package authclient

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	"k8s.io/client-go/tools/cache"
	"net/http"
	"strings"
)

const (
	emailIndex = "authc.hobbyfarm.io/user-email-index"
)

type AuthClient struct {
	hfClientSet *hfClientset.Clientset
	userIndexer cache.Indexer
}

func NewAuthClient(hfClientSet *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*AuthClient, error) {
	a := AuthClient{}
	a.hfClientSet = hfClientSet
	inf := hfInformerFactory.Hobbyfarm().V1().Users().Informer()
	indexers := map[string]cache.IndexFunc{emailIndex: emailIndexer}
	inf.AddIndexers(indexers)
	a.userIndexer = inf.GetIndexer()
	return &a, nil
}

func emailIndexer(obj interface{}) ([]string, error) {
	user, ok := obj.(*hfv1.User)
	if !ok {
		return []string{}, nil
	}
	return []string{user.Spec.Email}, nil
}

func (a AuthClient) getUserByEmail(email string) (hfv1.User, error) {
	if len(email) == 0 {
		return hfv1.User{}, fmt.Errorf("email passed in was empty")
	}

	obj, err := a.userIndexer.ByIndex(emailIndex, email)
	if err != nil {
		return hfv1.User{}, fmt.Errorf("error while retrieving user by e-mail: %s with error: %v", email, err)
	}

	if len(obj) < 1 {
		return hfv1.User{}, fmt.Errorf("user not found by email: %s", email)
	}

	user, ok := obj[0].(*hfv1.User)

	if !ok {
		return hfv1.User{}, fmt.Errorf("error while converting user found by email to object: %s", email)
	}

	return *user, nil

}

func (a AuthClient) AuthWS(w http.ResponseWriter, r *http.Request) (hfv1.User, error) {
	token := r.URL.Query().Get("auth")

	if len(token) == 0 {
		glog.Errorf("no auth token passed in websocket query string")
		//util.ReturnHTTPMessage(w, r, 403, "forbidden", "no token passed")
		return hfv1.User{}, fmt.Errorf("authentication failed")
	}

	return a.performAuth(token, false)
}

// if admin is true then check if user is an admin

func (a AuthClient) performAuth(token string, admin bool) (hfv1.User, error) {
	//glog.V(2).Infof("token passed in was: %s", token)

	user, err := a.ValidateJWT(token)

	if err != nil {
		glog.Errorf("error validating user %v", err)
		//util.ReturnHTTPMessage(w, r, 403, "forbidden", "forbidden")
		return hfv1.User{}, fmt.Errorf("authentication failed")
	}

	glog.V(2).Infof("validated user %s!", user.Spec.Email)
	if admin {
		if user.Spec.Admin {
			return user, nil
		} else {
			glog.Errorf("AUDIT: User %s attempted to access an admin protected resource.", user.Spec.Email)
			return hfv1.User{}, fmt.Errorf("authentication failed")
		}
	}
	//util.ReturnHTTPMessage(w, r, 200, "success", "test successful. valid token")
	return user, nil
}

func (a AuthClient) AuthN(w http.ResponseWriter, r *http.Request) (hfv1.User, error) {
	token := r.Header.Get("Authorization")

	if len(token) == 0 {
		glog.Errorf("no bearer token passed")
		//util.ReturnHTTPMessage(w, r, 403, "forbidden", "no token passed")
		return hfv1.User{}, fmt.Errorf("authentication failed")
	}

	var finalToken string

	splitToken := strings.Split(token, "Bearer")
	finalToken = strings.TrimSpace(splitToken[1])

	return a.performAuth(finalToken, false)
}

func (a AuthClient) AuthNAdmin(w http.ResponseWriter, r *http.Request) (hfv1.User, error) {
	token := r.Header.Get("Authorization")

	if len(token) == 0 {
		glog.Errorf("no bearer token passed")
		//util.ReturnHTTPMessage(w, r, 403, "forbidden", "no token passed")
		return hfv1.User{}, fmt.Errorf("authentication failed")
	}

	var finalToken string

	splitToken := strings.Split(token, "Bearer")
	finalToken = strings.TrimSpace(splitToken[1])

	return a.performAuth(finalToken, true)
}

func (a AuthClient) ValidateJWT(tokenString string) (hfv1.User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		var user hfv1.User
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			var err error
			user, err = a.getUserByEmail(fmt.Sprint(claims["email"]))
			if err != nil {
				glog.Errorf("could not find user that matched token %s", fmt.Sprint(claims["email"]))
				return hfv1.User{}, fmt.Errorf("could not find user that matched token %s", fmt.Sprint(claims["email"]))
			}
		}
		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(user.Spec.Password), nil
	})

	if err != nil {
		glog.Errorf("error while validating user: %v", err)
		return hfv1.User{}, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		user, err := a.getUserByEmail(fmt.Sprint(claims["email"]))
		if err != nil {
			return hfv1.User{}, err
		} else {
			return user, nil
		}
	}
	glog.Errorf("error while validating user")
	return hfv1.User{}, fmt.Errorf("error while validating user")
}
