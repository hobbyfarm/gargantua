package ldap

import (
	"fmt"
	"strings"
)

func dnToLabel(dn string, host string) string {
	pre := "ldap://" + host + "/" + dn

	return pre
}

func labelToDN(label string) (string, string, error) {
	pre := strings.TrimPrefix(label, "ldap://")
	post := strings.Split(pre, "/")

	if len(post) != 2 {
		return "", "", fmt.Errorf("invalid label format: %s", label)
	}

	return post[0], post[1], nil
}
