package vmtemplateservice

import (
	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
)

// VMTemplateCRDInstaller is a struct that can generate CRDs for virtual machine templates.
// It implements the CrdInstaller interface defined in "github.com/hobbyfarm/gargantua/v3/pkg/microservices"
type VMTemplateCRDInstaller struct{}

func (vmti VMTemplateCRDInstaller) GenerateCRDs() []crder.CRD {
	return []crder.CRD{
		crd.HobbyfarmCRD(&v1.VirtualMachineTemplate{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.VirtualMachineTemplate{}, nil)
		}),
	}
}
