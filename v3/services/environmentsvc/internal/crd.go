package environmentservice

import (
	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
)

// EnvironmentCRDInstaller is a struct that can generate CRDs for environments.
// It implements the CrdInstaller interface defined in "github.com/hobbyfarm/gargantua/v3/pkg/microservices"
type EnvironmentCRDInstaller struct{}

func (ei EnvironmentCRDInstaller) GenerateCRDs() []crder.CRD {
	return []crder.CRD{
		crd.HobbyfarmCRD(&v1.Environment{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.Environment{}, nil)
		}),
	}
}
