package crd

import (
	"github.com/ebauman/crder"
	"github.com/hobbyfarm/gargantua/v4/pkg/apis/hobbyfarm.io/v4alpha1"
)

func GenerateCRDs() []crder.CRD {
	return []crder.CRD{
		hobbyfarmCRD(&v4alpha1.Provider{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", &v4alpha1.Provider{}, nil).
				WithShortNames("prov")
		}),
		hobbyfarmCRD(&v4alpha1.MachineTemplate{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", &v4alpha1.Provider{}, func(cv *crder.Version) {
					cv.
						WithColumn("Type", ".spec.machineType").
						WithColumn("DisplayName", ".spec.displayName").
						WithColumn("Protocols", ".spec.connectProtocol").
						WithColumn("Prefix", ".spec.machineNamePrefix")
				}).
				WithShortNames("mt")
		}),
		hobbyfarmCRD(&v4alpha1.Environment{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", &v4alpha1.Environment{}, func(cv *crder.Version) {
					cv.
						WithColumn("Provider", ".spec.provider").
						WithColumn("DisplayName", ".spec.displayName").
						WithStatus().
						IsStored(true).
						IsServed(true)
				}).
				WithShortNames("env")
		}),
		hobbyfarmCRD(&v4alpha1.MachineSet{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", v4alpha1.MachineSet{}, func(cv *crder.Version) {
					cv.
						WithColumn("Provider", ".spec.provider").
						WithColumn("Environment", ".spec.environment").
						WithColumn("Provisioned", ".status.provisioned").
						WithColumn("Available", ".status.available").
						WithStatus()
				}).
				WithShortNames("ms")
		}),
		hobbyfarmCRD(&v4alpha1.Machine{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", v4alpha1.Machine{}, func(cv *crder.Version) {
					cv.
						WithColumn("Type", ".spec.machineType").
						WithColumn("Provider", ".spec.provider").
						WithColumn("MachineSet", ".spec.machineSet").
						WithColumn("PrimaryAddress", ".status.machineInformation['primary_address']").
						WithStatus()
				}).
				WithShortNames("m")
		}),
		hobbyfarmCRD(&v4alpha1.MachineClaim{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", v4alpha1.MachineClaim{}, func(cv *crder.Version) {
					cv.
						WithColumn("User", ".spec.user").
						WithColumn("BindStrategy", ".spec.bindStrategy").
						WithColumn("Phase", ".status.phase").
						WithStatus()
				}).
				WithShortNames("mc")
		}),
		hobbyfarmCRD(&v4alpha1.ScheduledEvent{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", v4alpha1.ScheduledEvent{}, func(cv *crder.Version) {
					cv.
						WithColumn("StartTime", ".spec.startTime").
						WithColumn("ProvisioningStartTime", ".spec.provisioningStartTime").
						WithColumn("EndTime", ".spec.endTime").
						WithColumn("ExpirationStrategy", ".spec.expirationStrategy").
						WithColumn("DisplayName", ".spec.displayName").
						WithStatus().
						IsServed(true).
						IsStored(true)
				}).
				WithShortNames("se")
		}),
		hobbyfarmCRD(&v4alpha1.AccessCode{}, func(c *crder.CRD) {
			c.
				AddVersion("v4alpha1", v4alpha1.AccessCode{}, func(cv *crder.Version) {
					cv.
						WithColumn("Code", ".spec.code").
						WithColumn("Status", ".status.status").
						WithColumn("NotBefore", ".spec.notBefore").
						WithColumn("NotAfter", ".spec.notAfter").
						WithStatus().
						IsStored(true).
						IsServed(true)
				}).
				WithShortNames("ac")
		}),
		hobbyfarmCRD(&v4alpha1.Session{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", v4alpha1.Session{}, func(cv *crder.Version) {
					cv.
						WithColumn("User", ".spec.user").
						WithColumn("AccessCode", ".spec.accessCode").
						WithColumn("ScheduledEvent", ".spec.scheduledEvent").
						WithColumn("MachineClaim", ".status.machineClaim").
						WithColumn("PersistenceStrategy", ".spec.persistenceStrategy").
						WithStatus().
						IsServed(true).
						IsStored(true)
				}).
				WithShortNames("sesh", "sess")
		}),
		hobbyfarmCRD(&v4alpha1.Course{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", v4alpha1.Course{}, func(cv *crder.Version) {
					cv.
						WithColumn("DisplayName", ".spec.displayName").
						WithColumn("Categories", ".spec.categories").
						WithColumn("Tags", ".spec.tags").
						IsServed(true).
						IsStored(true)
				})
		}),
		hobbyfarmCRD(&v4alpha1.OneTimeAccessCode{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", &v4alpha1.OneTimeAccessCode{}, func(cv *crder.Version) {
					cv.
						WithColumn("NotBefore", ".spec.notBefore").
						WithColumn("NotAfter", ".spec.notAfter").
						WithColumn("User", ".spec.user").
						WithColumn("Redeemed", ".status.redeemed").
						IsServed(true).IsStored(true).WithStatus()
				}).WithShortNames("otac")
		}),
		hobbyfarmCRD(&v4alpha1.PredefinedService{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", &v4alpha1.PredefinedService{}, func(cv *crder.Version) {
					cv.
						WithColumn("DisplayName", ".spec.displayName").
						WithColumn("Port", ".spec.port").
						WithColumn("Path", ".spec.path").
						IsStored(true).
						IsServed(true)
				}).
				WithShortNames("ps", "pds")
		}),
		hobbyfarmCRD(&v4alpha1.Progress{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", &v4alpha1.Progress{}, func(cv *crder.Version) {
					cv.
						WithColumn("User", ".spec.user").
						WithColumn("Scennario", ".spec.scenario").
						WithColumn("CurrentStep", ".status.currentStep").
						WithColumn("Started", ".status.started").
						WithColumn("Finished", ".status.finished").
						IsServed(true).IsStored(true)
				}).
				WithNames("progress", "progresses").
				WithShortNames("prog")
		}),
		hobbyfarmCRD(&v4alpha1.Scenario{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", &v4alpha1.Scenario{}, func(cv *crder.Version) {
					cv.
						WithColumn("DisplayName", ".spec.displayName").
						WithColumn("Categories", ".spec.categories").
						WithColumn("Tags", ".spec.tags").
						IsServed(true).IsStored(true)
				}).
				WithShortNames("sc", "scen")
		}),
		hobbyfarmCRD(&v4alpha1.ScenarioStep{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", &v4alpha1.ScenarioStep{}, func(cv *crder.Version) {
					cv.
						WithColumn("Name", ".metadata.name").
						WithColumn("ReferringScenarios", ".status.referringScenarios").
						IsServed(true).IsStored(true)
				}).
				WithShortNames("ss", "step")
		}),
		hobbyfarmCRD(&v4alpha1.Scope{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", &v4alpha1.Scope{}, func(cv *crder.Version) {
					cv.
						WithColumn("DisplayName", ".spec.displayName").
						IsServed(true).IsStored(true)
				})
		}),
		hobbyfarmCRD(&v4alpha1.Setting{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", &v4alpha1.Setting{}, func(cv *crder.Version) {
					cv.
						WithColumn("DisplayName", ".displayName").
						WithColumn("Value", ".value").
						IsStored(true).IsServed(true)
				}).
				WithShortNames("set")
		}),
		hobbyfarmCRD(&v4alpha1.User{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", &v4alpha1.User{}, func(cv *crder.Version) {
					cv.
						WithColumn("LastLogin", ".status.lastLoginTimestamp").
						IsServed(true).IsStored(true).WithStatus()
				})
		}),
		hobbyfarmCRD(&v4alpha1.ServiceAccount{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", &v4alpha1.ServiceAccount{}, nil)
		}),
		hobbyfarmCRD(&v4alpha1.Role{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", &v4alpha1.Role{}, nil)
		}),
		hobbyfarmCRD(&v4alpha1.RoleBinding{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v4alpha1", &v4alpha1.RoleBinding{}, nil)
		}),
	}
}

func hobbyfarmCRD(obj interface{}, customize func(c *crder.CRD)) crder.CRD {
	return *crder.NewCRD(obj, "hobbyfarm.io", customize)
}
