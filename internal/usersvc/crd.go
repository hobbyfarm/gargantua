package userservice

import (
	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	v2 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v2"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func GenerateUserCRD(caBundle string, reference apiextv1.ServiceReference) []crder.CRD {
	return []crder.CRD{
		hobbyfarmCRD(&v1.User{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.User{}, func(cv *crder.Version) {
					cv.
						WithColumn("Email", ".spec.email")

					cv.IsServed(true)
					cv.IsStored(false)
				}).
				AddVersion("v2", &v2.User{}, func(cv *crder.Version) {
					cv.WithColumn("Email", ".spec.email")

					cv.IsServed(true)
					cv.IsStored(true)
				}).
				WithConversion(func(cc *crder.Conversion) {
					cc.
						StrategyWebhook().
						WithCABundle(caBundle).
						WithService(serviceReferenceWithPath(reference, "/conversion/users.hobbyfarm.io")).
						WithVersions("v2", "v1")
				})
		}),
	}
}

func serviceReferenceWithPath(reference apiextv1.ServiceReference, path string) apiextv1.ServiceReference {
	ref := reference.DeepCopy()
	ref.Path = &path
	return *ref
}

func hobbyfarmCRD(obj interface{}, customize func(c *crder.CRD)) crder.CRD {
	return *crder.NewCRD(obj, "hobbyfarm.io", customize)
}
