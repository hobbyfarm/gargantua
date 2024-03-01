package vmtemplateservice

import (
	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
)

func GenerateVMSetCRD() []crder.CRD {
	return []crder.CRD{
		crd.HobbyfarmCRD(&v1.VirtualMachineTemplate{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.VirtualMachineTemplate{}, nil)
		}),
	}
}
