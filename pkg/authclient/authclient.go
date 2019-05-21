package authclient

// authclient is used by the rest of the components to interface with the auth microservice
// to determine whether the user is validated or not

import (
	"github.com/dgrijalva/jwt-go"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
)

func GenerateJWT(user hfv1.User) (string, error) {

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": user.Spec.Email,
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(user.Spec.Password)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ValidateJWT(jwt string) error {
	return nil
}