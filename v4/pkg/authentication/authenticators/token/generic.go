package token

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
	"github.com/hobbyfarm/gargantua/v4/pkg/authentication/user"
	"github.com/hobbyfarm/gargantua/v4/pkg/config"
	"github.com/hobbyfarm/gargantua/v4/pkg/names"
	"github.com/spf13/viper"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"log/slog"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

const defaultSigningSecret = "WLJLv1PXjqExQzjRd8OEMT8x3P4MK6LN"

var defaultExpirationTime = time.Now().Add(12 * time.Hour)

type HasDisplayName interface {
	DisplayName() string
}

type GenericGeneratorValidator struct {
	signingSecret string
	kclient       client.Client
}

type Claims struct {
	jwt.StandardClaims
	Groups      []string `json:"groups"`
	Principal   string   `json:"principal"`
	DisplayName string   `json:"displayName"`
}

func NewGenericGeneratorValidator(kclient client.Client) GenericGeneratorValidator {
	signingSecret := getSigningSecret(kclient)
	if signingSecret == "" {
		signingSecret = defaultSigningSecret
	}

	return GenericGeneratorValidator{
		signingSecret: signingSecret,
		kclient:       kclient,
	}
}

func (gv GenericGeneratorValidator) GenerateToken(user *user.User, principal string) (string, error) {
	claims := Claims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: gv.getExpirationTime().Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "hobbyfarm",
			NotBefore: time.Now().Unix(),
			Subject:   user.Name,
		},
		Principal:   principal,
		DisplayName: user.DisplayName,
		Groups:      user.Groups,
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := tok.SignedString([]byte(gv.signingSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (gv GenericGeneratorValidator) ValidateToken(tok string) (*user.User, bool) {
	var uid string

	token, err := jwt.ParseWithClaims(tok, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(gv.signingSecret), nil
	})

	if err != nil {
		return nil, false
	}

	return &user.User{
		Name:   token.Claims.(*Claims).Subject,
		UID:    uid,
		Groups: nil,
	}, true
}

func (gv GenericGeneratorValidator) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {
	tok := req.Header.Get("Authorization")
	if !strings.HasPrefix(tok, "Bearer ") {
		return nil, false, fmt.Errorf("token error")
	}

	tok = strings.TrimPrefix(tok, "Bearer ")
	if u, valid := gv.ValidateToken(tok); valid {
		return &authenticator.Response{
			User: u,
		}, true, nil
	} else {
		return nil, false, fmt.Errorf("invalid token")
	}
}

func (gv GenericGeneratorValidator) getExpirationTime() *time.Time {
	var timeout = defaultExpirationTime

	set := &v4alpha1.Setting{}
	if err := gv.kclient.Get(context.TODO(), client.ObjectKey{
		Name: names.UserTokenExpirationSetting,
	}, set); err != nil {
		slog.Error("error looking up setting "+names.UserTokenExpirationSetting+", using "+
			"default expiration time", "error", err.Error())
		return &timeout
	}

	anySet, err := set.FromJSON(set.Value)
	if err != nil {
		slog.Error("error parsing "+names.UserTokenExpirationSetting+" from json, "+
			"returning default expiration time", "error", err.Error())
		return &timeout
	}

	intSet := anySet.(int)
	timeout = time.Now().Add(time.Duration(intSet) * time.Second)
	return &timeout
}

func FromAuthHeader(req *http.Request) (string, error) {
	tok := req.Header.Get("Authorization")
	if !strings.HasPrefix(tok, "Bearer ") {
		return "", fmt.Errorf("invalid token")
	}

	tok = strings.TrimPrefix(tok, "Bearer ")

	return tok, nil
}

func getSigningSecret(kclient client.Client) string {
	secret := &v4alpha1.Secret{}
	err := kclient.Get(context.Background(), client.ObjectKey{Name: viper.GetString(config.JWTSigningKeySecretName)}, secret)
	if err != nil {
		return ""
	}

	data, ok := secret.Data[viper.GetString(config.JWTSigningKeySecretKey)]
	if !ok {
		return ""
	}

	return string(data)
}
