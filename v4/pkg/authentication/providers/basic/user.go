package basic

import "k8s.io/apiserver/pkg/authentication/user"

var _ user.Info = (*BasicUser)(nil)

type BasicUser struct {
	name   string
	uid    string
	groups []string
}

func (b BasicUser) GetName() string {
	return b.name
}

func (b BasicUser) GetUID() string {
	return b.uid
}

func (b BasicUser) GetGroups() []string {
	return b.groups
}

func (b BasicUser) GetExtra() map[string][]string {
	return map[string][]string{}
}
