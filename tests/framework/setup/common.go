package setup

import (
	"context"
	hobbyfarmv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/rancher/wrangler/pkg/yaml"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	ParentCM         = "parent-cm"
	DefaultNamespace = "gargantua-integration"
)

var commonItems = `
apiVersion: hobbyfarm.io/v1
kind: Environment
metadata:
  annotations:
    hobbyfarm.io/provisioner: external
  name: aws-demo
spec:
  burst_capable: true
  burst_capacity:
    cpu: 0
    memory: 0
    storage: 0
  burst_count_capacity:
    sles-15-sp2: 20
  capacity:
    cpu: 0
    memory: 0
    storage: 0
  capacity_mode: count
  count_capacity:
    sles-15-sp2: 5
  display_name: aws-ap-southeast-2
  dnssuffix: ""
  environment_specifics:
    cred_secret: aws-hf-creds
    region: ap-southeast-2
    subnet: subnet-0846aae7febd45d66
    vpc_security_group_id: sg-01b93204da19291a6
  ip_translation_map: {}
  provider: aws
  template_mapping:
    sles-15-sp2:
      image: ami-0fecb8817640dd01e
      instanceType: t3.xlarge
      ssh_username: ec2-user
      rootDiskSize: "50" 
  ws_endpoint: ws.localhost
`

var defaultEnvironment = []byte(`
apiVersion: hobbyfarm.io/v1
kind: Environment
metadata:
  annotations:
    hobbyfarm.io/provisioner: external
  name: aws-demo
spec:
  burst_capable: true
  burst_capacity:
    cpu: 0
    memory: 0
    storage: 0
  burst_count_capacity:
    sles-15-sp2: 20
  capacity:
    cpu: 0
    memory: 0
    storage: 0
  capacity_mode: count
  count_capacity:
    sles-15-sp2: 5
  display_name: aws-ap-southeast-2
  dnssuffix: ""
  environment_specifics:
    cred_secret: aws-hf-creds
    region: ap-southeast-2
    subnet: subnet-0846aae7febd45d66
    vpc_security_group_id: sg-01b93204da19291a6
  ip_translation_map: {}
  provider: aws
  template_mapping:
    sles-15-sp2:
      image: ami-0fecb8817640dd01e
      instanceType: t3.xlarge
      ssh_username: ec2-user
      rootDiskSize: "50" 
  ws_endpoint: ws.localhost
`)

// SetupCommonObjects will leverage Wrangler apply to seed the gargantua test environment
func SetupCommonObjects(ctx context.Context, config *rest.Config) error {

	k, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// create a CM as a parent. Deleting this will trigger GC of other resources
	_, err = k.CoreV1().ConfigMaps(DefaultNamespace).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ParentCM,
			Namespace: DefaultNamespace,
		},
	}, metav1.CreateOptions{})

	if err != nil {
		return err
	}

	hf, err := hfClientset.NewForConfig(config)
	if err != nil {
		return err
	}
	e := &hobbyfarmv1.Environment{}
	err = yaml.Unmarshal(defaultEnvironment, e)
	if err != nil {
		return err
	}

	_, err = hf.HobbyfarmV1().Environments(DefaultNamespace).Create(ctx, e, metav1.CreateOptions{})

	/*a, err := apply.NewForConfig(config)
	if err != nil {
		return err
	}

	runtimeList, err := yaml.ToObjects(strings.NewReader(commonItems))
	if err != nil {
		return err
	}

	err = a.WithOwner(cm).WithDefaultNamespace(DefaultNamespace).ApplyObjects(runtimeList...)*/

	return err

}

// CleanupCommonObjects will cleanup all the created objects
func CleanupCommonObjects(ctx context.Context, config *rest.Config) error {

	k, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	return k.CoreV1().ConfigMaps(DefaultNamespace).Delete(ctx, ParentCM, metav1.DeleteOptions{})
}
