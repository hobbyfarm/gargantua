package token

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/user"
	"time"
)

type TokenGeneratorValidator interface {
	GenerateToken(user *user.User, expiration time.Time) (string, error)
	ValidateToken(token string) (*user.User, bool)
}
