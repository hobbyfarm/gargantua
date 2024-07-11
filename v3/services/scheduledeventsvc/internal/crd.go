package eventservice

import (
	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
)

// ScheduledEventCRDInstaller is a struct that can generate CRDs for scheduled events.
// It implements the CrdInstaller interface defined in "github.com/hobbyfarm/gargantua/v3/pkg/microservices"
type ScheduledEventCRDInstaller struct{}

func (si ScheduledEventCRDInstaller) GenerateCRDs() []crder.CRD {
	return []crder.CRD{
		crd.HobbyfarmCRD(&v1.ScheduledEvent{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.ScheduledEvent{}, func(cv *crder.Version) {
					cv.
						WithColumn("AccessCode", ".spec.access_code").
						WithColumn("Active", ".status.active").
						WithColumn("Finished", ".status.finished").
						WithStatus()
				})
		}),
	}
}
