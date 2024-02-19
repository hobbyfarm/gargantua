package progressservice

import (
	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
)

func GenerateProgressCRD() []crder.CRD {
	return []crder.CRD{
		crd.HobbyfarmCRD(&v1.Progress{}, func(c *crder.CRD) {
			c.
				WithNames("progress", "progresses").
				IsNamespaced(true).
				AddVersion("v1", &v1.Progress{}, func(cv *crder.Version) {
					cv.
						WithColumn("CurrentStep", ".spec.current_step").
						WithColumn("Course", ".spec.course").
						WithColumn("Scenario", ".spec.scenario").
						WithColumn("User", ".spec.user").
						WithColumn("Started", ".spec.started").
						WithColumn("LastUpdate", ".spec.last_update")
				})
		}),
	}
}
