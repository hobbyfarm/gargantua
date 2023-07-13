package crd

import (
	"fmt"
	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	v2 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v2"
	terraformv1 "github.com/hobbyfarm/gargantua/pkg/apis/terraformcontroller.cattle.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/labels"
	"github.com/hobbyfarm/gargantua/pkg/util"
	v12 "k8s.io/api/admissionregistration/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	namespaceNameLabel = "kubernetes.io/metadata.name"
)

func GenerateCRDs(caBundle string, reference ServiceReference) []crder.CRD {
	return []crder.CRD{
		hobbyfarmCRD(&v1.VirtualMachine{}, func(c *crder.CRD) {
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
		hobbyfarmCRD(&v1.VirtualMachineClaim{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.VirtualMachineClaim{}, func(cv *crder.Version) {
					cv.
						WithColumn("BindMode", ".status.bind_mode").
						WithColumn("Bound", ".status.bound").
						WithColumn("Ready", ".status.ready").
						WithStatus()
				})
		}),
		hobbyfarmCRD(&v1.VirtualMachineTemplate{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.VirtualMachineTemplate{}, nil)
		}),
		hobbyfarmCRD(&v1.Environment{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.Environment{}, nil)
		}),
		hobbyfarmCRD(&v1.VirtualMachineSet{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.VirtualMachineSet{}, func(cv *crder.Version) {
					cv.
						WithColumn("Available", ".status.available").
						WithColumn("Provisioned", ".status.provisioned").
						WithStatus()
				})
		}),
		hobbyfarmCRD(&v1.Course{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.Course{}, nil)
		}),
		hobbyfarmCRD(&v1.Scenario{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.Scenario{}, nil)
		}),
		hobbyfarmCRD(&v1.Session{}, func(c *crder.CRD) {
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
		hobbyfarmCRD(&v1.Progress{}, func(c *crder.CRD) {
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
		hobbyfarmCRD(&v1.AccessCode{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.AccessCode{}, func(cv *crder.Version) {
					cv.
						WithColumn("AccessCode", ".spec.code").
						WithColumn("Expiration", ".spec.expiration")
				})
		}),
		hobbyfarmCRD(&v1.User{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.User{}, func(cv *crder.Version) {
					cv.
						WithColumn("Email", ".spec.email")

					cv.IsServed(true)
					cv.IsStored(false)
				}).
				AddVersion("v2", &v2.User{}, func(cv *crder.Version) {
					cv.WithColumn("Email", ".spec.email")

					cv.IsServed(true)
					cv.IsStored(true)
				}).
				WithConversion(func(cc *crder.Conversion) {
					cc.
						StrategyWebhook().
						WithCABundle(caBundle).
						WithService(reference.Toapiextv1WithPath("/conversion/users.hobbyfarm.io")).
						WithVersions("v2", "v1")
				})
		}),
		hobbyfarmCRD(&v1.ScheduledEvent{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.ScheduledEvent{}, func(cv *crder.Version) {
					cv.
						WithColumn("AccessCode", ".spec.access_code").
						WithColumn("Active", ".status.active").
						WithColumn("Finished", ".status.finished").
						WithStatus()

					cv.IsServed(true)
					cv.IsStored(false)	
				}).
				AddVersion("v2", &v2.ScheduledEvent{}, func(cv *crder.Version) {
					cv.
						WithColumn("AccessCode", ".spec.access_code").
						WithColumn("Active", ".status.active").
						WithColumn("Finished", ".status.finished").
						WithStatus()

					cv.IsServed(true)
					cv.IsStored(true)	
				}).
				WithConversion(func(cc *crder.Conversion) {
					cc.
						StrategyWebhook().
						WithCABundle(caBundle).
						WithService(reference.Toapiextv1WithPath("/conversion/scheduledevents.hobbyfarm.io")).
						WithVersions("v2", "v1")
				})
		}),
		hobbyfarmCRD(&v1.PredefinedService{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.PredefinedService{}, func(cv *crder.Version) {
					cv.
						WithColumn("Name", ".spec.name").
						WithColumn("Port", ".spec.port")
				})
		}),
		hobbyfarmCRD(&v1.DynamicBindConfiguration{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.DynamicBindConfiguration{}, nil)
		}),
		hobbyfarmCRD(&v1.Scope{}, func(c *crder.CRD) {
			c.IsNamespaced(true).AddVersion("v1", &v1.Scope{}, func(cv *crder.Version) {
				cv.WithColumn("DisplayName", ".displayName").IsServed(true).IsStored(true)
			})
		}),
		hobbyfarmCRD(&v1.Setting{}, func(c *crder.CRD) {
			c.IsNamespaced(true).AddVersion("v1", &v1.Setting{}, func(cv *crder.Version) {
				cv.
					WithColumn("DisplayName", ".displayName").
					WithColumn("Scope", fmt.Sprintf(".metadata.labels.%s", labels.DotEscapeLabel(labels.SettingScope))).
					WithColumn("Value", ".value").
					IsServed(true).
					IsStored(true)
			})
			c.AddValidation("settings.hobbyfarm.io", func(vv *crder.Validation) {
				vv.AddRules(v12.RuleWithOperations{
					Operations: []v12.OperationType{
						v12.Create,
						v12.Update,
					},
					Rule: v12.Rule{
						APIGroups:   []string{v1.SchemeGroupVersion.Group},
						APIVersions: []string{v1.SchemeGroupVersion.Version},
						Resources:   []string{"settings"},
					},
				})
				vv.WithCABundle(caBundle)
				vv.WithService(reference.ToadmissionRegistrationv1WithPath("/validation/hobbyfarm.io/v1/settings"))
				vv.WithVersions("v1")
				vv.SetNamespaceSelector(v13.LabelSelector{
					MatchLabels: map[string]string{
						namespaceNameLabel: util.GetReleaseNamespace(), // only process settings in our namespace
					},
				})
				vv.MatchPolicyExact()
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

func hobbyfarmCRD(obj interface{}, customize func(c *crder.CRD)) crder.CRD {
	return *crder.NewCRD(obj, "hobbyfarm.io", customize)
}

func terraformCRD(obj interface{}, customize func(c *crder.CRD)) crder.CRD {
	return *crder.NewCRD(obj, "terraformcontroller.cattle.io", customize)
}
