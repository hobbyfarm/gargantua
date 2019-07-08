package util

import (
	"bytes"
	"crypto/rsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base32"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"k8s.io/client-go/util/retry"
	mrand "math/rand"
	"github.com/golang/glog"
	"golang.org/x/crypto/ssh"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	hfListers "github.com/hobbyfarm/gargantua/pkg/client/listers/hobbyfarm.io/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type HTTPMessage struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func ReturnHTTPMessage(w http.ResponseWriter, r *http.Request, httpStatus int, messageType string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	err := HTTPMessage{
		Status:  strconv.Itoa(httpStatus),
		Message: message,
		Type:    messageType,
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(err)
}

type HTTPContent struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Content []byte `json:"content"`
}

func ReturnHTTPContent(w http.ResponseWriter, r *http.Request, httpStatus int, messageType string, content []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	err := HTTPContent{
		Status:  strconv.Itoa(httpStatus),
		Content: content,
		Type:    messageType,
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(err)
}

func GetHTTPErrorCode(httpStatus int) string {
	switch httpStatus {
	case 401:
		return "Unauthorized"
	case 404:
		return "NotFound"
	case 403:
		return "PermissionDenied"
	case 500:
		return "ServerError"
	}

	return "ServerError"
}
func UniqueStringSlice(stringSlice []string) []string {
	keys := make(map[string]bool)
	var list []string
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func GenerateResourceName(prefix string, input string, hashlength int) string {
	hasher := sha256.New()
	hasher.Write([]byte(input))
	sha := base32.StdEncoding.WithPadding(-1).EncodeToString(hasher.Sum(nil))[:hashlength]
	resourceName := fmt.Sprintf("%s-", prefix) + strings.ToLower(sha)

	return resourceName
}

func init() {
	mrand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[mrand.Intn(len(letterRunes))]
	}
	return string(b)
}

// borrowed from longhorn
func ResourceVersionAtLeast(curr, min string) bool {
	if curr == "" || min == "" {
		return true
	}
	currVersion, err := strconv.ParseInt(curr, 10, 64)
	if err != nil {
		glog.Errorf("datastore: failed to parse current resource version %v: %v", curr, err)
		return false
	}
	minVersion, err := strconv.ParseInt(min, 10, 64)
	if err != nil {
		glog.Errorf("datastore: failed to parse minimal resource version %v: %v", min, err)
		return false
	}
	return currVersion >= minVersion
}

func GenKeyPair() (string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	var private bytes.Buffer
	if err := pem.Encode(&private, privateKeyPEM); err != nil {
		return "", "", err
	}

	// generate public key
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}

	public := ssh.MarshalAuthorizedKey(pub)
	return string(public), private.String(), nil
}

func VerifyVM(vmLister hfListers.VirtualMachineLister, vm *hfv1.VirtualMachine) error {
	var err error
	glog.V(5).Infof("Verifying vm %s", vm.Name)
	for i := 0; i < 150000; i++ {
		var fromCache *hfv1.VirtualMachine
		fromCache, err = vmLister.Get(vm.Name)
		if err != nil {
			glog.Error(err)
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		if ResourceVersionAtLeast(fromCache.ResourceVersion, vm.ResourceVersion) {
			glog.V(5).Infof("resource version matched for %s", vm.Name)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	glog.Errorf("resource version didn't match for in time%s", vm.Name)
	return nil
}

func VerifyVMDeleted(vmLister hfListers.VirtualMachineLister, vm *hfv1.VirtualMachine) error {
	var err error
	glog.V(5).Infof("Verifying vm %s", vm.Name)
	for i := 0; i < 150000; i++ {
		_, err = vmLister.Get(vm.Name)
		if err != nil {
			glog.Error(err)
			if apierrors.IsNotFound(err) {
				return nil
			}
			continue
		}
		time.Sleep(100 * time.Millisecond)
	}
	glog.Errorf("vm doesn't appear to have been deleted in time: %s", vm.Name)
	return nil
}

func VerifyVMSet(vmSetLister hfListers.VirtualMachineSetLister, vms *hfv1.VirtualMachineSet) error {
	var err error
	glog.V(5).Infof("Verifying vms %s", vms.Name)
	for i := 0; i < 150000; i++ {
		var fromCache *hfv1.VirtualMachineSet
		fromCache, err = vmSetLister.Get(vms.Name)
		if err != nil {
			glog.Error(err)
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		if ResourceVersionAtLeast(fromCache.ResourceVersion, vms.ResourceVersion) {
			glog.V(5).Infof("resource version matched for %s", vms.Name)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	glog.Errorf("resource version didn't match for in time %s", vms.Name)
	return nil

}

func VerifyVMClaim(vmClaimLister hfListers.VirtualMachineClaimLister, vmc *hfv1.VirtualMachineClaim) error {
	var err error
	glog.V(5).Infof("Verifying vms %s", vmc.Name)
	for i := 0; i < 150000; i++ {
		var fromCache *hfv1.VirtualMachineClaim
		fromCache, err = vmClaimLister.Get(vmc.Name)
		if err != nil {
			glog.Error(err)
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		if ResourceVersionAtLeast(fromCache.ResourceVersion, vmc.ResourceVersion) {
			glog.V(5).Infof("resource version matched for %s", vmc.Name)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	glog.Errorf("resource version didn't match for in time %s", vmc.Name)
	return nil

}


func VerifyScenarioSession(ssLister hfListers.ScenarioSessionLister, ss *hfv1.ScenarioSession) error {
	var err error
	glog.V(5).Infof("Verifying ss %s", ss.Name)
	for i := 0; i < 150000; i++ {
		var fromCache *hfv1.ScenarioSession
		fromCache, err = ssLister.Get(ss.Name)
		if err != nil {
			glog.Error(err)
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		if ResourceVersionAtLeast(fromCache.ResourceVersion, ss.ResourceVersion) {
			glog.V(5).Infof("resource version matched for %s", ss.Name)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	glog.Errorf("resource version didn't match for in time %s", ss.Name)
	return nil

}

func  EnsureVMNotReady(hfClientset *hfClientset.Clientset, vmLister hfListers.VirtualMachineLister, vmName string) error {
	glog.V(5).Infof("ensuring VM %s is not ready", vmName)
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := hfClientset.HobbyfarmV1().VirtualMachines().Get(vmName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		result.Labels["ready"] = "false"

		result, updateErr := hfClientset.HobbyfarmV1().VirtualMachines().Update(result)
		if updateErr != nil {
			return updateErr
		}
		glog.V(4).Infof("updated result for vm")

		verifyErr := VerifyVM(vmLister, result)

		if verifyErr != nil {
			return verifyErr
		}
		return nil
	})
	if retryErr != nil {
		return retryErr
	}

	return nil
}