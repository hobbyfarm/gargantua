package environment

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/mink/pkg/stores"
	"github.com/hobbyfarm/mink/pkg/strategy"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
	"strings"
	"time"
)

type v1Validator struct {
}

func NewV1Storage(environmentStrategy strategy.CompleteStrategy) (rest.Storage, error) {
	v1v := v1Validator{}

	return stores.NewBuilder(environmentStrategy.Scheme(), &v1.Environment{}).
		WithValidateCreate(v1v).
		Build(), nil
}

func (v1v v1Validator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	env := obj.(*v1.Environment)

	hasher := sha256.New()
	hasher.Write([]byte(time.Now().String())) // generate random name
	sha := base32.StdEncoding.WithPadding(-1).EncodeToString(hasher.Sum(nil))[:10]
	env.Name = "env-" + strings.ToLower(sha)

	return
}
