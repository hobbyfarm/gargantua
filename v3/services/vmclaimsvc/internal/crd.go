package vmclaimservice

import (
	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
)

// VMClaimCRDInstaller is a struct that can generate CRDs for virtual machine claims.
// It implements the CrdInstaller interface defined in "github.com/hobbyfarm/gargantua/v3/pkg/microservices"
type VMClaimCRDInstaller struct{}

func (vmci VMClaimCRDInstaller) GenerateCRDs() []crder.CRD {
	return []crder.CRD{
		crd.HobbyfarmCRD(&v1.VirtualMachineClaim{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.VirtualMachineClaim{}, func(cv *crder.Version) {
					cv.
						WithColumn("BindMode", ".status.bind_mode").
						WithColumn("Bound", ".status.bound").
						WithColumn("Ready", ".status.ready").
						WithStatus()
				})
		}),
	}
}
