package util

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base32"
	"encoding/json"
	"encoding/pem"
	"fmt"
	mrand "math/rand"

	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfListers "github.com/hobbyfarm/gargantua/pkg/client/listers/hobbyfarm.io/v1"
	"golang.org/x/crypto/ssh"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
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

func ReturnHTTPRaw(w http.ResponseWriter, r *http.Request, content string) {
	fmt.Fprintf(w, "%s", content)
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

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
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
		fromCache, err = vmLister.VirtualMachines(GetReleaseNamespace()).Get(vm.Name)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			glog.Error(err)
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
		_, err = vmLister.VirtualMachines(GetReleaseNamespace()).Get(vm.Name)
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
		fromCache, err = vmSetLister.VirtualMachineSets(GetReleaseNamespace()).Get(vms.Name)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			glog.Error(err)
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
		fromCache, err = vmClaimLister.VirtualMachineClaims(GetReleaseNamespace()).Get(vmc.Name)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			glog.Error(err)
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

func VerifySession(sLister hfListers.SessionLister, s *hfv1.Session) error {
	var err error
	glog.V(5).Infof("Verifying cs %s", s.Name)
	for i := 0; i < 150000; i++ {
		var fromCache *hfv1.Session
		fromCache, err = sLister.Sessions(GetReleaseNamespace()).Get(s.Name)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			glog.Error(err)
			return err
		}
		if ResourceVersionAtLeast(fromCache.ResourceVersion, s.ResourceVersion) {
			glog.V(5).Infof("resource version matched for %s", s.Name)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	glog.Errorf("resource version didn't match for in time %s", s.Name)
	return nil

}

func EnsureVMNotReady(hfClientset hfClientset.Interface, vmLister hfListers.VirtualMachineLister, vmName string, ctx context.Context) error {
	//glog.V(5).Infof("ensuring VM %s is not ready", vmName)
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := hfClientset.HobbyfarmV1().VirtualMachines(GetReleaseNamespace()).Get(ctx, vmName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		if result.Labels["ready"] == "false" {
			return nil
		}
		result.Labels["ready"] = "false"

		result, updateErr := hfClientset.HobbyfarmV1().VirtualMachines(GetReleaseNamespace()).Update(ctx, result, metav1.UpdateOptions{})
		if updateErr != nil {
			return updateErr
		}
		glog.V(4).Infof("set vm %s to not ready", vmName)

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


// pending rename...
type Maximus struct {
	AvailableCount    map[string]int    `json:"available_count"`
}

func MaxAvailableDuringPeriod(hfClientset hfClientset.Interface, environment string, startString string, endString string, ctx context.Context) (Maximus, error) {

	duration, _ := time.ParseDuration("30m")

	start, err := time.Parse(time.UnixDate, startString)

	if err != nil {
		return Maximus{}, fmt.Errorf("error parsing start time %v", err)
	}

	start = start.Round(duration)

	end, err := time.Parse(time.UnixDate, endString)

	if err != nil {
		return Maximus{}, fmt.Errorf("error parsing end time %v", err)
	}

	end = end.Round(duration)

	environmentFromK8s, err := hfClientset.HobbyfarmV1().Environments(GetReleaseNamespace()).Get(ctx, environment, metav1.GetOptions{})

	if err != nil {
		return Maximus{}, fmt.Errorf("error retrieving environment %v", err)
	}

	scheduledEvents, err := hfClientset.HobbyfarmV1().ScheduledEvents(GetReleaseNamespace()).List(ctx, metav1.ListOptions{})

	if err != nil {
		return Maximus{}, fmt.Errorf("error retrieving scheduled events %v", err)
	}

	maxCounts := map[string]int{}
	maxCounts = make(map[string]int)
	// maxCount will be the largest number of virtual machines allocated from the environment
	/*for t, c := range environmentFromK8s.Spec.CountCapacity {
		maxCounts[t] = c
	}*/
	for i := start; i.Before(end) || i.Equal(end); i = i.Add(duration) {
		glog.V(8).Infof("Checking time at %s", i.Format(time.UnixDate))
		currentMaxCount := map[string]int{}
		for _, se := range scheduledEvents.Items {
			glog.V(4).Infof("Checking scheduled event %s", se.Spec.Name)
			if vmMapping, ok := se.Spec.RequiredVirtualMachines[environment]; ok {
				seStart, err := time.Parse(time.UnixDate, se.Spec.StartTime)
				if err != nil {
					return Maximus{}, fmt.Errorf("error parsing scheduled event start %v", err)
				}
				seEnd, err := time.Parse(time.UnixDate, se.Spec.EndTime)
				if err != nil {
					return Maximus{}, fmt.Errorf("error parsing scheduled event end %v", err)
				}
				// i is the checking time
				// if the time to be checked is after or equal to the start time of the scheduled event
				// and if i is before or equal to the end of the scheduled event
				if i.Equal(seStart) || i.Equal(seEnd) || (i.Before(seEnd) && i.After(seStart)) {
					glog.V(4).Infof("Scheduled Event %s was within the time period", se.Name)
					for vmTemplateName, vmTemplateCount := range vmMapping {
						glog.V(4).Infof("SE VM Template %s Count was %d", vmTemplateName, vmTemplateCount)
						currentMaxCount[vmTemplateName] = currentMaxCount[vmTemplateName] + vmTemplateCount
					}
				}
			}
		}
		for vmt, currentCount := range currentMaxCount {
			glog.V(4).Infof("currentCount for vmt %s is %d", vmt, currentCount)
			if maxCount, ok := maxCounts[vmt]; ok {
				glog.V(4).Infof("Current max count for vmt %s is %d", vmt, maxCount)
				if maxCount < currentCount {
					maxCounts[vmt] = currentCount
				}
			} else {
				maxCounts[vmt] = currentCount
			}
		}
	}
	max := Maximus{}
	max.AvailableCount = make(map[string]int)
	for k, v := range environmentFromK8s.Spec.CountCapacity {
		max.AvailableCount[k] = v
	}
	for vmt, count := range maxCounts {
		if vmtCap, ok := environmentFromK8s.Spec.CountCapacity[vmt]; ok {
			max.AvailableCount[vmt] = vmtCap - count
		} else {
			glog.Errorf("Error looking for maximum count capacity of virtual machine template %s", vmt)
			max.AvailableCount[vmt] = 0
		}
	}
	return max, nil
}

func GetReleaseNamespace() string {
	provisionNS := "hobbyfarm"
	ns := os.Getenv("HF_NAMESPACE")
	if ns != "" {
		provisionNS = ns
	}
	return provisionNS
}

func GetVMConfig(env *hfv1.Environment, vmt *hfv1.VirtualMachineTemplate) map[string]string {
	envSpecificConfigFromEnv := env.Spec.EnvironmentSpecifics
	envTemplateInfo, exists := env.Spec.TemplateMapping[vmt.Name]
	
	config := make(map[string]string)
	config["image"] = vmt.Spec.Image

	// First copy VMT Details (default)
	for k, v := range vmt.Spec.ConfigMap {
		config[k] = v
	}

	// Override with general environment specifics
	for k, v := range envSpecificConfigFromEnv {
		config[k] = v
	}

	//This environment has specifics for this vmt
	if exists {
			// Override with specific from VM on this environment
		for k, v := range envTemplateInfo {
			config[k] = v
		}
	}

	return config
}