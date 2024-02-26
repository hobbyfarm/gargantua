package accesscodeservice

import (
	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
)

func GenerateAccessCodeCRD() []crder.CRD {
	return []crder.CRD{
		hobbyfarmCRD(&v1.AccessCode{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.AccessCode{}, func(cv *crder.Version) {
					cv.
						WithColumn("AccessCode", ".spec.code").
						WithColumn("Expiration", ".spec.expiration")
				})
		}),
		hobbyfarmCRD(&v1.OneTimeAccessCode{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.OneTimeAccessCode{}, func(cv *crder.Version) {
					cv.
						WithColumn("User", ".spec.user").
						WithColumn("Redeemed", ".spec.redeemed_timestamp").
						WithColumn("MaxDuration", ".spec.max_duration")
				})
		}),
	}
}

func hobbyfarmCRD(obj interface{}, customize func(c *crder.CRD)) crder.CRD {
	return *crder.NewCRD(obj, "hobbyfarm.io", customize)
}
