package providers

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	hflabels "github.com/hobbyfarm/gargantua/v4/pkg/labels"
	"github.com/hobbyfarm/gargantua/v4/pkg/statuswriter"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apiserver/pkg/storage"
	"net/http"
	"time"
)

const (
	authFailed = "authentication failed"
)

type BasicAuthProvider struct {
	userLister    strategy.Lister
	secretGetter  strategy.Getter
	settingGetter strategy.Getter
}

func NewBasicAuthProvider(userLister strategy.Lister, secretGetter strategy.Getter, settingGetter strategy.Getter) BasicAuthProvider {
	return BasicAuthProvider{
		userLister:    userLister,
		secretGetter:  secretGetter,
		settingGetter: settingGetter,
	}
}

func (ba BasicAuthProvider) HandleLogin() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		up, err := ParseUsernamePasswordAuthRequest(request)

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

		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"userid":      u.Name,
			"principal":   "local://" + u.Name,
			"displayname": u.Spec.DisplayName,
			"nbf":         time.Now().String(),
			"exp":         timeout,
		})

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
