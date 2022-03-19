package setup

import (
	"context"
	hobbyfarmv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/rancher/wrangler/pkg/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

var (
	defaultEnvironment = []byte(`
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

	vmTemplate = []byte(`
apiVersion: hobbyfarm.io/v1
kind: VirtualMachineTemplate
metadata:
  name: sles-15-sp2
spec:
  id: sles-15-sp2
  image: sles-15-sp2
  name: sles-15-sp2
  resources:
    cpu: 2
    memory: 4096
    storage: 30`)

	scenario = []byte(`
apiVersion: hobbyfarm.io/v1
kind: Scenario
metadata:
  name: test-scenario
spec:
  categories: []
  description: R2V0dGluZyBzdGFydGVkIHdpdGggTmV1dmVjdG9y
  id: test-scenario
  keepalive_duration: 10m
  name: ZGVtbwo=
  pause_duration: 1h
  pauseable: false
  steps:
  - content: c3RlcCAxCg==
    title: c3RlcCAxCg==
  tags: []
  virtualmachines:
  - neuvector: sles-15-sp2`)
)

// SetupCommonObjects will leverage Wrangler apply to seed the gargantua test environment
func SetupCommonObjects(ctx context.Context, config *rest.Config, suffix string) error {

	labels := make(map[string]string)
	labels["suffix"] = suffix

	hf, err := hfClientset.NewForConfig(config)
	if err != nil {
		return err
	}

	// setup VMTemplate
	vmt := &hobbyfarmv1.VirtualMachineTemplate{}
	err = yaml.Unmarshal(vmTemplate, vmt)
	if err != nil {
		return err
	}

	vmt.Name = vmt.Name + "-" + suffix
	vmt.Labels = labels
	_, err = hf.HobbyfarmV1().VirtualMachineTemplates(DefaultNamespace).Create(ctx, vmt, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// setup environment
	e := &hobbyfarmv1.Environment{}
	err = yaml.Unmarshal(defaultEnvironment, e)
	if err != nil {
		return err
	}

	e.Name = e.Name + "-" + suffix
	e.Labels = labels
	e.Spec.CountCapacity[vmt.Name] = 5
	templateMap := e.Spec.TemplateMapping["sles-15-sp2"]
	e.Spec.TemplateMapping["sles-15-sp2-"+suffix] = templateMap
	_, err = hf.HobbyfarmV1().Environments(DefaultNamespace).Create(ctx, e, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// setup Scenario
	s := &hobbyfarmv1.Scenario{}
	err = yaml.Unmarshal(scenario, s)
	if err != nil {
		return err
	}

	s.Name = s.Name + "-" + suffix
	s.Labels = labels
	s.Spec.Id = s.Spec.Id + "-" + suffix
	_, err = hf.HobbyfarmV1().Scenarios(DefaultNamespace).Create(ctx, s, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// CleanupCommonObjects will cleanup all the created objects
func CleanupCommonObjects(ctx context.Context, config *rest.Config, suffix string) error {

	listOptions := metav1.ListOptions{LabelSelector: "suffix=" + suffix}

	hf, err := hfClientset.NewForConfig(config)
	if err != nil {
		return err
	}

	// cleanup all environments
	err = hf.HobbyfarmV1().Environments(DefaultNamespace).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return err
	}

	// cleanup all VMTemplates
	err = hf.HobbyfarmV1().VirtualMachineTemplates(DefaultNamespace).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return err
	}

	// cleanup all Scenarios
	err = hf.HobbyfarmV1().Scenarios(DefaultNamespace).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	if err != nil {
		return err
	}

	return nil
}
