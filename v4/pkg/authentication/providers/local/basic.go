package local

import (
	"context"
	"github.com/dgrijalva/jwt-go"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/providers"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/user"
	hflabels "github.com/hobbyfarm/gargantua/v4/pkg/labels"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	authFailed = "authentication failed"
)

var _ providers.CredentialedProvider = (*Provider)(nil)

type Claims struct {
	jwt.StandardClaims
	Principal   string `json:"principal"`
	DisplayName string `json:"displayName"`
}

type Provider struct {
	kclient client.Client
}

func NewProvider(kclient client.Client) *Provider {
	return &Provider{
		kclient: kclient,
	}
}

func (ba Provider) HandleLogin(ctx context.Context, creds *providers.Credentials) (*user.User, *errors.StatusError) {
	// lookup user
	userList := &v4alpha1.UserList{}
	err := ba.kclient.List(ctx, userList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			hflabels.UsernameLabel: creds.Username,
		}),
	})
	if err != nil {
		logrus.Error(err)
		// TODO: Add logging/tracing hook here
		return nil, errors.NewUnauthorized(authFailed)
	}

	if len(userList.Items) == 0 {
		// TODO: Add logging/tracing hook here
		return nil, errors.NewUnauthorized(authFailed)
	}

	if len(userList.Items) > 1 {
		// should only match one user for the username
		// auth has to fail because we don't know which one is correct
		// TODO: Add logging/tracing hook here
		return nil, errors.NewUnauthorized(authFailed)
	}

	// here we should have a single user
	u := userList.Items[0]

	// on the user, check that the username matches (in case label lookup was wrong)
	if u.Spec.LocalAuthDetails.Username != creds.Username {
		// TODO: Add logging/tracing hook here
		return nil, errors.NewUnauthorized(authFailed)
	}

	// get password
	secret := &v4alpha1.Secret{}
	err = ba.kclient.Get(ctx, client.ObjectKey{Name: u.Spec.LocalAuthDetails.PasswordSecret}, secret)
	if err != nil {
		// TODO: Add logging/tracing hook here
		return nil, errors.NewUnauthorized(authFailed)
	}

	hashedPw, ok := secret.Data["password"]
	if !ok || len(hashedPw) == 0 {
		// TODO: Add logging/tracing hook here
		return nil, errors.NewUnauthorized(authFailed)
	}

	if err := bcrypt.CompareHashAndPassword(hashedPw, []byte(creds.Password)); err != nil {
		// TODO: Add logging/tracing hook here
		return nil, errors.NewUnauthorized(authFailed)
	}

	return user.FromV4Alpha1User(&u), nil
}
