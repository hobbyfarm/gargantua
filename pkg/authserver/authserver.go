package authserver

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hobbyfarm/gargantua/pkg/settingclient"

	"github.com/hobbyfarm/gargantua/pkg/accesscode"
	"github.com/hobbyfarm/gargantua/pkg/rbacclient"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv2 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v2"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/errors"
	"github.com/hobbyfarm/gargantua/pkg/util"
	"golang.org/x/crypto/bcrypt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

const (
	emailIndex = "authn.hobbyfarm.io/user-email-index"
)

type AuthServer struct {
	auth             *authclient.AuthClient
	rbac             *rbacclient.Client
	hfClientSet      hfClientset.Interface
	accessCodeClient *accesscode.AccessCodeClient
	ctx              context.Context
}

func NewAuthServer(authClient *authclient.AuthClient, hfClientSet hfClientset.Interface, ctx context.Context, acClient *accesscode.AccessCodeClient, rbac *rbacclient.Client) (AuthServer, error) {
	a := AuthServer{}
	a.auth = authClient
	a.hfClientSet = hfClientSet
	a.ctx = ctx
	a.rbac = rbac
	a.accessCodeClient = acClient
	return a, nil
}

func (a AuthServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/auth/registerwithaccesscode", a.RegisterWithAccessCodeFunc).Methods("POST")
	r.HandleFunc("/auth/accesscode", a.ListAccessCodeFunc).Methods("GET")
	r.HandleFunc("/auth/accesscode", a.AddAccessCodeFunc).Methods("POST")
	r.HandleFunc("/auth/scheduledevents", a.ListScheduledEventsFunc).Methods("GET")
	r.HandleFunc("/auth/accesscode/{access_code}", a.RemoveAccessCodeFunc).Methods("DELETE")
	r.HandleFunc("/auth/changepassword", a.ChangePasswordFunc).Methods("POST")
	r.HandleFunc("/auth/settings", a.RetreiveSettingsFunc).Methods("GET")
	r.HandleFunc("/auth/settings", a.UpdateSettingsFunc).Methods("POST")
	r.HandleFunc("/auth/authenticate", a.AuthNFunc).Methods("POST")
	r.HandleFunc("/auth/access", a.GetAccessSet).Methods("GET")
	glog.V(2).Infof("set up route")
}

func (a AuthServer) getUserByEmail(email string) (hfv2.User, error) {
	if len(email) == 0 {
		return hfv2.User{}, fmt.Errorf("email passed in was empty")
	}

	users, err := a.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).List(a.ctx, metav1.ListOptions{})

	if err != nil {
		return hfv2.User{}, fmt.Errorf("error while retrieving user list")
	}

	for _, user := range users.Items {
		if user.Spec.Email == email {
			return user, nil
		}
	}

	return hfv2.User{}, fmt.Errorf("user not found")

}

// takes in parameters:
//	access_code: access code
//  email: e-mail
//  password: password (raw)
//
// spits out json with status:
//

func (a AuthServer) NewUser(email string, password string) (string, error) {

	if len(email) == 0 || len(password) == 0 {
		return "", fmt.Errorf("error creating user, email or password field blank")
	}

	_, err := a.getUserByEmail(email)

	if err == nil {
		// the user was found, we should return info
		return "", errors.NewAlreadyExists("user already exists")
	}

	newUser := hfv2.User{}

	hasher := sha256.New()
	hasher.Write([]byte(email))
	sha := base32.StdEncoding.WithPadding(-1).EncodeToString(hasher.Sum(nil))[:10]
	id := "u-" + strings.ToLower(sha)
	newUser.Name = id
	newUser.Spec.Email = email

	settings := make(map[string]string)
	settings["terminal_theme"] = "default"

	newUser.Spec.Settings = settings

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("error while hashing password for email %s", email)
	}

	newUser.Spec.Password = string(passwordHash)

	_, err = a.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Create(a.ctx, &newUser, metav1.CreateOptions{})

	if err != nil {
		return "", fmt.Errorf("error creating user")
	}

	return id, nil
}

func (a AuthServer) ChangePasswordFunc(w http.ResponseWriter, r *http.Request) {
	user, err := a.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to change password")
		return
	}

	r.ParseForm()

	oldPassword := r.PostFormValue("old_password")
	newPassword := r.PostFormValue("new_password")

	err = a.ChangePassword(user.Name, oldPassword, newPassword)

	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", fmt.Sprintf("error changing password for user %s", user.Name))
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "success", fmt.Sprintf("password changed"))

	glog.V(2).Infof("changed password for user %s", user.Spec.Email)
}

func (a AuthServer) UpdateSettingsFunc(w http.ResponseWriter, r *http.Request) {
	user, err := a.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update settings")
		return
	}

	r.ParseForm()

	newSettings := make(map[string]string)
	for key, _ := range r.Form {
		newSettings[key] = r.FormValue(key) //Ignore when multiple values were set for one argument. Just take the first one
	}

	err = a.UpdateSettings(user.Name, newSettings)

	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", fmt.Sprintf("error updating settings for user %s", user.Name))
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "success", fmt.Sprintf("settings updated"))

	glog.V(2).Infof("settings updated for user %s", user.Spec.Email)
}

func (a AuthServer) ListAccessCodeFunc(w http.ResponseWriter, r *http.Request) {
	user, err := a.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get accesscode")
		return
	}

	encodedACList, err := json.Marshal(user.Spec.AccessCodes)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedACList)

	glog.V(2).Infof("retrieved accesscode list for user %s", user.Spec.Email)
}

func (a AuthServer) ListScheduledEventsFunc(w http.ResponseWriter, r *http.Request) {
	user, err := a.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list suitable scheduledevents")
		return
	}

	accessCodes, err := a.accessCodeClient.GetAccessCodes(user.Spec.AccessCodes)

	accessCodeScheduledEvent := make(map[string]string)

	//Getting single SEs should be faster than listing all of them and iterating them in O(n^2), in most cases users only have a hand full of accessCodes.
	for _, ac := range accessCodes {
		se, err := a.hfClientSet.HobbyfarmV2().ScheduledEvents(util.GetReleaseNamespace()).Get(a.ctx, ac.Labels[util.ScheduledEventLabel], metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			continue
		}
		accessCodeScheduledEvent[ac.Spec.Code] = se.Spec.Name
	}

	encodedMap, err := json.Marshal(accessCodeScheduledEvent)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedMap)
}

func (a AuthServer) RetreiveSettingsFunc(w http.ResponseWriter, r *http.Request) {
	user, err := a.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get settings")
		return
	}

	latestUser, err := a.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Get(a.ctx, user.Name, metav1.GetOptions{})

	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", fmt.Sprintf("error retrieving user %s", user.Name))
		return
	}

	encodedSettings, err := json.Marshal(latestUser.Spec.Settings)

	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedSettings)

	glog.V(2).Infof("retrieved settings list for user %s", user.Spec.Email)
}

func (a AuthServer) AddAccessCodeFunc(w http.ResponseWriter, r *http.Request) {
	user, err := a.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get accesscode")
		return
	}

	r.ParseForm()

	accessCode := strings.ToLower(r.PostFormValue("access_code"))

	err = a.AddAccessCode(user.Name, accessCode)

	if err != nil {
		glog.Error(err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error adding access code")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "success", accessCode)

	glog.V(2).Infof("added accesscode %s to user %s", accessCode, user.Spec.Email)
}

func (a AuthServer) RemoveAccessCodeFunc(w http.ResponseWriter, r *http.Request) {
	user, err := a.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get accesscode")
		return
	}

	vars := mux.Vars(r)

	accessCode := strings.ToLower(vars["access_code"])

	err = a.RemoveAccessCode(user.Name, accessCode)

	if err != nil {
		glog.Error(err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error removing access code")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "success", accessCode)

	glog.V(2).Infof("removed accesscode %s to user %s", accessCode, user.Spec.Email)
}

func (a AuthServer) AddAccessCode(userId string, accessCode string) error {
	if len(userId) == 0 || len(accessCode) == 0 {
		return fmt.Errorf("bad parameters passed, %s:%s", userId, accessCode)
	}

	accessCode = strings.ToLower(accessCode)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		user, err := a.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Get(a.ctx, userId, metav1.GetOptions{})

		if err != nil {
			return fmt.Errorf("error retrieving user")
		}

		if len(user.Spec.AccessCodes) == 0 {
			user.Spec.AccessCodes = []string{}
		} else {
			for _, ac := range user.Spec.AccessCodes {
				if ac == accessCode {
					return fmt.Errorf("access code already added to user")
				}
			}
		}

		user.Spec.AccessCodes = append(user.Spec.AccessCodes, accessCode)

		_, updateErr := a.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Update(a.ctx, user, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return retryErr
	}

	return nil
}

func (a AuthServer) RemoveAccessCode(userId string, accessCode string) error {
	if len(userId) == 0 || len(accessCode) == 0 {
		return fmt.Errorf("bad parameters passed, %s:%s", userId, accessCode)
	}

	accessCode = strings.ToLower(accessCode)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		user, err := a.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Get(a.ctx, userId, metav1.GetOptions{})

		if err != nil {
			return fmt.Errorf("error retrieving user %s", userId)
		}

		var newAccessCodes []string

		newAccessCodes = make([]string, 0)

		if len(user.Spec.AccessCodes) == 0 {
			// there were no access codes at this point so what are we doing
			return fmt.Errorf("accesscode %s for user %s was not found", accessCode, userId)
		} else {
			found := false
			for _, ac := range user.Spec.AccessCodes {
				if ac == accessCode {
					found = true
				} else {
					newAccessCodes = append(newAccessCodes, ac)
				}
			}
			if !found {
				// the access code wasn't found so no update required
				return nil
			}
		}

		user.Spec.AccessCodes = newAccessCodes

		_, updateErr := a.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Update(a.ctx, user, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return retryErr
	}

	return nil
}

func (a AuthServer) ChangePassword(userId string, oldPassword string, newPassword string) error {
	if len(userId) == 0 || len(oldPassword) == 0 || len(newPassword) == 0 {
		return fmt.Errorf("bad parameters passed, %s", userId)
	}

	user, err := a.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Get(a.ctx, userId, metav1.GetOptions{})
	if err != nil {
		glog.Errorf("error retrieving user: %v", err)
		return fmt.Errorf("error retrieving user")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Spec.Password), []byte(oldPassword))

	if err != nil {
		glog.Errorf("old password incorrect for user ID %s: %v", userId, err)
		return fmt.Errorf("bad password change")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("error while hashing password for email %s", user.Spec.Email)
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		user, err := a.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Get(a.ctx, userId, metav1.GetOptions{})

		if err != nil {
			return fmt.Errorf("error retrieving user")
		}

		user.Spec.Password = string(passwordHash)

		_, updateErr := a.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Update(a.ctx, user, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return retryErr
	}

	return nil
}

func (a AuthServer) UpdateSettings(userId string, newSettings map[string]string) error {
	if len(userId) == 0 {
		return fmt.Errorf("bad parameters passed, %s", userId)
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		user, err := a.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Get(a.ctx, userId, metav1.GetOptions{})

		if err != nil {
			return fmt.Errorf("error retrieving user")
		}

		user.Spec.Settings = newSettings

		_, updateErr := a.hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()).Update(a.ctx, user, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return retryErr
	}

	return nil
}

func (a AuthServer) RegisterWithAccessCodeFunc(w http.ResponseWriter, r *http.Request) {
	if set := settingclient.GetSetting(settingclient.SettingRegistrationDisabled); set == nil {
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error performing registration")
		return
	} else {
		if set.(bool) {
			util.ReturnHTTPMessage(w, r, http.StatusConflict, "disabled", "registration disabled")
			return
		}
	}

	r.ParseForm()

	email := r.PostFormValue("email")
	accessCode := strings.ToLower(r.PostFormValue("access_code"))
	password := r.PostFormValue("password")
	// should we reconcile based on the access code posted in? nah

	if len(email) == 0 || len(accessCode) == 0 || len(password) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "error", "invalid input. required fields: email, access_code, password")
		return
	}

	userId, err := a.NewUser(email, password)

	if err != nil {
		var msg string
		var code = 400
		if errors.IsAlreadyExists(err) {
			code = 409
			msg = err.Error()
		} else {
			glog.Errorf("error creating user %s %v", email, err)
			msg = "error creating user"
		}

		util.ReturnHTTPMessage(w, r, code, "error", msg)
		return
	}

	err = a.AddAccessCode(userId, accessCode)

	if err != nil {
		glog.Errorf("error creating user %s %v", email, err)
		util.ReturnHTTPMessage(w, r, 400, "error", "error creating user")
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

func GenerateJWT(user hfv2.User) (string, error) {

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": user.Spec.Email,
		"nbf":   time.Now().Unix(),                     // not valid before now
		"exp":   time.Now().Add(time.Hour * 24).Unix(), // expire in 24 hours
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(user.Spec.Password))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (a AuthServer) ValidateJWT(tokenString string) (hfv2.User, error) {
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
				glog.Errorf("could not find user that matched token %s", fmt.Sprint(claims["email"]))
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

func (a *AuthServer) GetAccessSet(w http.ResponseWriter, r *http.Request) {
	user, err := a.auth.AuthN(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	// need to get the user's access set and publish to front end
	as, err := a.rbac.GetAccessSet(user.Name)
	if err != nil {
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error fetching access set")
		glog.Error(err)
		return
	}

	encodedAS, err := json.Marshal(as)
	if err != nil {
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "internalerror", "internal error encoding access set")
		glog.Error(err)
		return
	}

	util.ReturnHTTPContent(w, r, http.StatusOK, "access_set", encodedAS)
}
