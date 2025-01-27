package ldap

import (
	"encoding/base64"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_LabelToDn(t *testing.T) {
	// ldap://dc01.it.example.org/cn=user01,ou=users,dc=it,dc=example,dc=org
	var label = "bGRhcDovL2RjMDEuaXQuZXhhbXBsZS5vcmcvY249dXNlcjAxLG91PXVzZXJzLGRjPWl0LGRjPWV4YW1wbGUsZGM9b3Jn"

	host, dn, err := labelToDN(label)

	if err != nil {
		t.Error(err)
	}

	if host != "dc01.it.example.org" {
		t.Errorf("invalid host returned, expected %s, got %s", "dc01.it.example.org", host)
	}

	if dn != "cn=user01,ou=users,dc=it,dc=example,dc=org" {
		t.Errorf("invalid dn returned, expected %s, got %s", "cn=user01,ou=users,dc=example,dc=org", dn)
	}
}

func Test_DnToLabel(t *testing.T) {
	var host = "dc01.it.example.org"
	var dn = "cn=user01,ou=users,dc=it,dc=example,dc=org"
	l := dnToLabel(dn, &v4alpha1.LdapConfig{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec: v4alpha1.LdapConfigSpec{
			LdapHost: host,
		},
		Status: v4alpha1.LdapConfigStatus{},
	})

	res := base64.StdEncoding.EncodeToString([]byte("ldap://" + host + "/" + dn))

	if l != res {
		t.Errorf("invalid label returned, expected %s, got %s", res, l)
	}
}
