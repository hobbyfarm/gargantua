package tfpcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	tfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/terraformcontroller.cattle.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	v12 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/client/listers/terraformcontroller.cattle.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"math/rand"
	"reflect"
	"strings"
	"time"

	"github.com/golang/glog"
	k8sv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
)

type TerraformProvisionerController struct {
	hfClientSet hfClientset.Interface
	//vmWorkqueue workqueue.RateLimitingInterface

	vmWorkqueue workqueue.Interface

	k8sClientset k8s.Interface

	tfClientset hfClientset.Interface

	vmLister  v12.VirtualMachineLister
	envLister v12.EnvironmentLister
	vmtLister v12.VirtualMachineTemplateLister

	tfsLister v1.StateLister
	tfeLister v1.ExecutionLister

	vmSynced  cache.InformerSynced
	vmtSynced cache.InformerSynced
	tfsSynced cache.InformerSynced
	tfeSynced cache.InformerSynced
	envSynced cache.InformerSynced
	ctx       context.Context
}

func NewTerraformProvisionerController(k8sClientSet k8s.Interface, hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, ctx context.Context) (*TerraformProvisionerController, error) {
	tfpController := TerraformProvisionerController{}
	tfpController.hfClientSet = hfClientSet

	tfpController.tfClientset = hfClientSet
	tfpController.k8sClientset = k8sClientSet

	//tfpController.vmWorkqueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "VM")

	tfpController.vmWorkqueue = workqueue.New()

	tfpController.vmLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Lister()
	tfpController.vmSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer().HasSynced

	tfpController.envLister = hfInformerFactory.Hobbyfarm().V1().Environments().Lister()
	tfpController.envSynced = hfInformerFactory.Hobbyfarm().V1().Environments().Informer().HasSynced

	tfpController.vmtLister = hfInformerFactory.Hobbyfarm().V1().VirtualMachineTemplates().Lister()
	tfpController.vmtSynced = hfInformerFactory.Hobbyfarm().V1().VirtualMachineTemplates().Informer().HasSynced

	tfpController.tfsLister = hfInformerFactory.Terraformcontroller().V1().States().Lister()
	tfpController.tfsSynced = hfInformerFactory.Terraformcontroller().V1().States().Informer().HasSynced

	tfpController.tfeLister = hfInformerFactory.Terraformcontroller().V1().Executions().Lister()
	tfpController.tfeSynced = hfInformerFactory.Terraformcontroller().V1().Executions().Informer().HasSynced

	vmInformer := hfInformerFactory.Hobbyfarm().V1().VirtualMachines().Informer()

	vmInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: tfpController.enqueueVM,
		UpdateFunc: func(old, new interface{}) {
			tfpController.enqueueVM(new)
		},
		DeleteFunc: tfpController.enqueueVM,
	}, time.Minute*30)

	tfpController.ctx = ctx

	return &tfpController, nil
}

func (t *TerraformProvisionerController) enqueueVM(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		//utilruntime.HandleError(err)
		return
	}
	glog.V(8).Infof("Enqueueing vm %s", key)
	//t.vmWorkqueue.AddRateLimited(key)
	t.vmWorkqueue.Add(key)
}

func (t *TerraformProvisionerController) Run(stopCh <-chan struct{}) error {
	defer t.vmWorkqueue.ShutDown()

	glog.V(4).Infof("Starting Terraform Provisioner controller")
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, t.vmSynced, t.envSynced, t.vmtSynced, t.tfsSynced, t.tfeSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	glog.Info("Starting TFP controller worker")
	go wait.Until(t.runTFPWorker, time.Second, stopCh)
	glog.Info("Started TFP controller worker")
	<-stopCh
	return nil
}

func (t *TerraformProvisionerController) runTFPWorker() {
	for t.processNextVM() {

	}
}

func (t *TerraformProvisionerController) processNextVM() bool {
	obj, shutdown := t.vmWorkqueue.Get()

	if shutdown {
		return false
	}
	err := func() error {
		defer t.vmWorkqueue.Done(obj)
		//glog.V(8).Infof("processing vm in tfp controller: %v", obj)
		_, objName, err := cache.SplitMetaNamespaceKey(obj.(string)) // this is actually not necessary because VM's are not namespaced yet...
		if err != nil {
			glog.Errorf("error while splitting meta namespace key %v", err)
			return nil
		}

		vm, err := t.vmLister.VirtualMachines(util.GetReleaseNamespace()).Get(objName)
		if err != nil {
			glog.Errorf("error while retrieving virtual machine %s, likely deleted, forgetting in queue %v", objName, err)
			//t.vmWorkqueue.Forget(obj)
			return nil
		}

		err, requeue := t.handleProvision(vm)

		if err != nil {
			glog.Error(err)
		}

		if requeue {
			t.vmWorkqueue.Add(obj)
		}

		//glog.V(4).Infof("vm processed by tfp controller %v", objName)
		return nil

	}()

	if err != nil {
		return true
	}

	return true
}

// returns an error and a boolean of requeue
func (t *TerraformProvisionerController) handleProvision(vm *hfv1.VirtualMachine) (error, bool) {
	// VM shall not be provisioned by internal terraform controller
	if !vm.Spec.Provision {
		if prov, ok := vm.ObjectMeta.Labels["hobbyfarm.io/provisioner"]; ok && prov != "" {
			glog.V(8).Infof("vm %s ignored by internal provisioner due to 3rd party provisioning label", vm.Name)
			t.vmWorkqueue.Done(vm.Name)
		}
		glog.V(8).Infof("vm %s was not a provisioned vm", vm.Name)
		return nil, false
	}

	//glog.V(5).Infof("vm spec was to provision %s", vm.Name)
	if vm.Status.Tainted && vm.DeletionTimestamp == nil {
		util.EnsureVMNotReady(t.hfClientSet, t.vmLister, vm.Name, t.ctx)
		deleteVMErr := t.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).Delete(t.ctx, vm.Name, metav1.DeleteOptions{})
		if deleteVMErr != nil {
			return fmt.Errorf("there was an error while deleting the virtual machine %s", vm.Name), true
		}
		t.vmWorkqueue.Add(vm.Name)
		return nil, false
	}
	if vm.DeletionTimestamp != nil {
		//glog.V(5).Infof("destroying virtual machine")
		util.EnsureVMNotReady(t.hfClientSet, t.vmLister, vm.Name, t.ctx)
		if vm.Status.TFState == "" {
			// vm already deleted let's delete our finalizer
			t.removeFinalizer(vm)
		}
		stateDel := t.tfClientset.TerraformcontrollerV1().States(util.GetReleaseNamespace()).Delete(t.ctx, vm.Status.TFState, metav1.DeleteOptions{})
		if stateDel != nil {
			t.removeFinalizer(vm)
		} else {
			return nil, true // no error, but need to requeue
		}
		return nil, false
	}
	//Status is ReadyForProvisioning AND No Secret provided (Do not provision VM twice, happens due to vm.status being updated after vm.status)
	if vm.Status.Status == hfv1.VmStatusRFP {
		vmt, err := t.vmtLister.VirtualMachineTemplates(util.GetReleaseNamespace()).Get(vm.Spec.VirtualMachineTemplateId)
		if err != nil {
			glog.Errorf("error getting vmt %v", err)
			return err, true
		}
		env, err := t.envLister.Environments(util.GetReleaseNamespace()).Get(vm.Status.EnvironmentId)
		if err != nil {
			glog.Errorf("error getting env %v", err)
			return err, true
		}

		_, exists := env.Spec.TemplateMapping[vmt.Name]
		if !exists {
			glog.Errorf("error pulling environment template info %v", err)
			return fmt.Errorf("Error during RFP: environment %s does not support vmt %s.", env.Name, vmt.Name), true
		}

		// let's provision the vm
		pubKey, privKey, err := util.GenKeyPair()
		if err != nil {
			glog.Errorf("error generating keypair %v", err)
			return err, true
		}
		config := util.GetVMConfig(env, vmt)

		config["name"] = vm.Name
		config["public_key"] = pubKey

		image, exists := config["image"]
		if !exists || image == "" {
			return fmt.Errorf("image does not exist or is empty in vm config for vmt %s", vmt.Name), true
		}

		moduleName, exists := config["module"]
		if !exists || moduleName == "" {
			return fmt.Errorf("module name does not exist or is empty in vm config for vmt %s", vmt.Name), true
		}

		executorImage, exists := config["executor_image"]
		if !exists || executorImage == "" {
			return fmt.Errorf("executorimage does not exist or is empty in vm config for vmt %s", vmt.Name), true
		}

		password, exists := config["password"]
		if !exists {
			password = ""
		}

		r := fmt.Sprintf("%08x", rand.Uint32())
		cm := &k8sv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: strings.Join([]string{vm.Name + "-cm", r}, "-"),
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "hobbyfarm.io/v1",
						Kind:       "VirtualMachine",
						Name:       vm.Name,
						UID:        vm.UID,
					},
				},
			},
			Data: config,
		}

		cm, err = t.k8sClientset.CoreV1().ConfigMaps(util.GetReleaseNamespace()).Create(t.ctx, cm, metav1.CreateOptions{})

		if err != nil {
			glog.Errorf("error creating configmap %s: %v", cm.Name, err)
		}

		keypair := &k8sv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: strings.Join([]string{vm.Name + "-secret", r}, "-"),
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "hobbyfarm.io/v1",
						Kind:       "VirtualMachine",
						Name:       vm.Name,
						UID:        vm.UID,
					},
				},
			},
			Data: map[string][]byte{
				"private_key": []byte(privKey),
				"public_key":  []byte(pubKey),
				"password":    []byte(password),
			},
		}

		keypair, err = t.k8sClientset.CoreV1().Secrets(util.GetReleaseNamespace()).Create(t.ctx, keypair, metav1.CreateOptions{})

		if err != nil {
			glog.Errorf("error creating secret %s: %v", keypair.Name, err)
		}

		tfs := &tfv1.State{
			ObjectMeta: metav1.ObjectMeta{
				Name: strings.Join([]string{vm.Name + "-tfs", r}, "-"),
			},
			Spec: tfv1.StateSpec{
				Variables: tfv1.Variables{
					ConfigNames: []string{cm.Name},
				},
				Image:           executorImage,
				AutoConfirm:     true,
				DestroyOnDelete: true,
				ModuleName:      moduleName,
			},
		}

		credentialsSecret, exists := config["cred_secret"]
		if !exists {
			glog.Errorf("cred secret does not exist in env template")
		}
		if credentialsSecret != "" {
			tfs.Spec.Variables.SecretNames = []string{credentialsSecret}
		}

		tfs, err = t.tfClientset.TerraformcontrollerV1().States(util.GetReleaseNamespace()).Create(t.ctx, tfs, metav1.CreateOptions{})

		if err != nil {
			glog.Errorf("error creating tfs %v", err)
		}

		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			toUpdate, err := t.vmLister.VirtualMachines(util.GetReleaseNamespace()).Get(vm.Name)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				} else {
					glog.Errorf("unknown error encountered when setting terminating %v", err)
				}
			}

			toUpdate.Status.Status = hfv1.VmStatusProvisioned
			toUpdate.Status.TFState = tfs.Name

			toUpdate, updateErr := t.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).UpdateStatus(t.ctx, toUpdate, metav1.UpdateOptions{})

			if updateErr != nil {
				glog.Errorf("error while updating VirtualMachine status %s", toUpdate.Name)
				return updateErr
			}

			toUpdate.Spec.SecretName = keypair.Name
			toUpdate.Labels["ready"] = "false"
			toUpdate.Finalizers = []string{"tfp.controllers.hobbyfarm.io"}

			toUpdate, updateErr = t.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).Update(t.ctx, toUpdate, metav1.UpdateOptions{})

			if updateErr != nil {
				glog.Errorf("error while updating VirtualMachine %s", toUpdate.Name)
				return updateErr
			}

			if err := util.VerifyVM(t.vmLister, toUpdate); err != nil {
				glog.Errorf("error while verifying machine %s", toUpdate.Name)
			}
			return updateErr
		})

		if retryErr != nil {
			return retryErr, true
		}
		glog.V(6).Infof("provisioned vm %s", vm.Name)
		return nil, false

	} else if vm.Status.Status == hfv1.VmStatusProvisioned {
		// let's check the status of our tf provision
		/*tfState, err := t.tfsLister.States(util.GetReleaseNamespace()).Get(vm.Status.TFState)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return fmt.Errorf("execution not found")
			}
			return nil
		} */
		// TEMPORARY WORKAROUND UNTIL WE FIGURE OUT A BETTER WAY TO DO THIS

		if vm.Status.TFState == "" {
			return fmt.Errorf("tf state was blank in object"), true
		}

		tfExecs, err := t.tfeLister.List(labels.Set{
			"state": string(vm.Status.TFState),
		}.AsSelector())

		if err != nil {
			return err, true
		}

		var newestTimestamp *metav1.Time
		var tfExec *tfv1.Execution
		if len(tfExecs) == 0 {
			return fmt.Errorf("no executions found for terraform state"), true
		}

		newestTimestamp = &tfExecs[0].CreationTimestamp
		tfExec = tfExecs[0]
		for _, e := range tfExecs {
			if newestTimestamp.Before(&e.CreationTimestamp) {
				newestTimestamp = &e.CreationTimestamp
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
		if tfExec.Status.Outputs == "" {
			return nil, true
		}

		var tfOutput map[string]map[string]string

		err = json.Unmarshal([]byte(tfExec.Status.Outputs), &tfOutput)
		if err != nil {
			glog.Error(err)
		}
		env, err := t.envLister.Environments(util.GetReleaseNamespace()).Get(vm.Status.EnvironmentId)
		if err != nil {
			glog.Error(err)
			return fmt.Errorf("error getting environment"), true
		}
		glog.V(8).Infof("private ip is: %s", tfOutput["private_ip"]["value"])

		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			toUpdate, err := t.vmLister.VirtualMachines(util.GetReleaseNamespace()).Get(vm.Name)
			old := toUpdate.DeepCopy()
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				} else {
					glog.Errorf("unknown error encountered when setting terminating %v", err)
				}
			}

			toUpdate.Labels["ready"] = "true"

			if reflect.DeepEqual(old.Status, toUpdate.Status) && reflect.DeepEqual(old.Labels, toUpdate.Labels) {
				return nil
			}

			toUpdate, updateErr := t.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).Update(t.ctx, toUpdate, metav1.UpdateOptions{})

			if updateErr != nil {
				glog.Errorf("error while updating machine: %s", toUpdate.Name)
				return updateErr
			}

			toUpdate.Status.PrivateIP = tfOutput["private_ip"]["value"]
			if _, exists := tfOutput["public_ip"]; exists {
				toUpdate.Status.PublicIP = tfOutput["public_ip"]["value"]
			} else {
				toUpdate.Status.PublicIP = translatePrivToPub(env.Spec.IPTranslationMap, tfOutput["private_ip"]["value"])
			}
			toUpdate.Status.Hostname = tfOutput["hostname"]["value"]
			toUpdate.Status.Status = hfv1.VmStatusRunning

			_, updateErr = t.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).UpdateStatus(t.ctx, toUpdate, metav1.UpdateOptions{})

			if err := util.VerifyVM(t.vmLister, toUpdate); err != nil {
				glog.Errorf("error while verifying machine!!! %s", toUpdate.Name)
			}
			return updateErr
		})

		if retryErr != nil {
			return retryErr, true
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

func (t *TerraformProvisionerController) removeFinalizer(vm *hfv1.VirtualMachine) error {
	glog.V(5).Infof("removing finalizer for vm %s", vm.Name)
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		toUpdate, err := t.vmLister.VirtualMachines(util.GetReleaseNamespace()).Get(vm.Name)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			} else {
				glog.Errorf("unknown error encountered when setting terminating %v", err)
			}
		}
		if reflect.DeepEqual(toUpdate.Finalizers, []string{}) {
			return nil
		}
		toUpdate.Finalizers = []string{}
		glog.V(5).Infof("removing vm finalizer for %s", vm.Name)
		toUpdate, updateErr := t.hfClientSet.HobbyfarmV1().VirtualMachines(util.GetReleaseNamespace()).Update(t.ctx, toUpdate, metav1.UpdateOptions{})
		if err := util.VerifyVMDeleted(t.vmLister, toUpdate); err != nil {
			glog.Errorf("error while verifying machine deleted!!! %s", toUpdate.Name)
		}
		return updateErr
	})

	if retryErr != nil {
		glog.Errorf("error while updating vm object while setting terminating %v", retryErr)
	}
	return retryErr
}
