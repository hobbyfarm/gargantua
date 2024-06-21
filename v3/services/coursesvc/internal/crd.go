package courseservice

import (
	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
)

// CourseCRDInstaller is a struct that can generate CRDs for scenarios.
// It implements the CrdInstaller interface defined in "github.com/hobbyfarm/gargantua/v3/pkg/microservices"
type CourseCRDInstaller struct{}

func (si CourseCRDInstaller) GenerateCRDs() []crder.CRD {
	return []crder.CRD{
		crd.HobbyfarmCRD(&v1.Course{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.Course{}, nil)
		}),
	}
}
