package scenarioservice

import (
	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
)

// ScenarioCRDInstaller is a struct that can generate CRDs for scenarios.
// It implements the CrdInstaller interface defined in "github.com/hobbyfarm/gargantua/v3/pkg/microservices"
type ScenarioCRDInstaller struct{}

func (si ScenarioCRDInstaller) GenerateCRDs() []crder.CRD {
	return []crder.CRD{
		crd.HobbyfarmCRD(&v1.Scenario{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.Scenario{}, nil)
		}),
	}
}
