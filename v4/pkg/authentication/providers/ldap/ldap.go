package ldap

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-ldap/ldap/v3"
	"github.com/gorilla/mux"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/authenticators/token"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/providers"
	user2 "github.com/hobbyfarm/gargantua/v4/pkg/authentication/user"
	labels2 "github.com/hobbyfarm/gargantua/v4/pkg/labels"
	"github.com/hobbyfarm/gargantua/v4/pkg/statuswriter"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

type ldapCreds struct {
	Server string `json:"server'"`
	providers.Credentials
}

type Provider struct {
	kclient   client.Client
	userCache cache.Cache
	token.TokenGeneratorValidator
	*mux.Router
}

func New(kclient client.Client, userCache cache.Cache, tok token.TokenGeneratorValidator, router *mux.Router) *Provider {
	s := &Provider{
		kclient:                 kclient,
		TokenGeneratorValidator: tok,
		Router:                  router,
		userCache:               userCache,
	}

	s.HandleFunc("/login", s.HandleLogin)

	return s
}

func Indexers() map[string]client.IndexerFunc {
	return map[string]client.IndexerFunc{
		labels2.LdapPrincipalKey: ldapDnIndexer,
	}
}

func (p *Provider) HandleLogin(w http.ResponseWriter, r *http.Request) {
	creds, err := parseLdapCredentials(r)
	if err != nil {
		slog.Debug("error parsing ldap credentials", "error", err.Error())
		statuswriter.WriteError(errors.NewUnauthorized(providers.Unauthorized), w)
		return
	}

	// lookup ldap config and attempt a connection
	lc := &v4alpha1.LdapConfig{}
	if err := p.kclient.Get(r.Context(), client.ObjectKey{
		Name: creds.Server,
	}, lc); err != nil {
		slog.Error("error retrieving ldap config", "ldapConfig",
			creds.Server, "error", err.Error())
		statuswriter.WriteError(errors.NewUnauthorized(providers.Unauthorized), w)
		return
	}

	// is this a valid config?
	if lc.Status.Conditions[v4alpha1.ConditionBindSuccessful].Status != corev1.ConditionTrue {
		// nope
		slog.Error("login attempted using invalid ldap config",
			"ldapConfig", lc.Name)
		statuswriter.WriteError(errors.NewUnauthorized(providers.Unauthorized), w)
		return
	}

	// yep
	// test bind
	conn, err := ldap.DialURL("ldap://" + lc.Spec.LdapHost)
	if err != nil {
		slog.Error("error dialing ldap host", "host", lc.Spec.LdapHost, "error", err.Error())
		statuswriter.WriteError(errors.NewUnauthorized(providers.Unauthorized), w)
		return
	}

	defer func() {
		if err = conn.Close(); err != nil {
			slog.Error("error closing ldap connection", "error", err.Error())
		}
	}()

	if err := p.bindAdminAccount(r.Context(), conn, lc); err != nil {
		slog.Error("error binding ldap admin account", "ldapHost",
			lc.Spec.LdapHost, "error", err.Error())
		statuswriter.WriteError(errors.NewUnauthorized(providers.Unauthorized), w)
		return
	}

	// admin bind successful, lookup user
	ldapUser, err := p.lookupUser(r.Context(), creds.Username, conn, lc)
	if err != nil {
		slog.Info("error looking up user", "ldapHost", lc.Spec.LdapHost, "error", err.Error())
		statuswriter.WriteError(errors.NewUnauthorized(providers.Unauthorized), w)
		return
	}

	// user found, attempt binding
	if err := conn.Bind(ldapUser.DN, creds.Password); err != nil {
		slog.Info("invalid ldap credentials", "ldapHost", lc.Spec.LdapHost, "userDN",
			ldapUser.DN)
		statuswriter.WriteError(errors.NewUnauthorized(providers.Unauthorized), w)
		return
	}

	// binding successful, get or create user
	user, err := p.findOrCreateHfUser(r.Context(), lc, ldapUser)
	if err != nil {
		slog.Error("error looking up or creating user", "ldapHost",
			lc.Spec.LdapHost, "error", err.Error())
		statuswriter.WriteError(errors.NewUnauthorized(providers.Unauthorized), w)
		return
	}

	// we have a user, they are valid, so we can issue a token! yay!
	tok, err := p.GenerateToken(user2.FromV4Alpha1User(user), dnToLabel(ldapUser.DN, lc.Spec.LdapHost))
	if err != nil {
		slog.Error("error generating token", "error",
			err.Error(), "user", user.Name)
		statuswriter.WriteError(errors.NewInternalError(err), w)
		return
	}

	statuswriter.WriteSuccess(tok, w)
}

func parseLdapCredentials(req *http.Request) (*ldapCreds, error) {
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	var out = &ldapCreds{}
	if err := json.Unmarshal(bodyBytes, out); err != nil {
		return nil, err
	}

	if out.Server == "" {
		// might be specified by a backslash in the username
		// try to parse it from there
		su := strings.Split(out.Username, "\\")

		if len(su) != 2 {
			return nil, fmt.Errorf("ldap server not provided")
		}

		out.Server = su[0]
	}

	if out.Username == "" {
		return nil, fmt.Errorf("username not provided")
	}

	if out.Password == "" {
		return nil, fmt.Errorf("password not provided")
	}

	return out, nil
}

func (p *Provider) bindAdminAccount(ctx context.Context, conn *ldap.Conn, lc *v4alpha1.LdapConfig) error {
	// first need to get password for bind acct
	secret := &v4alpha1.Secret{}
	if err := p.kclient.Get(ctx, client.ObjectKey{
		Name: lc.Spec.BindPasswordSecret,
	}, secret); err != nil {
		return err
	}

	// is password in there?
	var pw []byte
	var ok bool
	if pw, ok = secret.Data["password"]; !ok {
		return fmt.Errorf("key 'password' not found in secret data")
	}

	// attempt bind
	if err := conn.Bind(lc.Spec.BindUsername, string(pw)); err != nil {
		return err
	}

	return nil
}

func (p *Provider) lookupUser(ctx context.Context, username string, conn *ldap.Conn, lc *v4alpha1.LdapConfig) (*ldap.Entry, error) {
	res, err := conn.Search(&ldap.SearchRequest{
		BaseDN: lc.Spec.SearchBase,
		Scope:  lc.Spec.SearchScope.ConvertToLdapScope(),
		Filter: p.buildUserFilter(username, lc),
	})

	if err != nil {
		return nil, err
	}

	if len(res.Entries) == 0 {
		return nil, fmt.Errorf("no user found")
	}

	if len(res.Entries) > 1 {
		return nil, fmt.Errorf("more than one user found with username %s", username)
	}

	return res.Entries[0], nil
}

func (p *Provider) buildUserFilter(username string, lc *v4alpha1.LdapConfig) string {
	return fmt.Sprintf("(&(objectClass=%s)(%s=%s)%s)", lc.Spec.UserObjectClass,
		lc.Spec.UsernameField, username, lc.Spec.SearchFilter)
}

func (p *Provider) findOrCreateHfUser(ctx context.Context, lc *v4alpha1.LdapConfig, entry *ldap.Entry) (*v4alpha1.User, error) {
	user, err := p.findHfUser(ctx, dnToLabel(entry.DN, lc.Spec.LdapHost))
	if err != nil {
		return nil, err
	}

	if user != nil {
		// update user with latest details
		if err := p.updateUser(ctx, entry, lc, user); err != nil {
			slog.Error("error updating user", "user", user.Name)
			return nil, err
		}

		return user, nil
	}

	// if we get here, we need to make a user
	user, err = p.createUser(ctx, entry, lc)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (p *Provider) findHfUser(ctx context.Context, dnLabel string) (*v4alpha1.User, error) {
	var users = &v4alpha1.UserList{}
	if err := p.userCache.List(ctx, users, client.MatchingFields{
		labels2.LdapPrincipalKey: dnLabel,
	}); err != nil {
		return nil, err
	}

	if len(users.Items) == 0 {
		return nil, nil
	}

	if len(users.Items) > 1 {
		return nil, fmt.Errorf("more than one user found with ldap principal: %s", dnLabel)
	}

	return &users.Items[0], nil
}

func (p *Provider) updateUser(ctx context.Context, entry *ldap.Entry, lc *v4alpha1.LdapConfig, user *v4alpha1.User) error {
	lbl := dnToLabel(entry.DN, lc.Spec.LdapHost)

	if len(user.Spec.Principals) == 0 {
		user.Spec.Principals = make(map[string]string, 1)
	}

	user.Spec.Principals["ldap"] = lbl

	// make sure obj has right label as well
	user.Annotations[labels2.LdapPrincipalKey] = lbl

	if err := p.kclient.Update(ctx, user); err != nil {
		return err
	}

	// grab groups and update those too
	groups := p.hfGroupsFromLdapGroups(ctx, p.ldapGroupsForUser(entry, lc))

	user.Status.GroupMemberships = groups
	user.Status.LastLoginTimestamp = v1.Time{Time: time.Now()}

	if err := p.kclient.Status().Update(ctx, user); err != nil {
		return err
	}

	return nil
}

func (p *Provider) createUser(ctx context.Context, entry *ldap.Entry, lc *v4alpha1.LdapConfig) (*v4alpha1.User, error) {
	var displayName = entry.GetAttributeValue(lc.Spec.DisplayNameField)
	if displayName == "" {
		return nil, fmt.Errorf("display name attribute yielded empty string")
	}

	lbl := dnToLabel(entry.DN, lc.Spec.LdapHost)
	var user = &v4alpha1.User{
		ObjectMeta: v1.ObjectMeta{
			GenerateName: "u-",
			Annotations: map[string]string{
				labels2.LdapPrincipalKey: lbl,
			},
		},
		Spec: v4alpha1.UserSpec{
			Principals: map[string]string{
				"ldap": lbl,
			},
			DisplayName: displayName,
		},
	}

	if err := p.kclient.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
