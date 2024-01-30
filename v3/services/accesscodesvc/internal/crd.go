package accesscodeservice

import (
	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
)

func GenerateAccessCodeCRD() []crder.CRD {
	return []crder.CRD{
		crd.HobbyfarmCRD(&v1.AccessCode{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.AccessCode{}, func(cv *crder.Version) {
					cv.
						WithColumn("AccessCode", ".spec.code").
						WithColumn("Expiration", ".spec.expiration")
				})
		}),
		crd.HobbyfarmCRD(&v1.OneTimeAccessCode{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.OneTimeAccessCode{}, func(cv *crder.Version) {
					cv.
						WithColumn("User", ".spec.user").
						WithColumn("Redeemed", ".spec.redeemed_timestamp")
				})
		}),
	}
}
