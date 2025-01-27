package ldap

import (
	"context"
	"github.com/go-ldap/ldap/v3"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"log/slog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (p *Provider) ldapGroupsForUser(user *ldap.Entry, lc *v4alpha1.LdapConfig) []string {
	// if configured, try group lookup field
	if lc.Spec.GroupLookupField != "" {
		return user.GetAttributeValues(lc.Spec.GroupLookupField)
	}

	// if no group lookup field specified, try memberOf
	groups := user.GetAttributeValues("memberOf")
	if len(groups) > 0 {
		return groups
	}

	return []string{}
}

func (p *Provider) hfGroupsFromLdapGroups(ctx context.Context, ldapGroups []string) []string {
	// for each ldapgroup, see if a corresponding hf group exists
	var groups = []string{}
	var groupList = &v4alpha1.GroupList{}
	for _, lg := range ldapGroups {
		if err := p.userCache.List(ctx, groupList, client.MatchingFields{
			"group-provider-members-ldap": lg,
		}); err != nil {
			slog.Error("listing groups using cache and index group-provider-members-ldap",
				"error", err.Error())
			return nil
		}

		groups = append(groups, groupNames(groupList.Items)...)
	}

	return groups
}

func groupNames(groups []v4alpha1.Group) []string {
	var out = make([]string, len(groups))
	for i, g := range groups {
		out[i] = g.Name
	}

	return out
}
