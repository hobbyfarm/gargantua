package crd

import (
	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
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
				AddVersion("v1", v1.Environment{}, func(cv *crder.Version) {
					cv.IsServed(true).IsStored(false)
				}).
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
				AddVersion("v1", &v1.ScheduledEvent{}, func(cv *crder.Version) {
					cv.
						WithColumn("AccessCode", ".spec.access_code").
						WithColumn("Active", ".status.active").
						WithColumn("Finished", ".status.finished").
						WithStatus().
						IsServed(true).
						IsStored(false)
				}).
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
				AddVersion("v1", &v1.AccessCode{}, func(cv *crder.Version) {
					cv.
						WithColumn("AccessCode", ".spec.code").
						WithColumn("Expiration", ".spec.expiration").
						IsStored(false).
						IsServed(true)
				}).
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
				AddVersion("v1", &v1.Session{}, func(cv *crder.Version) {
					cv.
						WithColumn("Paused", ".status.paused").
						WithColumn("Active", ".status.active").
						WithColumn("Finished", ".status.finished").
						WithColumn("StartTime", ".status.start_time").
						WithColumn("ExpirationTime", ".status.end_time").
						WithStatus().
						IsServed(true).
						IsStored(false)
				}).
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
	}
}

func hobbyfarmCRD(obj interface{}, customize func(c *crder.CRD)) crder.CRD {
	return *crder.NewCRD(obj, "hobbyfarm.io", customize)
}
