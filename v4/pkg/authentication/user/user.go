package user

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"k8s.io/apiserver/pkg/authentication/user"
)

var _ user.Info = (*User)(nil)

func FromV4Alpha1User(user *v4alpha1.User) *User {
	return &User{
		Name:   user.Name,
		UID:    string(user.UID),
		Groups: []string{}, // TODO - Fix this
		Extra: map[string][]string{
			"DisplayName": {user.Spec.DisplayName},
		},
	}
}

type User struct {
	Name   string
	UID    string
	Groups []string
	Extra  map[string][]string
}

func (u *User) GetName() string {
	return u.Name
}

func (u *User) GetUID() string {
	return u.UID
}

func (u *User) GetGroups() []string {
	return u.Groups
}

func (u *User) GetExtra() map[string][]string {
	return u.Extra
}
