package terraformsvc

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	controllers "github.com/hobbyfarm/gargantua/v3/pkg/microservices/controller"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	terraformpb "github.com/hobbyfarm/gargantua/v3/protos/terraform"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/golang/glog"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	environmentpb "github.com/hobbyfarm/gargantua/v3/protos/environment"
	vmpb "github.com/hobbyfarm/gargantua/v3/protos/vm"
	vmclaimpb "github.com/hobbyfarm/gargantua/v3/protos/vmclaim"
	vmsetpb "github.com/hobbyfarm/gargantua/v3/protos/vmset"
	vmtemplatepb "github.com/hobbyfarm/gargantua/v3/protos/vmtemplate"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	vmSetFinalizer = "finalizer.hobbyfarm.io/vmset"
)

type VMController struct {
	controllers.DelayingWorkqueueController
	controllers.Reconciler
	VMClient          vmpb.VMSvcClient
	configMapClient   corev1.ConfigMapInterface
	environmentClient environmentpb.EnvironmentSvcClient
	secretClient      corev1.SecretInterface
	HFVMClient        v1.VirtualMachineInterface
	terraformClient   *GrpcTerraformServer
	vmClaimClient     vmclaimpb.VMClaimSvcClient
	vmSetClient       vmsetpb.VMSetSvcClient
	vmTemplateClient  vmtemplatepb.VMTemplateSvcClient
}

func NewVMController(
	hfClient *versioned.Clientset,
	kubeClient *kubernetes.Clientset,
	VMClient vmpb.VMSvcClient,
	hfInformerFactory hfInformers.SharedInformerFactory,
	environmentClient environmentpb.EnvironmentSvcClient,
	terraformClient *GrpcTerraformServer,
	vmClaimClient vmclaimpb.VMClaimSvcClient,
	vmSetClient vmsetpb.VMSetSvcClient,
	vmTemplateClient vmtemplatepb.VMTemplateSvcClient,
	ctx context.Context,
) (*VMController, error) {
	kubeClient.CoreV1().ConfigMaps("")
	vmInformer := hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer()
	delayingWorkqueueController := *controllers.NewDelayingWorkqueueController(
		ctx,
		vmInformer,
		kubeClient,
		"vm-controller",
		time.Minute*30,
		nil,
	)

	vmController := &VMController{
		DelayingWorkqueueController: delayingWorkqueueController,
		VMClient:                    VMClient,
		configMapClient:             kubeClient.CoreV1().ConfigMaps(util.GetReleaseNamespace()),
		environmentClient:           environmentClient,
		secretClient:                kubeClient.CoreV1().Secrets(util.GetReleaseNamespace()),
		HFVMClient:                  hfClient.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()),
		terraformClient:             terraformClient,
		vmClaimClient:               vmClaimClient,
		vmSetClient:                 vmSetClient,
		vmTemplateClient:            vmTemplateClient,
	}
	vmController.SetReconciler(vmController)
	vmController.SetWorkScheduler(vmController)

	return vmController, nil
}

func (v *VMController) Reconcile(objName string) error {
	glog.V(8).Infof("reconciling vm %s inside vm controller", objName)
	// fetch vm
	vm, err := v.VMClient.GetVM(v.Context, &generalpb.GetRequest{Id: objName})
	if err != nil {
		if hferrors.IsGrpcNotFound(err) {
			glog.Infof("vm %s not found on queue.. ignoring", objName)
			return nil
		} else {
			glog.Errorf("error while retrieving vm %s from queue with err %v", objName, err)
			return err
		}
	}

	// VM shall not be provisioned by internal terraform controller
	if prov, ok := vm.GetLabels()["hobbyfarm.io/provisioner"]; ok && prov != "" {
		glog.V(8).Infof("vm %s ignored by terraform controller due to 3rd party provisioning label", vm.GetId())
		v.GetWorkqueue().Done(vm.GetId())
	}

	// trigger reconcile on vmClaims only when associated VM is running
	// this should avoid triggering unwanted reconciles of VMClaims until the VM's are running
	if vm.GetVmClaimId() != "" && vm.GetStatus().GetStatus() == string(hfv1.VmStatusRunning) {
		v.vmClaimClient.AddToWorkqueue(v.Context, &generalpb.ResourceId{Id: vm.GetVmClaimId()})
	}
	if vm.GetStatus().GetTainted() && vm.GetDeletionTimestamp() == nil {
		err, requeue := v.deleteVM(vm)
		v.handleRequeue(err, requeue, vm.GetId())
	} else if vm.GetDeletionTimestamp() != nil {
		err, requeue := v.handleDeletion(vm)
		v.handleRequeue(err, requeue, vm.GetId())
	} else {
		err, requeue := v.handleProvision(vm)
		v.handleRequeue(err, requeue, vm.GetId())
	}
	return nil
}

func (v *VMController) handleRequeue(err error, requeue bool, vmId string) {
	if err != nil {
		glog.Error(err)
	}
	if requeue {
		v.GetWorkqueue().Add(vmId)
	}
}

// returns an error and a boolean of requeue
func (v *VMController) deleteVM(vm *vmpb.VM) (error, bool) {
	_, deleteVMErr := v.VMClient.DeleteVM(v.Context, &generalpb.ResourceId{Id: vm.GetId()})
	if deleteVMErr != nil {
		return fmt.Errorf("there was an error while deleting the virtual machine %s", vm.GetId()), true
	}
	// We do not need to manually requeue this vm if it is deleted successfully. The controller picks up deletion events by design.
	return nil, false
}

// returns an error and a boolean of requeue
func (v *VMController) handleDeletion(vm *vmpb.VM) (error, bool) {
	if vm.GetVmSetId() != "" && util.ContainsFinalizer(vm.GetFinalizers(), vmSetFinalizer) {
		glog.V(4).Infof("requeuing vmset %s to account for tainted vm %s", vm.GetVmSetId(), vm.GetId())
		updatedVmFinalizers := util.RemoveFinalizer(vm.GetFinalizers(), vmSetFinalizer)
		_, err := v.VMClient.UpdateVM(v.Context, &vmpb.UpdateVMRequest{Id: vm.GetId(), Finalizers: &generalpb.StringArray{
			Values: updatedVmFinalizers,
		}})
		if err != nil {
			glog.Errorf("error removing vm finalizer on vm %s", vm.GetId())
			return err, true
		}
		v.vmSetClient.AddToWorkqueue(v.Context, &generalpb.ResourceId{Id: vm.GetVmSetId()})
		// We do not need to manually requeue this vm if it is updated successfully. The controller picks up update events by design.
		return nil, false
	}

	if vm.GetStatus().GetTfstate() == "" {
		return v.updateAndVerifyVMDeletion(vm)
	}

	_, err := v.terraformClient.DeleteState(v.Context, &generalpb.ResourceId{Id: vm.GetStatus().GetTfstate()})
	if hferrors.IsGrpcNotFound(err) {
		// Our vm has no associated terraform state (anymore). Let's remove its remaining finalizers!
		return v.updateAndVerifyVMDeletion(vm)
	} else if err != nil {
		// Something went wrong during the terraform state deletion process. Let's requeue and try again!
		return err, true
	} else {
		// The terraform state was deleted successfully.
		// We still need to requeue, remove the finalizers and confirm that the vm was deleted successfully
		return nil, true
	}
}

// returns an error and a boolean of requeue
func (v *VMController) updateAndVerifyVMDeletion(vm *vmpb.VM) (error, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resultCh := make(chan error, 1)

	// start verification of deletion in a separate goroutine
	go func() {
		resultCh <- util.VerifyDeletion(ctx, v.HFVMClient, vm.GetId())
	}()
	_, err := v.VMClient.UpdateVM(v.Context, &vmpb.UpdateVMRequest{
		Id:         vm.GetId(),
		Finalizers: &generalpb.StringArray{Values: []string{}},
	})
	if err != nil {
		// Something went wrong while removing the remaining finalizers. Let's requeue and try again.
		return err, true
	}

	// At this point the remaining finalizers were removed successfully.
	// But the verification of the vm deletion might fail, e. g. if the context deadline is exceeded.
	// We have chosen not to requeue in this scenario to ensure that the controller remains responsive for other tasks.
	err = <-resultCh
	if err != nil {
		glog.Warningf("VM deletion verification failed: %v", err)
	} else {
		glog.Infof("VM %s deleted successfully", vm.GetId())
	}
	return nil, false
}

// returns an error and a boolean of requeue
func (v *VMController) handleProvision(vm *vmpb.VM) (error, bool) {
	//Status is ReadyForProvisioning AND No Secret provided (Do not provision VM twice, happens due to vm.status being updated after vm.status)
	if vm.Status.Status == string(hfv1.VmStatusRFP) {
		vmt, err := v.vmTemplateClient.GetVMTemplate(v.Context, &generalpb.GetRequest{Id: vm.GetVmTemplateId(), LoadFromCache: true})
		if err != nil {
			glog.Errorf("error getting vmt %v", err)
			return err, true
		}
		env, err := v.environmentClient.GetEnvironment(v.Context, &generalpb.GetRequest{Id: vm.GetStatus().GetEnvironmentId(), LoadFromCache: true})
		if err != nil {
			glog.Errorf("error getting env %v", err)
			return err, true
		}

		_, exists := env.GetTemplateMapping()[vmt.GetId()]
		if !exists {
			glog.Errorf("error pulling environment template info %v", err)
			// @TODO: Why do we requeue here??? This will fail for each iteration as long as the environment is not updated...
			return fmt.Errorf("Error during RFP: environment %s does not support vmt %s.", env.GetId(), vmt.GetId()), true
		}

		// let's provision the vm
		pubKey, privKey, err := util.GenKeyPair()
		if err != nil {
			glog.Errorf("error generating keypair %v", err)
			return err, true
		}
		config := util.GetVMConfig(env, vmt)

		config["name"] = vm.GetId()
		config["public_key"] = pubKey

		image, exists := config["image"]
		if !exists || image == "" {
			return fmt.Errorf("image does not exist or is empty in vm config for vmt %s", vmt.GetId()), true
		}

		moduleName, exists := config["module"]
		if !exists || moduleName == "" {
			return fmt.Errorf("module name does not exist or is empty in vm config for vmt %s", vmt.GetId()), true
		}

		executorImage, exists := config["executor_image"]
		if !exists || executorImage == "" {
			return fmt.Errorf("executorimage does not exist or is empty in vm config for vmt %s", vmt.GetId()), true
		}

		password, exists := config["password"]
		if !exists {
			password = ""
		}

		shellEndpoint := vm.GetStatus().WsEndpoint

		_, exists = config["cloud-config"]
		if exists {
			config["cloud-config"] = strings.Replace(config["cloud-config"], "$_SHELL_ENDPOINT_$", shellEndpoint, -1)
		}

		vmOwnerReference := []metav1.OwnerReference{
			{
				APIVersion: "hobbyfarm.io/v1",
				Kind:       "VirtualMachine",
				Name:       vm.GetId(),
				UID:        types.UID(vm.GetUid()),
			},
		}

		r := fmt.Sprintf("%08x", rand.Uint32())
		cm := &k8sv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:            strings.Join([]string{vm.GetId() + "-cm", r}, "-"),
				OwnerReferences: vmOwnerReference,
			},
			Data: config,
		}

		cm, err = v.configMapClient.Create(v.Context, cm, metav1.CreateOptions{})

		if err != nil {
			glog.Errorf("error creating configmap %s: %v", cm.Name, err)
		}

		keypair := &k8sv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:            strings.Join([]string{vm.GetId() + "-secret", r}, "-"),
				OwnerReferences: vmOwnerReference,
			},
			Data: map[string][]byte{
				"private_key": []byte(privKey),
				"public_key":  []byte(pubKey),
				"password":    []byte(password),
			},
		}

		keypair, err = v.secretClient.Create(v.Context, keypair, metav1.CreateOptions{})

		if err != nil {
			glog.Errorf("error creating secret %s: %v", keypair.Name, err)
		}

		credentialSecrets := []string{}
		credentialsSecret, exists := config["cred_secret"]
		if !exists {
			glog.Errorf("cred secret does not exist in env template")
		}
		if credentialsSecret != "" {
			credentialSecrets = append(credentialSecrets, credentialsSecret)
		}

		tfsId, err := v.terraformClient.CreateState(v.Context, &terraformpb.CreateStateRequest{
			VmId:  vm.GetId(),
			Image: executorImage,
			Variables: &terraformpb.Variables{
				ConfigNames: []string{cm.Name},
				SecretNames: credentialSecrets,
			},
			ModuleName:      moduleName,
			AutoConfirm:     true,
			DestroyOnDelete: true,
		})

		if err != nil {
			glog.Errorf("error creating tfs %v", err)
		}

		_, err = v.VMClient.UpdateVMStatus(v.Context, &vmpb.UpdateVMStatusRequest{
			Id:      vm.GetId(),
			Status:  string(hfv1.VmStatusProvisioned),
			Tfstate: tfsId.GetId(),
		})
		if err != nil {
			return err, true
		}

		var updatedFinalizers []string
		if vm.GetFinalizers() != nil {
			updatedFinalizers = append(vm.GetFinalizers(), "vm.controllers.hobbyfarm.io")
		} else {
			updatedFinalizers = []string{"vm.controllers.hobbyfarm.io"}
		}
		_, err = v.VMClient.UpdateVM(v.Context, &vmpb.UpdateVMRequest{
			Id:         vm.GetId(),
			SecretName: keypair.Name,
			Finalizers: &generalpb.StringArray{Values: updatedFinalizers},
		})
		if err != nil {
			return err, true
		}

		glog.V(6).Infof("provisioned vm %s", vm.GetId())
		return nil, false

	} else if vm.Status.Status == string(hfv1.VmStatusProvisioned) {
		// let's check the status of our tf provision
		/*tfState, err := t.tfsLister.States(util.GetReleaseNamespace()).Get(vm.Status.TFState)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return fmt.Errorf("execution not found")
			}
			return nil
		} */
		// TEMPORARY WORKAROUND UNTIL WE FIGURE OUT A BETTER WAY TO DO THIS

		if vm.GetStatus().GetTfstate() == "" {
			return fmt.Errorf("tf state was blank in object"), true
		}

		labelSelectorString := labels.Set{"state": string(vm.GetStatus().GetTfstate())}.AsSelector().String()
		tfExecsList, err := v.terraformClient.ListExecution(v.Context, &generalpb.ListOptions{
			LabelSelector: labelSelectorString,
		})

		if err != nil {
			return err, true
		}

		tfExecs := tfExecsList.GetExecutions()

		var newestTimestamp time.Time
		var tfExec *terraformpb.Execution
		if len(tfExecs) == 0 {
			return fmt.Errorf("no executions found for terraform state"), true
		}

		newestTimestamp = tfExecs[0].GetCreationTimestamp().AsTime()
		tfExec = tfExecs[0]
		for _, e := range tfExecs {
			if newestTimestamp.Before(e.GetCreationTimestamp().AsTime()) {
				newestTimestamp = e.GetCreationTimestamp().AsTime()
				tfExec = e
			}
		}
		// END TEMPORARY WORKAROUND

		//executionName := tfState.Status.ExecutionName
		/*
			tfExec, err := t.tfeLister.Executions(util.GetReleaseNamespace()).Get(executionName)
			if err != nil {
				//glog.Error(err)
				if apierrors.IsNotFound(err) {
					return fmt.Errorf("execution not found")
				}
				return nil
			}
		*/
		if tfExec.GetStatus().GetOutputs() == "" {
			return nil, true
		}

		tfOutput, err := util.GenericUnmarshal[map[string]map[string]string](tfExec.GetStatus().GetOutputs(), "terraform execution output")
		if err != nil {
			glog.Error(err)
		}
		env, err := v.environmentClient.GetEnvironment(v.Context, &generalpb.GetRequest{
			Id:            vm.GetStatus().GetEnvironmentId(),
			LoadFromCache: true,
		})
		if err != nil {
			glog.Error(err)
			return fmt.Errorf("error getting environment"), true
		}
		glog.V(8).Infof("private ip is: %s", tfOutput["private_ip"]["value"])

		var publicIP string
		if _, exists := tfOutput["public_ip"]; exists {
			publicIP = tfOutput["public_ip"]["value"]
		} else {
			publicIP = translatePrivToPub(env.GetIpTranslationMap(), tfOutput["private_ip"]["value"])
		}

		_, err = v.VMClient.UpdateVMStatus(v.Context, &vmpb.UpdateVMStatusRequest{
			Id:        vm.GetId(),
			Status:    string(hfv1.VmStatusRunning),
			PublicIp:  wrapperspb.String(publicIP),
			PrivateIp: wrapperspb.String(tfOutput["private_ip"]["value"]),
			Hostname:  wrapperspb.String(tfOutput["hostname"]["value"]),
		})

		if err != nil {
			return err, true
		}
	}
	return nil, false
}

func translatePrivToPub(translationMap map[string]string, priv string) string {
	splitIp := strings.Split(priv, ".")

	origPrefix := splitIp[0] + "." + splitIp[1] + "." + splitIp[2]

	translation, ok := translationMap[origPrefix]

	if ok {
		return translation + "." + splitIp[3]
	}
	return ""

}
