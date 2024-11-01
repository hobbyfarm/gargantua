package ldap

import (
	"context"
	"fmt"
	ldap "github.com/go-ldap/ldap/v3"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/providers"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/user"
	labels2 "github.com/hobbyfarm/gargantua/v4/pkg/labels"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const unauth = "unauthorized"

var _ providers.CredentialedProvider = (*Provider)(nil)

type Provider struct {
	authConn  *ldap.Conn
	adminConn *ldap.Conn
	config    *v4alpha1.LdapConfig
	kclient   client.Client
}

func NewProvider(kclient client.Client) *Provider {
	return &Provider{
		kclient: kclient,
	}
}

func (p *Provider) ConfigureAndTest(ctx context.Context, config *v4alpha1.LdapConfig) (bool, error) {
	adminConn, err := ldap.DialURL(config.LdapHost)
	if err != nil {
		return false, err
	}

	authConn, err := ldap.DialURL(config.LdapHost)
	if err != nil {
		return false, err
	}

	sec := &v4alpha1.Secret{}
	if err := p.kclient.Get(ctx, client.ObjectKey{Name: config.BindPasswordSecret}, nil); err != nil {
		return false, err
	}

	if err := adminConn.Bind(config.BindUsername, string(sec.Data["password"])); err != nil {
		return false, err
	}

	p.config = config
	p.adminConn = adminConn
	p.authConn = authConn
	return true, nil
}

func (p *Provider) HandleLogin(ctx context.Context, creds *providers.Credentials) (*user.User, *errors.StatusError) {
	if p.adminConn == nil || p.authConn == nil {
		return nil, errors.NewBadRequest("provider not instantiated")
	}

	res, err := p.adminConn.Search(&ldap.SearchRequest{
		BaseDN: p.config.SearchBase,
		Scope:  p.config.SearchScope.ConvertToLdapScope(),
		Filter: p.buildUserFilter(creds.Username),
	})
	if err != nil {
		// TODO - Add observability here
		return nil, errors.NewUnauthorized(unauth)
	}

	if len(res.Entries) != 1 {
		return nil, errors.NewUnauthorized(unauth)
	}

	userdn := res.Entries[0].DN

	theUser := &v4alpha1.User{}
	userList := &v4alpha1.UserList{}
	err = p.kclient.List(ctx, userList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			labels2.LdapPrincipalLabel: userdn,
		}),
	})

	if err != nil {
		// TODO - Add observability here
		// This does not indicate that we should stop, rather that there was an issue listing
		// (e.g. an empty list would not throw an error here)
		return nil, errors.NewUnauthorized("unauthorized")
		// TODO - Should this be giving "unauthorized" back to the requester? Or should we be giving a 500 here?
	}

	if len(userList.Items) == 0 {
		// new user, need to create if we are permitted to?
		user := p.NewUser(userdn, p.GetDisplayName(res.Entries[0]))

		// TODO - What are the criteria under which we are not permitted to create a new user?
		err := p.kclient.Create(ctx, user)
		if err != nil {
			// TODO - Add observability here
			return nil, errors.NewUnauthorized("unauthorized")
			// TODO - Should this be giving "unathorized" back to the requester? Or should we be giving a 500 here?
		}

		// If we get here, there's no error and the user now exists
		theUser = user
	} else if len(userList.Items) == 1 {
		theUser = &userList.Items[0]
	} else {
		// more than one user, uh oh
		// TODO - Add observability here
		return nil, errors.NewUnauthorized("unauthorized")
	}

	return user.FromV4Alpha1User(theUser), nil
}

func (p *Provider) buildUserFilter(username string) string {
	return fmt.Sprintf("&(objectClass=%s)(%s=%s)%s", p.config.UserObjectClass,
		p.config.UsernameField, username, p.config.SearchFilter)
}

func (p *Provider) NewUser(principal string, name string) *v4alpha1.User {
	return &v4alpha1.User{
		ObjectMeta: v1.ObjectMeta{
			GenerateName: "u-",
		},
		Spec: v4alpha1.UserSpec{
			Principals: []string{
				principal,
			},
			DisplayName: name,
		},
	}
}

func (p *Provider) GetDisplayName(entry *ldap.Entry) string {
	fields := []string{p.config.DisplayNameField, p.config.UsernameField}

	for _, f := range fields {
		av := entry.GetAttributeValue(f)
		if av != "" {
			return av
		}
	}

	// TODO - Add tracking here that we were unable to determine the user's name
	return "Unknown"
}
