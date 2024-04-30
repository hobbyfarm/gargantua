package basic

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/providers"
	hflabels "github.com/hobbyfarm/gargantua/v4/pkg/labels"
	"github.com/hobbyfarm/gargantua/v4/pkg/statuswriter"
	hfStrategy "github.com/hobbyfarm/gargantua/v4/pkg/stores/strategy"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/storage"
	"net/http"
	"strings"
	"time"
)

const (
	authFailed = "authentication failed"
)

var _ authenticator.Request = (*BasicAuthProvider)(nil)

type Claims struct {
	jwt.StandardClaims
	Principal   string `json:"principal"`
	DisplayName string `json:"displayName"`
}

type BasicAuthProvider struct {
	userLister    hfStrategy.ListerGetter
	secretGetter  strategy.Getter
	settingGetter strategy.Getter
}

func NewBasicAuthProvider(userLister hfStrategy.ListerGetter, secretGetter strategy.Getter, settingGetter strategy.Getter) BasicAuthProvider {
	return BasicAuthProvider{
		userLister:    userLister,
		secretGetter:  secretGetter,
		settingGetter: settingGetter,
	}
}

func (ba BasicAuthProvider) HandleLogin() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		up, err := providers.ParseCredentials(request)

		if err != nil {
			br := errors.NewBadRequest(err.Error())
			statuswriter.WriteError(br, writer)
			return
		}

		// lookup user
		list, err := ba.userLister.List(request.Context(), util.GetReleaseNamespace(), storage.ListOptions{
			Predicate: storage.SelectionPredicate{
				Label: labels.SelectorFromSet(map[string]string{
					hflabels.UsernameLabel: up.Username,
				}),
			},
		})

		if err != nil {
			// TODO: Add logging/tracing hook here
			statuswriter.WriteError(errors.NewUnauthorized(authFailed), writer)
			return
		}

		userList := list.(*v4alpha1.UserList)
		if len(userList.Items) == 0 {
			// TODO: Add logging/tracing hook here
			statuswriter.WriteError(errors.NewUnauthorized(authFailed), writer)
			return
		}

		if len(userList.Items) > 1 {
			// should only match one user for the username
			// auth has to fail because we don't know which one is correct
			// TODO: Add logging/tracing hook here
			statuswriter.WriteError(errors.NewUnauthorized(authFailed), writer)
			return
		}

		// here we should have a single user
		u := userList.Items[0]

		// on the user, check that the username matches (in case label lookup was wrong)
		if u.Spec.LocalAuthDetails.Username != up.Username {
			// TODO: Add logging/tracing hook here
			statuswriter.WriteError(errors.NewUnauthorized(authFailed), writer)
			return
		}

		// get password
		obj, err := ba.secretGetter.Get(request.Context(), util.GetReleaseNamespace(), u.Spec.LocalAuthDetails.PasswordSecret)
		if err != nil {
			// TODO: Add logging/tracing hook here
			statuswriter.WriteError(errors.NewUnauthorized(authFailed), writer)
			return
		}

		secret := obj.(*v4alpha1.Secret)
		hashedPw, ok := secret.Data["password"]
		if !ok || len(hashedPw) == 0 {
			// TODO: Add logging/tracing hook here
			statuswriter.WriteError(errors.NewUnauthorized(authFailed), writer)
			return
		}

		if err := bcrypt.CompareHashAndPassword(hashedPw, []byte(up.Password)); err != nil {
			// TODO: Add logging/tracing hook here
			statuswriter.WriteError(errors.NewUnauthorized(authFailed), writer)
			return
		}

		// if we get here, everything is good, user can be authenticated
		// so now we need to issue them a token
		// how long should that token be valid for? Usually its 12 hours but lets check settings
		obj, err = ba.settingGetter.Get(request.Context(), util.GetReleaseNamespace(), "user-token-expiration")

		var timeout = time.Now().Add(12 * time.Hour)
		if err == nil {
			set := obj.(*v4alpha1.Setting)
			anySet, err := set.FromJSON(set.Value)
			if err == nil {
				// err would be non-nil if empty
				intSet := anySet.(int)
				timeout = time.Now().Add(time.Duration(intSet) * time.Hour)
			}
		}

		claims := Claims{
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: timeout.Unix(),
				IssuedAt:  time.Now().Unix(),
				Issuer:    "hobbyfarm",
				NotBefore: time.Now().Unix(),
				Subject:   u.Name,
			},
			Principal:   "local://" + u.Name,
			DisplayName: u.Spec.DisplayName,
		}

		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

		tokenString, err := tok.SignedString(hashedPw)

		statuswriter.WriteStatus(&metav1.Status{
			Status:  metav1.StatusSuccess,
			Message: tokenString,
			Reason:  "login successful",
			Details: nil,
			Code:    http.StatusOK,
		}, writer)
	}
}

func (ba BasicAuthProvider) validateToken(ctx context.Context, tok string) (*BasicUser, bool) {
	var uid string
	token, err := jwt.ParseWithClaims(tok, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		claims := token.Claims.(*Claims)
		// get user for verification
		obj, err := ba.userLister.Get(ctx, "", claims.Subject)
		if err != nil {
			return nil, fmt.Errorf("could not find user")
		}
		user := obj.(*v4alpha1.User)

		obj, err = ba.secretGetter.Get(ctx, "", user.Spec.LocalAuthDetails.PasswordSecret)
		if err != nil {
			return nil, fmt.Errorf("password lookup error")
		}
		sec := obj.(*v4alpha1.Secret)

		uid = string(user.UID)
		return sec.Data["password"], nil
	})

	if err != nil {
		return nil, false
	}

	return &BasicUser{
		name:   token.Claims.(*Claims).Subject,
		uid:    uid,
		groups: nil,
	}, true
}

func (ba BasicAuthProvider) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {
	tok := req.Header.Get("Authorization")
	if !strings.HasPrefix(tok, "Bearer ") {
		return nil, false, fmt.Errorf("token error")
	}

	tok = strings.TrimPrefix(tok, "Bearer ")
	if u, valid := ba.validateToken(req.Context(), tok); valid {
		return &authenticator.Response{
			User: u,
		}, true, nil
	}

	return nil, false, nil
}
