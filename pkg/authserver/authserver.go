package authserver

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/client-go/tools/cache"
	"net/http"
	"strings"
)

const (
	emailIndex = "authn.hobbyfarm.io/user-email-index"
)

type AuthServer struct {
	hfClientSet *hfClientset.Clientset
	userIndexer cache.Indexer
}

func NewAuthServer(hfClientSet *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (AuthServer, error) {
	a := AuthServer{}
	a.hfClientSet = hfClientSet
	inf := hfInformerFactory.Hobbyfarm().V1().Users().Informer()
	indexers := map[string]cache.IndexFunc{emailIndex: emailIndexer}
	inf.AddIndexers(indexers)
	a.userIndexer = inf.GetIndexer()
	return a, nil
}

func (a AuthServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/auth/registerwithaccesscode", a.RegisterWithAccessCodeFunc).Methods("POST")
	r.HandleFunc("/auth/authenticate", a.AuthNFunc).Methods("POST")
	//r.HandleFunc("/auth/test", a.AuthN)
	glog.V(2).Infof("set up route")
}

func (a AuthServer) AuthN(w http.ResponseWriter, r *http.Request) error {
	var finalToken string
	token := r.Header.Get("Authorization")

	if len(token) == 0 {
		glog.Errorf("no bearer token passed")
		//util.ReturnHTTPMessage(w, r, 403, "forbidden", "no token passed")
		return fmt.Errorf("authentication failed")
	}

	splitToken := strings.Split(token, "Bearer")
	finalToken = strings.TrimSpace(splitToken[1])

	glog.V(2).Infof("token passed in was: %s", finalToken)

	user, err := a.ValidateJWT(finalToken)

	if err != nil {
		glog.Errorf("error validating user %v", err)
		//util.ReturnHTTPMessage(w, r, 403, "forbidden", "forbidden")
		return fmt.Errorf("authentication failed")
	}

	glog.V(2).Infof("validated user %s!", user.Spec.Email)

	//util.ReturnHTTPMessage(w, r, 200, "success", "test successful. valid token")
	return nil
}

func emailIndexer(obj interface{}) ([]string, error) {
	user, ok := obj.(*hfv1.User)
	if !ok {
		return []string{}, nil
	}
	return []string{user.Spec.Email}, nil
}

func (a AuthServer) getUserByEmail(email string) (hfv1.User, error) {
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

// takes in parameters:
//	access_code: access code
//  email: e-mail
//  password: password (raw)
//
// spits out json with status:
//

func (a AuthServer) RegisterWithAccessCodeFunc(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	email := r.PostFormValue("email")
	access_code := r.PostFormValue("access_code")
	password := r.PostFormValue("password")
	// should we reconcile based on the access code posted in? nah
	_, err := a.getUserByEmail(email)

	if len(email) == 0 || len(access_code) == 0 || len(password) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "error", "invalid input. required fields: email, access_code, password")
		return
	}

	if err == nil {
		// the user was found, we should return info
		util.ReturnHTTPMessage(w, r, 409, "error", "user already exists")
		return
	}

	newUser := hfv1.User{}

	hasher := sha256.New()
	hasher.Write([]byte(email))
	sha := base32.StdEncoding.WithPadding(-1).EncodeToString(hasher.Sum(nil))[:10]
	id := "u-" + strings.ToLower(sha)
	newUser.Name = id
	newUser.Spec.Id = id
	accessCodes := make([]string, 1)
	accessCodes[0] = access_code
	newUser.Spec.AccessCodes = accessCodes
	newUser.Spec.Email = email

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		glog.Errorf("error while hashing password for email %s", email)
		util.ReturnHTTPMessage(w, r, 500, "error", "error working on password")
		return
	}

	newUser.Spec.Password = string(passwordHash)

	_, err = a.hfClientSet.HobbyfarmV1().Users().Create(&newUser)

	if err != nil {
		glog.Errorf("error creating new user for email %s: %v", email, err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error creating user")
		return
	}

	glog.V(2).Infof("created user %s", email)
	util.ReturnHTTPMessage(w, r, 201, "info", "created user")
}

func (a AuthServer) AuthNFunc(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	email := r.PostFormValue("email")
	password := r.PostFormValue("password")

	user, err := a.getUserByEmail(email)

	if err != nil {
		glog.Errorf("there was an error retrieving the user %s: %v", email, err)
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "login failed")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Spec.Password), []byte(password))

	if err != nil {
		glog.Errorf("password incorrect for user %s: %v", email, err)
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "login failed")
		return
	}

	token, err := GenerateJWT(user)

	if err != nil {
		glog.Error(err)
	}

	util.ReturnHTTPMessage(w, r, 200, "authorized", token)
}

func GenerateJWT(user hfv1.User) (string, error) {

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": user.Spec.Email,
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(user.Spec.Password))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (a AuthServer) ValidateJWT(tokenString string) (hfv1.User, error) {
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
