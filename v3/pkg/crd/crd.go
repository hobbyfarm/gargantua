package crd

import (
	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	terraformv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/terraformcontroller.cattle.io/v1"
)

func GenerateCRDs() []crder.CRD {
	return []crder.CRD{
		HobbyfarmCRD(&v1.VirtualMachine{}, func(c *crder.CRD) {
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
		HobbyfarmCRD(&v1.VirtualMachineTemplate{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.VirtualMachineTemplate{}, nil)
		}),
		HobbyfarmCRD(&v1.Environment{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.Environment{}, nil)
		}),
		HobbyfarmCRD(&v1.Course{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.Course{}, nil)
		}),
		HobbyfarmCRD(&v1.Scenario{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.Scenario{}, nil)
		}),
		HobbyfarmCRD(&v1.Session{}, func(c *crder.CRD) {
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
		HobbyfarmCRD(&v1.ScheduledEvent{}, func(c *crder.CRD) {
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
		HobbyfarmCRD(&v1.PredefinedService{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.PredefinedService{}, func(cv *crder.Version) {
					cv.
						WithColumn("Name", ".spec.name").
						WithColumn("Port", ".spec.port")
				})
		}),
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

func HobbyfarmCRD(obj interface{}, customize func(c *crder.CRD)) crder.CRD {
	return *crder.NewCRD(obj, "hobbyfarm.io", customize)
}

func terraformCRD(obj interface{}, customize func(c *crder.CRD)) crder.CRD {
	return *crder.NewCRD(obj, "terraformcontroller.cattle.io", customize)
}
