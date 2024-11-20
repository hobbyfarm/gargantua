package local

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/authenticators/token"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/providers"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/user"
	hflabels "github.com/hobbyfarm/gargantua/v4/pkg/labels"
	"github.com/hobbyfarm/gargantua/v4/pkg/statuswriter"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/apimachinery/pkg/api/errors"
	"log/slog"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	authFailed = "authentication failed"
)

type Claims struct {
	jwt.StandardClaims
	Principal   string `json:"principal"`
	DisplayName string `json:"displayName"`
}

type Provider struct {
	kclient   client.Client
	userCache cache.Cache
	token.TokenGeneratorValidator
	*mux.Router
}

func New(kclient client.Client, userCache cache.Cache, tok token.TokenGeneratorValidator, router *mux.Router) *Provider {
	p := &Provider{
		kclient:                 kclient,
		userCache:               userCache,
		TokenGeneratorValidator: tok,
		Router:                  router,
	}

	p.HandleFunc("/login", p.HandleLogin)

	return p
}

func Indexers() map[string]client.IndexerFunc {
	return map[string]client.IndexerFunc{
		hflabels.LocalPrincipalKey: usernameIndexer,
	}
}

func (ba Provider) HandleLogin(w http.ResponseWriter, r *http.Request) {
	creds, err := providers.ParseCredentials(r)
	if err != nil {
		slog.Info("error parsing credentials", "error", err.Error())
		statuswriter.WriteError(errors.NewUnauthorized("invalid credentials"), w)
		return
	}

	// lookup user
	u, err := ba.findUser(r.Context(), creds.Username)
	if err != nil {
		statuswriter.WriteError(errors.NewUnauthorized(err.Error()), w)
		return
	}

	// on the user, check that the username matches (in case label lookup was wrong)
	if u.Spec.LocalAuthDetails.Username != creds.Username {
		slog.Error("hf list returned user with mismatched username", "hf-user-resource", u.Name,
			"hf-user-username", u.Spec.LocalAuthDetails.Username, "request-username", creds.Username)
		statuswriter.WriteError(errors.NewUnauthorized(providers.Unauthorized), w)
		return
	}

	// get password
	secret := &v4alpha1.Secret{}
	err = ba.kclient.Get(r.Context(), client.ObjectKey{Name: u.Spec.LocalAuthDetails.PasswordSecret}, secret)
	if err != nil {
		slog.Error("error getting password secret for user", "user", u.Name, "secret-name", u.Spec.LocalAuthDetails.PasswordSecret)
		statuswriter.WriteError(errors.NewUnauthorized(providers.Unauthorized), w)
		return
	}

	hashedPw, ok := secret.Data["password"]
	if !ok || len(hashedPw) == 0 {
		slog.Error("local user password secret contains invalid data", "user", u.Name,
			"secret-name", secret.Name)
		statuswriter.WriteError(errors.NewUnauthorized(providers.Unauthorized), w)
		return
	}

	if err := bcrypt.CompareHashAndPassword(hashedPw, []byte(creds.Password)); err != nil {
		slog.Info("invalid username/password for user", "user", creds.Username)
		statuswriter.WriteError(errors.NewUnauthorized(providers.Unauthorized), w)
		return
	}

	// valid user, issue token
	tok, err := ba.GenerateToken(user.FromV4Alpha1User(u), "local://"+u.Name)
	if err != nil {
		slog.Error("error generating token for user", "user", creds.Username, "error", err.Error())
		statuswriter.WriteError(errors.NewUnauthorized(providers.Unauthorized), w)
		return
	}

	statuswriter.WriteSuccess(tok, w)
}

func (p *Provider) findUser(ctx context.Context, username string) (*v4alpha1.User, error) {
	var userList = &v4alpha1.UserList{}
	if err := p.userCache.List(ctx, userList, client.MatchingFields{
		hflabels.LocalUsernameKey: usernameToPrincipal(username),
	}); err != nil {
		return nil, err
	}

	if len(userList.Items) == 0 {
		slog.Info("no user found for username", "username", username)
		return nil, fmt.Errorf("no user found for username %s", username)
	}

	if len(userList.Items) > 1 {
		// should only match one user for the username
		// auth has to fail because we don't know which one is correct
		slog.Error("multiple users found for username", "username", username)
		return nil, fmt.Errorf("multiple users found for username %s", username)
	}

	// here we should have a single user
	return &userList.Items[0], nil
}

func usernameToPrincipal(username string) string {
	return "local://" + username
}

func principalToUsername(principal string) string {
	return strings.Trim(principal, "local://")
}
