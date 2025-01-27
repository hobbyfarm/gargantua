package token

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/user"
)

type TokenGeneratorValidator interface {
	GenerateToken(user *user.User, principal string) (string, error)
	ValidateToken(token string) (*user.User, bool)
}
