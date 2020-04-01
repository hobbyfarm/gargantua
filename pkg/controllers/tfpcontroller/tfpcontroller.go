package tfpcontroller

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	tfv1 "github.com/hobbyfarm/gargantua/pkg/apis/terraformcontroller.cattle.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	hfListers "github.com/hobbyfarm/gargantua/pkg/client/listers/hobbyfarm.io/v1"
	tfListers "github.com/hobbyfarm/gargantua/pkg/client/listers/terraformcontroller.cattle.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/util"
	k8sv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/client-go/util/workqueue"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type TerraformProvisionerController struct {
	hfClientSet *hfClientset.Clientset
	//vmWorkqueue workqueue.RateLimitingInterface

	vmWorkqueue workqueue.Interface

	k8sClientset *k8s.Clientset

	tfClientset *hfClientset.Clientset

	vmLister  hfListers.VirtualMachineLister
	envLister hfListers.EnvironmentLister
	vmtLister hfListers.VirtualMachineTemplateLister

	tfsLister tfListers.StateLister
	tfeLister tfListers.ExecutionLister

	vmSynced  cache.InformerSynced
	vmtSynced cache.InformerSynced
	tfsSynced cache.InformerSynced
	tfeSynced cache.InformerSynced
	envSynced cache.InformerSynced
}

var provisionNS = "hobbyfarm"

func init() {
	ns := os.Getenv("HF_NAMESPACE")
	if ns != "" {
		provisionNS = ns
	}
}

func NewTerraformProvisionerController(k8sClientSet *k8s.Clientset, hfClientSet *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*TerraformProvisionerController, error) {
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

		vm, err := t.vmLister.Get(objName)
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
	if vm.Spec.Provision {
		//glog.V(5).Infof("vm spec was to provision %s", vm.Name)
		if vm.Status.Tainted && vm.DeletionTimestamp == nil {
			util.EnsureVMNotReady(t.hfClientSet, t.vmLister, vm.Name)
			deleteVMErr := t.hfClientSet.HobbyfarmV1().VirtualMachines().Delete(vm.Name, &metav1.DeleteOptions{})
			if deleteVMErr != nil {
				return fmt.Errorf("there was an error while deleting the virtual machine %s", vm.Name), true
			}
			t.vmWorkqueue.Add(vm.Name)
			return nil, false
		}
		if vm.DeletionTimestamp != nil {
			//glog.V(5).Infof("destroying virtual machine")
			util.EnsureVMNotReady(t.hfClientSet, t.vmLister, vm.Name)
			if vm.Status.TFState == "" {
				// vm already deleted let's delete our finalizer
				t.removeFinalizer(vm)
			}
			stateDel := t.tfClientset.TerraformcontrollerV1().States(provisionNS).Delete(vm.Status.TFState, &metav1.DeleteOptions{})
			if stateDel != nil {
				t.removeFinalizer(vm)
			} else {
				return nil, true // no error, but need to requeue
			}
			return nil, false
		}
		if vm.Status.Status == hfv1.VmStatusRFP {
			vmt, err := t.vmtLister.Get(vm.Spec.VirtualMachineTemplateId)
			if err != nil {
				glog.Errorf("error getting vmt %v", err)
				return err, true
			}
			env, err := t.envLister.Get(vm.Status.EnvironmentId)
			if err != nil {
				glog.Errorf("error getting env %v", err)
				return err, true
			}
			// let's provision the vm
			pubKey, privKey, err := util.GenKeyPair()
			if err != nil {
				glog.Errorf("error generating keypair %v", err)
				return err, true
			}
			envSpecificConfigFromEnv := env.Spec.EnvironmentSpecifics
			envTemplateInfo, exists := env.Spec.TemplateMapping[vmt.Name]
			if !exists {
				glog.Errorf("error pulling environment template info %v", err)
				return fmt.Errorf("environment template info does not exist for this template %s", vmt.Name), true
			}
			config := make(map[string]string)
			for k, v := range envSpecificConfigFromEnv {
				config[k] = v
			}

			for k, v := range envTemplateInfo {
				config[k] = v
			}

			config["name"] = vm.Name
			config["public_key"] = pubKey
			config["cpu"] = strconv.Itoa(vmt.Spec.Resources.CPU)
			config["memory"] = strconv.Itoa(vmt.Spec.Resources.Memory)
			config["disk"] = strconv.Itoa(vmt.Spec.Resources.Storage)
			image, exists := envTemplateInfo["image"]
			if !exists {
				glog.Errorf("image does not exist in env template")
				return fmt.Errorf("image did not exist"), true
			}
			config["image"] = image

			r := fmt.Sprintf("%08x", rand.Uint32())
			cm := &k8sv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: strings.Join([]string{vm.Name + "-cm", r}, "-"),
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "v1",
							Kind:       "VirtualMachine",
							Name:       vm.Name,
							UID:        vm.UID,
						},
					},
				},
				Data: config,
			}

			cm, err = t.k8sClientset.CoreV1().ConfigMaps(provisionNS).Create(cm)

			if err != nil {
				glog.Errorf("error creating configmap %s: %v", cm.Name, err)
			}

			keypair := &k8sv1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: strings.Join([]string{vm.Name + "-secret", r}, "-"),
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "v1",
							Kind:       "VirtualMachine",
							Name:       vm.Name,
							UID:        vm.UID,
						},
					},
				},
				Data: map[string][]byte{
					"private_key": []byte(privKey),
					"public_key":  []byte(pubKey),
				},
			}

			keypair, err = t.k8sClientset.CoreV1().Secrets(provisionNS).Create(keypair)

			if err != nil {
				glog.Errorf("error creating secret %s: %v", keypair.Name, err)
			}

			moduleName, exists := envTemplateInfo["module"]
			if !exists {
				moduleName, exists = config["module"]
				if !exists {
					glog.Errorf("module name does not exist")
				}
			}

			if moduleName == "" {
				return fmt.Errorf("module name does not exist"), true
			}

			executorImage, exists := envTemplateInfo["executor_image"]
			if !exists {
				executorImage, exists = config["executor_image"]
				if !exists {
					glog.Errorf("executor image does not exist")
				}
			}
			if executorImage == "" {
				return fmt.Errorf("executorimage does not exist"), true
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

			credentialsSecret, exists := envTemplateInfo["cred_secret"]
			if !exists {
				credentialsSecret, exists = config["cred_secret"]
				if !exists {
					glog.Errorf("cred secret does not exist in env template")
				}
			}
			if credentialsSecret != "" {
				tfs.Spec.Variables.SecretNames = []string{credentialsSecret}
			}

			tfs, err = t.tfClientset.TerraformcontrollerV1().States(provisionNS).Create(tfs)

			if err != nil {
				glog.Errorf("error creating tfs %v", err)
			}

			retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				toUpdate, err := t.vmLister.Get(vm.Name)
				if err != nil {
					if apierrors.IsNotFound(err) {
						return nil
					} else {
						glog.Errorf("unknown error encountered when setting terminating %v", err)
					}
				}
				toUpdate.Spec.KeyPair = keypair.Name
				toUpdate.Status.Status = hfv1.VmStatusProvisioned
				toUpdate.Status.TFState = tfs.Name
				toUpdate.Labels["ready"] = "false"
				toUpdate.Finalizers = []string{"tfp.controllers.hobbyfarm.io"}

				toUpdate, updateErr := t.hfClientSet.HobbyfarmV1().VirtualMachines().Update(toUpdate)
				if err := util.VerifyVM(t.vmLister, toUpdate); err != nil {
					glog.Errorf("error while verifying machine!!! %s", toUpdate.Name)
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
			/*tfState, err := t.tfsLister.States(provisionNS).Get(vm.Status.TFState)
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
				tfExec, err := t.tfeLister.Executions(provisionNS).Get(executionName)
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
			env, err := t.envLister.Get(vm.Status.EnvironmentId)
			if err != nil {
				glog.Error(err)
				return fmt.Errorf("error getting environment"), true
			}
			glog.V(8).Infof("private ip is: %s", tfOutput["private_ip"]["value"])

			retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				toUpdate, err := t.vmLister.Get(vm.Name)
				old := toUpdate.DeepCopy()
				if err != nil {
					if apierrors.IsNotFound(err) {
						return nil
					} else {
						glog.Errorf("unknown error encountered when setting terminating %v", err)
					}
				}

				toUpdate.Status.PrivateIP = tfOutput["private_ip"]["value"]
				if _, exists := tfOutput["public_ip"]; exists {
					toUpdate.Status.PublicIP = tfOutput["public_ip"]["value"]
				} else {
					toUpdate.Status.PublicIP = translatePrivToPub(env.Spec.IPTranslationMap, tfOutput["private_ip"]["value"])
				}
				toUpdate.Status.Hostname = tfOutput["hostname"]["value"]
				toUpdate.Status.Status = hfv1.VmStatusRunning
				toUpdate.Labels["ready"] = "true"

				if reflect.DeepEqual(old.Spec, toUpdate.Spec) && reflect.DeepEqual(old.Status, toUpdate.Status) && reflect.DeepEqual(old.Labels, toUpdate.Labels) {
					return nil
				}
				toUpdate, updateErr := t.hfClientSet.HobbyfarmV1().VirtualMachines().Update(toUpdate)
				if err := util.VerifyVM(t.vmLister, toUpdate); err != nil {
					glog.Errorf("error while verifying machine!!! %s", toUpdate.Name)
				}
				return updateErr
			})

			if retryErr != nil {
				return retryErr, true
			}

		}
	} else {
		glog.V(8).Infof("vm %s was not a provisioned vm", vm.Name)
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
		toUpdate, err := t.vmLister.Get(vm.Name)
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
		toUpdate, updateErr := t.hfClientSet.HobbyfarmV1().VirtualMachines().Update(toUpdate)
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
