package settingservice

import (
	"fmt"

	"github.com/ebauman/crder"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/crd"
	"github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	v12 "k8s.io/api/admissionregistration/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	namespaceNameLabel = "kubernetes.io/metadata.name"
)

// SettingCRDInstaller is a struct that can generate CRDs for settings.
// It implements the CrdInstallerWithServiceReference interface defined in "github.com/hobbyfarm/gargantua/v3/pkg/microservices"
type SettingCRDInstaller struct{}

func (si SettingCRDInstaller) GenerateCRDs(caBundle string, reference crd.ServiceReference) []crder.CRD {
	return []crder.CRD{
		crd.HobbyfarmCRD(&v1.Scope{}, func(c *crder.CRD) {
			c.IsNamespaced(true).AddVersion("v1", &v1.Scope{}, func(cv *crder.Version) {
				cv.WithColumn("DisplayName", ".displayName").IsServed(true).IsStored(true)
			})
		}),
		crd.HobbyfarmCRD(&v1.Setting{}, func(c *crder.CRD) {
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
	}
}
