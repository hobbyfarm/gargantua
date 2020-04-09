package util

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base32"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfListers "github.com/hobbyfarm/gargantua/pkg/client/listers/hobbyfarm.io/v1"
	"golang.org/x/crypto/ssh"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	mrand "math/rand"

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

func VerifySession(sLister hfListers.SessionLister, s *hfv1.Session) error {
	var err error
	glog.V(5).Infof("Verifying cs %s", s.Name)
	for i := 0; i < 150000; i++ {
		var fromCache *hfv1.Session
		fromCache, err = sLister.Get(s.Name)
		if err != nil {
			glog.Error(err)
			if apierrors.IsNotFound(err) {
				continue
			}
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

func EnsureVMNotReady(hfClientset *hfClientset.Clientset, vmLister hfListers.VirtualMachineLister, vmName string) error {
	//glog.V(5).Infof("ensuring VM %s is not ready", vmName)
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := hfClientset.HobbyfarmV1().VirtualMachines().Get(vmName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		if result.Labels["ready"] == "false" {
			return nil
		}
		result.Labels["ready"] = "false"

		result, updateErr := hfClientset.HobbyfarmV1().VirtualMachines().Update(result)
		if updateErr != nil {
			return updateErr
		}
		glog.V(4).Infof("set vm %s to not ready")

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

func AvailableRawCapacity(hfClientset *hfClientset.Clientset, capacity hfv1.CMSStruct, virtualMachines []hfv1.VirtualMachine) *hfv1.CMSStruct {
	vmTemplates, err := hfClientset.HobbyfarmV1().VirtualMachineTemplates().List(metav1.ListOptions{})
	if err != nil {
		glog.Errorf("unable to list virtual machine templates, got error %v", err)
		return nil
	}

	currentUsage := hfv1.CMSStruct{}
	for _, vm := range virtualMachines {
		for _, vmTemplate := range vmTemplates.Items {
			if vmTemplate.Spec.Id == vm.Spec.VirtualMachineTemplateId {
				currentUsage.CPU = currentUsage.CPU + vmTemplate.Spec.Resources.CPU
				currentUsage.Memory = currentUsage.Memory + vmTemplate.Spec.Resources.Memory
				currentUsage.Storage = currentUsage.Storage + vmTemplate.Spec.Resources.Storage
			}
		}
	}

	availableCapacity := hfv1.CMSStruct{}

	availableCapacity.CPU = capacity.CPU - currentUsage.CPU
	availableCapacity.Memory = capacity.Memory - currentUsage.Memory
	availableCapacity.Storage = capacity.Storage - currentUsage.Storage

	return &availableCapacity
}

func MaxVMCountsRaw(hfClientset *hfClientset.Clientset, vmTemplates map[string]int, available hfv1.CMSStruct) int {
	vmTemplatesFromK8s, err := hfClientset.HobbyfarmV1().VirtualMachineTemplates().List(metav1.ListOptions{})
	if err != nil {
		glog.Errorf("unable to list virtual machine templates, got error %v", err)
		return 0
	}

	maxCount := 0

	var neededResources hfv1.CMSStruct

	for _, vmTemplate := range vmTemplatesFromK8s.Items {
		if vmtCount, ok := vmTemplates[vmTemplate.Name]; ok {
			neededResources.CPU = neededResources.CPU + vmTemplate.Spec.Resources.CPU*vmtCount
			neededResources.Memory = neededResources.Memory + vmTemplate.Spec.Resources.Memory*vmtCount
			neededResources.Storage = neededResources.Storage + vmTemplate.Spec.Resources.Storage*vmtCount
		}
	}

	maxCount = available.CPU / neededResources.CPU

	if available.Memory/neededResources.Memory > maxCount {
		maxCount = available.Memory / neededResources.Memory
	}

	if available.Storage/neededResources.Storage > maxCount {
		maxCount = available.Storage / neededResources.Storage
	}

	return maxCount

}

// pending rename...
type Maximus struct {
	CapacityMode      hfv1.CapacityMode `json:"capacity_mode"`
	AvailableCount    map[string]int    `json:"available_count"`
	AvailableCapacity hfv1.CMSStruct    `json:"available_capacity"`
}

func MaxAvailableDuringPeriod(hfClientset *hfClientset.Clientset, environment string, startString string, endString string) (Maximus, error) {

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

	environmentFromK8s, err := hfClientset.HobbyfarmV1().Environments().Get(environment, metav1.GetOptions{})

	if err != nil {
		return Maximus{}, fmt.Errorf("error retrieving environment %v", err)
	}

	vmTemplatesFromK8s, err := hfClientset.HobbyfarmV1().VirtualMachineTemplates().List(metav1.ListOptions{})

	if err != nil {
		return Maximus{}, fmt.Errorf("error retrieving virtual machine templates %v", err)
	}

	vmTemplateResources := map[string]hfv1.CMSStruct{}

	for _, vmTemplateInfo := range vmTemplatesFromK8s.Items {
		vmTemplateResources[vmTemplateInfo.Name] = vmTemplateInfo.Spec.Resources
	}

	scheduledEvents, err := hfClientset.HobbyfarmV1().ScheduledEvents().List(metav1.ListOptions{})

	if err != nil {
		return Maximus{}, fmt.Errorf("error retrieving scheduled events %v", err)
	}

	maxRaws := make([]hfv1.CMSStruct, 1)
	maxCounts := map[string]int{}
	maxCounts = make(map[string]int)
	// maxCount will be the largest number of virtual machines allocated from the environment
	/*for t, c := range environmentFromK8s.Spec.CountCapacity {
		maxCounts[t] = c
	}*/
	for i := start; i.Before(end) || i.Equal(end); i = i.Add(duration) {
		glog.V(8).Infof("Checking time at %s", i.Format(time.UnixDate))
		maxRaw := hfv1.CMSStruct{}
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
					glog.V(4).Infof("Scheduled Event %s was within the time period")
					if environmentFromK8s.Spec.CapacityMode == hfv1.CapacityModeRaw {
						for vmTemplateName, vmTemplateCount := range vmMapping {
							if vmTemplateR, ok := vmTemplateResources[vmTemplateName]; ok {
								maxRaw.CPU = vmTemplateR.CPU * vmTemplateCount
								maxRaw.Memory = vmTemplateR.Memory * vmTemplateCount
								maxRaw.Storage = vmTemplateR.Storage * vmTemplateCount
							} else {
								return Maximus{}, fmt.Errorf("error retrieving vm template %s resources %v", vmTemplateName, err)
							}
						}
					} else if environmentFromK8s.Spec.CapacityMode == hfv1.CapacityModeCount {
						for vmTemplateName, vmTemplateCount := range vmMapping {
							glog.V(4).Infof("SE VM Template %s Count was %d", vmTemplateName, vmTemplateCount)
							currentMaxCount[vmTemplateName] = currentMaxCount[vmTemplateName] + vmTemplateCount
						}
					} else {
						return Maximus{}, fmt.Errorf("environment %s had unexpected capacity mode %s", environment, environmentFromK8s.Spec.CapacityMode)
					}
				}
			}
		}
		maxRaws = append(maxRaws, maxRaw)
		if environmentFromK8s.Spec.CapacityMode == hfv1.CapacityModeCount {
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
	}
	max := Maximus{}
	max.CapacityMode = environmentFromK8s.Spec.CapacityMode
	if environmentFromK8s.Spec.CapacityMode == hfv1.CapacityModeRaw {
		maxCPU := 0
		maxMem := 0
		maxStorage := 0
		for _, raw := range maxRaws {
			if maxCPU < raw.CPU {
				maxCPU = raw.CPU
			}
			if maxMem < raw.Memory {
				maxMem = raw.Memory
			}
			if maxStorage < raw.Storage {
				maxStorage = raw.Storage
			}
		}
		max.AvailableCapacity.CPU = environmentFromK8s.Spec.Capacity.CPU - maxCPU
		max.AvailableCapacity.Memory = environmentFromK8s.Spec.Capacity.Memory - maxMem
		max.AvailableCapacity.Storage = environmentFromK8s.Spec.Capacity.Storage - maxStorage
	} else if environmentFromK8s.Spec.CapacityMode == hfv1.CapacityModeCount {
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
	} else {
		return Maximus{}, fmt.Errorf("environment %s had unexpected capacity mode %s", environment, environmentFromK8s.Spec.CapacityMode)
	}
	return max, nil
}

// Contains returns true if string e is in list s, and false otherwise
func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
