package authclient

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	hfv2 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v2"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbacclient"
	"k8s.io/client-go/tools/cache"
)

const (
	emailIndex = "authc.hobbyfarm.io/user-email-index"
)

type AuthClient struct {
	hfClientSet hfClientset.Interface
	userIndexer cache.Indexer
	rbacServer  *rbacclient.Client
}

func NewAuthClient(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, rbacServer *rbacclient.Client) (*AuthClient, error) {
	a := AuthClient{}
	a.hfClientSet = hfClientSet
	inf := hfInformerFactory.Hobbyfarm().V2().Users().Informer()
	indexers := map[string]cache.IndexFunc{emailIndex: emailIndexer}
	inf.AddIndexers(indexers)
	a.userIndexer = inf.GetIndexer()
	a.rbacServer = rbacServer
	return &a, nil
}

func emailIndexer(obj interface{}) ([]string, error) {
	user, ok := obj.(*hfv2.User)
	if !ok {
		return []string{}, nil
	}
	return []string{user.Spec.Email}, nil
}

func (a AuthClient) getUserByEmail(email string) (hfv2.User, error) {
	if len(email) == 0 {
		return hfv2.User{}, fmt.Errorf("email passed in was empty")
	}

	obj, err := a.userIndexer.ByIndex(emailIndex, email)
	if err != nil {
		return hfv2.User{}, fmt.Errorf("error while retrieving user by e-mail: %s with error: %v", email, err)
	}

	if len(obj) < 1 {
		return hfv2.User{}, fmt.Errorf("user not found by email: %s", email)
	}

	user, ok := obj[0].(*hfv2.User)

	if !ok {
		return hfv2.User{}, fmt.Errorf("error while converting user found by email to object: %s", email)
	}

	return *user, nil

}

func (a AuthClient) AuthWS(w http.ResponseWriter, r *http.Request) (hfv2.User, error) {
	token := r.URL.Query().Get("auth")

	if len(token) == 0 {
		glog.Errorf("no auth token passed in websocket query string")
		//util.ReturnHTTPMessage(w, r, 403, "forbidden", "no token passed")
		return hfv2.User{}, fmt.Errorf("authentication failed")
	}

	return a.performAuth(token)
}

// if admin is true then check if user is an admin

func (a AuthClient) performAuth(token string) (hfv2.User, error) {
	user, err := a.ValidateJWT(token)

	if err != nil {
		glog.Errorf("error validating user %v", err)
		return hfv2.User{}, fmt.Errorf("authentication failed")
	}

	return user, nil
}

func (a *AuthClient) VerifyRBAC(request *rbacclient.Request, user hfv2.User) (hfv2.User, error) {
	if request.GetOperator() == rbacclient.OperatorAnd {
		// operator AND, all need to match
		for _, p := range request.GetPermissions() {
			g, err := a.rbacServer.Grants(user.Name, p)
			if err != nil {
				return hfv2.User{}, err
			}

			if !g {
				return hfv2.User{}, fmt.Errorf("permission denied")
			}
		}
		// if we get here, AND has succeeded
		return user, nil
	} else {
		// operator OR, only one needs to match
		for _, p := range request.GetPermissions() {
			g, err := a.rbacServer.Grants(user.Name, p)
			if err != nil {
				return hfv2.User{}, err
			}

			if g {
				return user, nil
			}
		}
	}

	return hfv2.User{}, fmt.Errorf("permission denied")
}

func (a *AuthClient) AuthGrantWS(request *rbacclient.Request, w http.ResponseWriter, r *http.Request) (hfv2.User, error) {
	user, err := a.AuthWS(w, r)
	if err != nil {
		return user, err
	}

	return a.VerifyRBAC(request, user)
}

func (a *AuthClient) AuthGrant(request *rbacclient.Request, w http.ResponseWriter, r *http.Request) (hfv2.User, error) {
	user, err := a.AuthN(w, r)
	if err != nil {
		return user, err
	}

	return a.VerifyRBAC(request, user)
}

func (a AuthClient) AuthN(w http.ResponseWriter, r *http.Request) (hfv2.User, error) {
	token := r.Header.Get("Authorization")

	if len(token) == 0 {
		glog.Errorf("no bearer token passed")
		//util.ReturnHTTPMessage(w, r, 403, "forbidden", "no token passed")
		return hfv2.User{}, fmt.Errorf("authentication failed")
	}

	var finalToken string

	splitToken := strings.Split(token, "Bearer")
	finalToken = strings.TrimSpace(splitToken[1])

	return a.performAuth(finalToken)
}

func (a AuthClient) ValidateJWT(tokenString string) (hfv2.User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		var user hfv2.User
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			var err error
			user, err = a.getUserByEmail(fmt.Sprint(claims["email"]))
			if err != nil {
				glog.Errorf("could not find user that matched email %s", fmt.Sprint(claims["email"]))
				return hfv2.User{}, fmt.Errorf("could not find user that matched token %s", fmt.Sprint(claims["email"]))
			}
		}
		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(user.Spec.Password), nil
	})

	if err != nil {
		glog.Errorf("error while validating user: %v", err)
		return hfv2.User{}, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		user, err := a.getUserByEmail(fmt.Sprint(claims["email"]))
		if err != nil {
			return hfv2.User{}, err
		} else {
			return user, nil
		}
	}
	glog.Errorf("error while validating user")
	return hfv2.User{}, fmt.Errorf("error while validating user")
}
