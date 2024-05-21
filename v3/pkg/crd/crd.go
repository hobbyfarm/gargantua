package crd

import (
	"os"

	"github.com/ebauman/crder"
	"github.com/golang/glog"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"k8s.io/client-go/rest"
)

// A service managing a resource always needs to implement one of the following CRDInstaller interfaces
type CrdInstaller interface {
	GenerateCRDs() []crder.CRD
}
type CrdInstallerWithServiceReference interface {
	GenerateCRDs(ca string, serviceReference ServiceReference) []crder.CRD
}

func GenerateCRDs() []crder.CRD {
	return []crder.CRD{
		HobbyfarmCRD(&v1.PredefinedService{}, func(c *crder.CRD) {
			c.
				IsNamespaced(true).
				AddVersion("v1", &v1.PredefinedService{}, func(cv *crder.Version) {
					cv.
						WithColumn("Name", ".spec.name").
						WithColumn("Port", ".spec.port")
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

func InstallCrds[T CrdInstaller](crdInstaller T, cfg *rest.Config, resourceName string) {
	crds := crdInstaller.GenerateCRDs()
	installCRDsFunc(cfg, resourceName, crds)
}
func InstallCrdsWithServiceReference[T CrdInstallerWithServiceReference](crdInstaller T, cfg *rest.Config, resourceName string, webhookTlsCa string) {
	ca, err := os.ReadFile(webhookTlsCa)
	if err != nil {
		glog.Fatalf("error reading ca certificate: %s", err.Error())
	}

	crds := crdInstaller.GenerateCRDs(string(ca), ServiceReference{
		Namespace: util.GetReleaseNamespace(),
		Name:      "hobbyfarm-webhook",
	})

	installCRDsFunc(cfg, resourceName, crds)
}

func installCRDsFunc(cfg *rest.Config, resourceName string, crds []crder.CRD) {
	glog.Infof("installing/updating %s CRDs", resourceName)
	err := crder.InstallUpdateCRDs(cfg, crds...)
	if err != nil {
		glog.Fatalf("failed installing/updating %s CRDs: %s", resourceName, err.Error())
	}
	glog.Infof("finished installing/updating %s CRDs", resourceName)
}
