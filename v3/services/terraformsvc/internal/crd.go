package terraformsvc

import (
	"github.com/ebauman/crder"
	terraformv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/terraformcontroller.cattle.io/v1"
)

// TerraformCRDInstaller is a struct that generates necessary CRDs for terraform.
// It implements the CrdInstaller interface defined in "github.com/hobbyfarm/gargantua/v3/pkg/microservices"
type TerraformCRDInstaller struct{}

func (ti TerraformCRDInstaller) GenerateCRDs() []crder.CRD {
	return []crder.CRD{
		terraformCRD(&terraformv1.Module{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &terraformv1.Module{}, func(cv *crder.Version) {
					cv.
						WithColumn("CheckTime", ".status.time")
				})
		}),
		terraformCRD(&terraformv1.State{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &terraformv1.State{}, func(cv *crder.Version) {
					cv.
						WithColumn("LastRunHash", ".status.lasRunHash").
						WithColumn("ExecutionName", ".status.executionName").
						WithColumn("StatePlanName", ".status.executionPlanName")
				})
		}),
		terraformCRD(&terraformv1.Execution{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &terraformv1.Execution{}, func(cv *crder.Version) {
					cv.
						WithColumn("JobName", ".status.jobName").
						WithColumn("PlanConfirmed", ".status.planConfirmed")
				})
		}),
	}
}

func terraformCRD(obj interface{}, customize func(c *crder.CRD)) crder.CRD {
	return *crder.NewCRD(obj, "terraformcontroller.cattle.io", customize)
}
