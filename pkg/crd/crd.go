package crd

import (
	"context"
	hobbyfarmv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	terraformv1 "github.com/hobbyfarm/gargantua/pkg/apis/terraformcontroller.cattle.io/v1"
	"io"
	"os"
	"path/filepath"

	"github.com/rancher/wrangler/pkg/crd"
	"github.com/rancher/wrangler/pkg/yaml"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

func WriteFile(filename string) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return Print(f)
}

func Print(out io.Writer) error {
	obj, err := Objects(false)
	if err != nil {
		return err
	}
	data, err := yaml.Export(obj...)
	if err != nil {
		return err
	}
	/* uncomment when adding directly to a helm chart
	objV1Beta1, err := Objects(true)
	if err != nil {
		return err
	}
	dataV1Beta1, err := yaml.Export(objV1Beta1...)
	if err != nil {
		return err
	}


	data = append([]byte("{{- if .Capabilities.APIVersions.Has \"apiextensions.k8s.io/v1\" -}}\n"), data...)
	data = append(data, []byte("{{- else -}}\n---\n")...)
	data = append(data, dataV1Beta1...)
	data = append(data, []byte("{{- end -}}")...) */
	_, err = out.Write(data)
	return err
}

func Objects(v1beta1 bool) (result []runtime.Object, err error) {
	for _, crdDef := range List() {
		if v1beta1 {
			crd, err := crdDef.ToCustomResourceDefinitionV1Beta1()
			if err != nil {
				return nil, err
			}
			result = append(result, crd)
		} else {
			crd, err := crdDef.ToCustomResourceDefinition()
			if err != nil {
				return nil, err
			}
			result = append(result, crd)
		}
	}
	return
}

func List() []crd.CRD {
	return []crd.CRD{
		hobbyfarmCRD(&hobbyfarmv1.VirtualMachine{}, func(c crd.CRD) crd.CRD {
			return c.
				WithColumn("Status", ".status.status").
				WithColumn("Allocated", ".status.allocated").
				WithColumn("publicIP", ".status.public_ip").
				WithColumn("privateIP", ".status.private_ip")

		}),
		hobbyfarmCRD(&hobbyfarmv1.VirtualMachineClaim{}, func(c crd.CRD) crd.CRD {
			return c.
				WithColumn("BindMode", ".status.bind_mode").
				WithColumn("Bound", ".status.bound").
				WithColumn("Ready", ".status.ready")

		}),
		hobbyfarmCRD(&hobbyfarmv1.VirtualMachineTemplate{}, nil),
		hobbyfarmCRD(&hobbyfarmv1.Environment{}, nil),
		hobbyfarmCRD(&hobbyfarmv1.VirtualMachineSet{}, func(c crd.CRD) crd.CRD {
			return c.
				WithColumn("Available", ".status.available").
				WithColumn("Provisioned", ".status.provisioned")

		}),
		hobbyfarmCRD(&hobbyfarmv1.Course{}, nil),
		hobbyfarmCRD(&hobbyfarmv1.Scenario{}, nil),
		hobbyfarmCRD(&hobbyfarmv1.Session{}, func(c crd.CRD) crd.CRD {
			return c.
				WithColumn("Paused", ".status.paused").
				WithColumn("Active", ".status.active").
				WithColumn("Finished", ".status.finished").
				WithColumn("StartTime", ".status.start_time").
				WithColumn("ExpirationTime", ".status.expiration_time")
		}),
		hobbyfarmCRD(&hobbyfarmv1.AccessCode{}, nil),
		hobbyfarmCRD(&hobbyfarmv1.User{}, nil),
		hobbyfarmCRD(&hobbyfarmv1.ScheduledEvent{}, func(c crd.CRD) crd.CRD {
			return c.
				WithColumn("AccessCode", ".status.access_code_id").
				WithColumn("Active", ".status.active").
				WithColumn("Finished", ".status.finished")
		}),
		hobbyfarmCRD(&hobbyfarmv1.DynamicBindConfiguration{}, nil),
		hobbyfarmCRD(&hobbyfarmv1.DynamicBindRequest{}, func(c crd.CRD) crd.CRD {
			return c.
				WithColumn("CurrentAttempts", ".status.current_attempts").
				WithColumn("Expired", ".status.expired").
				WithColumn("Fulfilled", ".status.fulfilled")
		}),
		terraformControllerCRD(&terraformv1.Module{}, func(c crd.CRD) crd.CRD {
			return c.
				WithColumn("CheckTime", ".status.time")
		}),
		terraformControllerCRD(&terraformv1.State{}, func(c crd.CRD) crd.CRD {
			return c.
				WithColumn("LastRunHash", ".status.lastRunHash").
				WithColumn("ExecutionName", ".status.executionName").
				WithColumn("StatePlanName", ".status.executionPlanName")
		}),
		terraformControllerCRD(&terraformv1.Execution{}, func(c crd.CRD) crd.CRD {
			return c.
				WithColumn("JobName", ".status.jobName").
				WithColumn("PlanConfirmed", ".status.planConfirmed")
		}),
	}
}

func Create(ctx context.Context, cfg *rest.Config) error {
	factory, err := crd.NewFactoryFromClient(cfg)
	if err != nil {
		return err
	}

	return factory.BatchCreateCRDs(ctx, List()...).BatchWait()
}

func hobbyfarmCRD(obj interface{}, customize func(crd.CRD) crd.CRD) crd.CRD {
	return newCRD("hobbyfarm.io", "v1", obj, customize)
}

func terraformControllerCRD(obj interface{}, customize func(crd.CRD) crd.CRD) crd.CRD {
	return newCRD("terraformcontroller.cattle.io", "v1", obj, customize)
}

func newCRD(group string, version string, obj interface{}, customize func(crd.CRD) crd.CRD) crd.CRD {
	crd := crd.CRD{
		GVK: schema.GroupVersionKind{
			Group:   group,
			Version: version,
		},
		Status:       false,
		SchemaObject: obj,
	}
	if customize != nil {
		crd = customize(crd)
	}
	return crd
}
