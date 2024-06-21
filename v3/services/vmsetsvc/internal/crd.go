package vmsetservice

import (
	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
)

// VMSetCRDInstaller is a struct that can generate CRDs for virtual machine sets.
// It implements the CrdInstaller interface defined in "github.com/hobbyfarm/gargantua/v3/pkg/microservices"
type VMSetCRDInstaller struct{}

func (vmsi VMSetCRDInstaller) GenerateCRDs() []crder.CRD {
	return []crder.CRD{
		crd.HobbyfarmCRD(&v1.VirtualMachineSet{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.VirtualMachineSet{}, func(cv *crder.Version) {
					cv.
						WithColumn("Available", ".status.available").
						WithColumn("Provisioned", ".status.provisioned").
						WithStatus()
				})
		}),
	}
}
