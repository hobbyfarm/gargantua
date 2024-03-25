package vmservice

import (
	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
)

// VmCRDInstaller is a struct that can generate CRDs for virtual machines.
// It implements the CrdInstaller interface defined in "github.com/hobbyfarm/gargantua/v3/pkg/microservices"
type VmCRDInstaller struct{}

func (vmi VmCRDInstaller) GenerateCRDs() []crder.CRD {
	return []crder.CRD{
		crd.HobbyfarmCRD(&v1.VirtualMachine{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.VirtualMachine{}, func(cv *crder.Version) {
					cv.
						WithColumn("Status", ".status.status").
						WithColumn("Allocated", ".status.allocated").
						WithColumn("PublicIP", ".status.public_ip").
						WithColumn("PrivateIP", ".status.private_ip").
						WithStatus()
				})
		}),
	}
}
