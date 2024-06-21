package sessionservice

import (
	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
)

// SessionCRDInstaller is a struct that can generate CRDs for sessions.
// It implements the CrdInstaller interface defined in "github.com/hobbyfarm/gargantua/v3/pkg/microservices"
type SessionCRDInstaller struct{}

func (si SessionCRDInstaller) GenerateCRDs() []crder.CRD {
	return []crder.CRD{
		crd.HobbyfarmCRD(&v1.Session{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.Session{}, func(cv *crder.Version) {
					cv.
						WithColumn("Paused", ".status.paused").
						WithColumn("Active", ".status.active").
						WithColumn("Finished", ".status.finished").
						WithColumn("StartTime", ".status.start_time").
						WithColumn("ExpirationTime", ".status.end_time").
						WithStatus()
				})
		}),
	}
}
