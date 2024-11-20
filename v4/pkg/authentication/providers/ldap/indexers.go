package ldap

import (
	labels2 "github.com/hobbyfarm/gargantua/v4/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ldapDnIndexer is an indexing function for user objects that have an LDAP
// principal annotation. On success it returns the DN of the user, nil slice otherwise.
func ldapDnIndexer(obj client.Object) []string {
	anno, ok := obj.GetAnnotations()[labels2.LdapPrincipalKey]
	if !ok {
		return nil
	}

	return []string{anno}
}
