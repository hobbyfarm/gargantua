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
	"slices"

	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	environmentpb "github.com/hobbyfarm/gargantua/v3/protos/environment"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	scenariopb "github.com/hobbyfarm/gargantua/v3/protos/scenario"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
	vmpb "github.com/hobbyfarm/gargantua/v3/protos/vm"
	vmtemplatepb "github.com/hobbyfarm/gargantua/v3/protos/vmtemplate"

	"github.com/golang/glog"
	"golang.org/x/crypto/ssh"
	"google.golang.org/protobuf/encoding/protojson"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/util/retry"

	"net/http"
	"os"
	"sort"
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

func VerifyVM(vmLister v1.VirtualMachineLister, vm *hfv1.VirtualMachine) error {
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

func VerifyVMDeleted(vmLister v1.VirtualMachineLister, vm *hfv1.VirtualMachine) error {
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

func VerifyVMSet(vmSetLister v1.VirtualMachineSetLister, vms *hfv1.VirtualMachineSet) error {
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

func VerifyVMClaim(vmClaimLister v1.VirtualMachineClaimLister, vmc *hfv1.VirtualMachineClaim) error {
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

func VerifySession(sLister v1.SessionLister, s *hfv1.Session) error {
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

func EnsureVMNotReady(hfClientset hfClientset.Interface, vmLister v1.VirtualMachineLister, vmName string, ctx context.Context) error {
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

// Range with reserved virtual machine amounts for given time range
type Range struct {
	Start     time.Time
	End       time.Time
	VMMapping map[string]uint32
}

// These functions are used to sort arrays of time.Time
type ByTime []time.Time

func (t ByTime) Len() int           { return len(t) }
func (t ByTime) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t ByTime) Less(i, j int) bool { return t[i].Before(t[j]) }
func sortTime(timeArray []time.Time) {
	sort.Sort(ByTime(timeArray))
}

func MaxUint32(x, y uint32) uint32 {
	if x < y {
		return y
	}
	return x
}

// Calculates available virtualMachineTemplates for a given period (startString, endString) and environment
// Returns a map with timestamps and corresponding availability of virtualmachines. Also returns the maximum available count of virtualmachinetemplates over the whole duration.
func VirtualMachinesUsedDuringPeriod(eventClient scheduledeventpb.ScheduledEventSvcClient, environment string, startString string, endString string, ctx context.Context) (map[time.Time]map[string]uint32, map[string]uint32, error) {
	start, err := time.Parse(time.UnixDate, startString)
	if err != nil {
		return map[time.Time]map[string]uint32{}, map[string]uint32{}, fmt.Errorf("error parsing start time %v", err)
	}

	// We only want to calculate for the future. Otherwise old ( even finished ) events will be considered too.
	if start.Before(time.Now()) {
		start = time.Now()
	}

	end, err := time.Parse(time.UnixDate, endString)
	if err != nil {
		return map[time.Time]map[string]uint32{}, map[string]uint32{}, fmt.Errorf("error parsing end time %v", err)
	}

	scheduledEventList, err := eventClient.ListScheduledEvent(ctx, &generalpb.ListOptions{})
	if err != nil {
		return map[time.Time]map[string]uint32{}, map[string]uint32{}, fmt.Errorf("error retrieving scheduled events: %s", hferrors.GetErrorMessage(err))
	}

	var timeRange []Range
	var changingTimestamps []time.Time                           // All timestamps where number of virtualmachines changes (Begin or End of Scheduled Event)
	virtualMachineCount := make(map[time.Time]map[string]uint32) // Count of virtualmachines per VMTemplate for any given timestamp where a change happened
	maximumVirtualMachineCount := make(map[string]uint32)        // Maximum VirtualMachine Count per VirtualMachineTemplate over all timestamps

	for _, se := range scheduledEventList.GetScheduledevents() {
		// Scheduled Event uses the environment we are checking
		if vmMapping, ok := se.GetRequiredVms()[environment]; ok {
			seStart, err := time.Parse(time.UnixDate, se.GetStartTime())
			if err != nil {
				return map[time.Time]map[string]uint32{}, map[string]uint32{}, fmt.Errorf("error parsing scheduled event start %v", err)
			}
			seEnd, err := time.Parse(time.UnixDate, se.GetEndTime())
			if err != nil {
				return map[time.Time]map[string]uint32{}, map[string]uint32{}, fmt.Errorf("error parsing scheduled event end %v", err)
			}
			// Scheduled Event is withing our timerange. We consider it by adding it to our Ranges
			if start.Equal(seStart) || end.Equal(seEnd) || (start.Before(seEnd) && end.After(seStart)) {
				timeRange = append(timeRange, Range{
					Start:     seStart,
					End:       seEnd,
					VMMapping: vmMapping.GetVmTemplateCounts(),
				})
				changingTimestamps = append(changingTimestamps, seStart)
				changingTimestamps = append(changingTimestamps, seEnd)
				virtualMachineCount[seStart] = make(map[string]uint32)
				virtualMachineCount[seEnd] = make(map[string]uint32)
				glog.V(4).Infof("Scheduled Event %s was within the time period", se.Name)
			}
		}
	}

	// Sort timestamps
	sortTime(changingTimestamps)

	for _, eventRange := range timeRange {
		// For any given Scheduled Event check if the timestamp is during the duration of our event. Add required Virtualmachine Counts to this timestamp.
		for _, timestamp := range changingTimestamps {
			if eventRange.Start.After(timestamp) {
				continue
			}
			if eventRange.End.Before(timestamp) {
				break
			}

			// When we are here the timestamp is in the duration of this event.
			for vmTemplateName, vmTemplateCount := range eventRange.VMMapping {
				// VM Capacity for this timestamp
				if currentVMCapacity, ok := virtualMachineCount[timestamp][vmTemplateName]; ok {
					virtualMachineCount[timestamp][vmTemplateName] = currentVMCapacity + vmTemplateCount
				} else {
					virtualMachineCount[timestamp][vmTemplateName] = vmTemplateCount
				}
				// Highest VM Capacity over all timestamps
				if maximumVMCapacity, ok := maximumVirtualMachineCount[vmTemplateName]; ok {
					maximumVirtualMachineCount[vmTemplateName] = MaxUint32(maximumVMCapacity, virtualMachineCount[timestamp][vmTemplateName])
				} else {
					maximumVirtualMachineCount[vmTemplateName] = vmTemplateCount
				}
			}

		}
	}

	return virtualMachineCount, maximumVirtualMachineCount, nil
}

func CountMachinesPerTemplateAndEnvironment(ctx context.Context, vmClient vmpb.VMSvcClient, template string, enviroment string) (int, error) {
	vmLabels := labels.Set{
		hflabels.EnvironmentLabel:       enviroment,
		hflabels.VirtualMachineTemplate: template,
	}

	vmList, err := vmClient.ListVM(ctx, &generalpb.ListOptions{LabelSelector: vmLabels.AsSelector().String(), LoadFromCache: true})
	return len(vmList.GetVms()), err
}

func CountMachinesPerTemplateAndEnvironmentAndScheduledEvent(ctx context.Context, vmClient vmpb.VMSvcClient, template string, enviroment string, se string) (int, error) {
	vmLabels := labels.Set{
		hflabels.EnvironmentLabel:       enviroment,
		hflabels.VirtualMachineTemplate: template,
		hflabels.ScheduledEventLabel:    se,
	}

	vmList, err := vmClient.ListVM(ctx, &generalpb.ListOptions{LabelSelector: vmLabels.AsSelector().String(), LoadFromCache: true})
	return len(vmList.GetVms()), err
}

func GetReleaseNamespace() string {
	provisionNS := "hobbyfarm"
	ns := os.Getenv("HF_NAMESPACE")
	if ns != "" {
		provisionNS = ns
	}
	return provisionNS
}

func GetVMConfig(env *environmentpb.Environment, vmt *vmtemplatepb.VMTemplate) map[string]string {
	envSpecificConfigFromEnv := env.EnvironmentSpecifics
	envTemplateInfo, exists := env.TemplateMapping[vmt.GetId()]

	config := make(map[string]string)
	config["image"] = vmt.GetImage()

	// First copy VMT Details (default)
	for k, v := range vmt.GetConfigMap() {
		config[k] = v
	}

	// Override with general environment specifics
	for k, v := range envSpecificConfigFromEnv {
		config[k] = v
	}

	//This environment has specifics for this vmt
	if exists {
		// Override with specific from VM on this environment
		for k, v := range envTemplateInfo.GetValue() {
			config[k] = v
		}
	}

	return config
}

func GetLock(lockName string, cfg *rest.Config) (resourcelock.Interface, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	ns := GetReleaseNamespace()
	return resourcelock.NewFromKubeconfig(resourcelock.LeasesResourceLock, ns, lockName, resourcelock.ResourceLockConfig{Identity: hostname}, cfg, 15*time.Second)
}

func GetProtoMarshaller() protojson.MarshalOptions {
	return protojson.MarshalOptions{
		EmitUnpopulated: true,
		UseProtoNames:   true,
	}
}

func StringPtr(s string) *string {
	return &s
}

// This Method converts a given duration into a valid duration for time.ParseDuration.
// time.ParseDuration does not accept "d" for days
func GetDurationWithDays(s string) (string, error) {
	// When the duration is given in days, convert it to hours instead as time.ParseDuration does not accept Days
	if strings.HasSuffix(s, "d") {
		durationWithoutSuffix := strings.TrimSuffix(s, "d")
		// string to int
		durationDays, err := strconv.Atoi(durationWithoutSuffix)
		if err != nil {
			return "", err
		}

		s = fmt.Sprintf("%dh", durationDays*24)
	}

	return s, nil
}

func AppendDynamicScenariosByCategories(
	ctx context.Context,
	scenariosList []string,
	categories []string,
	listScenariosFunc func(ctx context.Context, in *generalpb.ListOptions) (*scenariopb.ListScenariosResponse, error),
) []string {
	for _, categoryQuery := range categories {
		categorySelectors := []string{}
		categoryQueryParts := strings.Split(categoryQuery, "&")
		for _, categoryQueryPart := range categoryQueryParts {
			operator := "in"
			if strings.HasPrefix(categoryQueryPart, "!") {
				operator = "notin"
				categoryQueryPart = categoryQueryPart[1:]
			}
			categorySelectors = append(categorySelectors, fmt.Sprintf("category-%s %s (true)", categoryQueryPart, operator))
		}
		categorySelectorString := strings.Join(categorySelectors, ",")

		selector, err := labels.Parse(categorySelectorString)
		if err != nil {
			glog.Errorf("error while parsing label selector %s: %v", categorySelectorString, err)
			continue
		}

		scenarios, err := listScenariosFunc(ctx, &generalpb.ListOptions{
			LabelSelector: selector.String(),
			LoadFromCache: true,
		})

		if err != nil {
			glog.Errorf("error while retrieving scenarios: %s", hferrors.GetErrorMessage(err))
			continue
		}
		for _, scenario := range scenarios.GetScenarios() {
			scenariosList = append(scenariosList, scenario.GetId())
		}
	}

	scenariosList = UniqueStringSlice(scenariosList)
	return scenariosList
}

// This function returns a slice of all strings from the target slice, which are not also found in the input slice
func GetUniqueStringsFromSlice(targetSlice []string, inputSlice []string) []string {
	unique := make([]string, 0)
	for _, v := range targetSlice {
		idx := slices.IndexFunc(inputSlice, func(s string) bool { return s == v })
		if idx == -1 {
			unique = append(unique, v)
		}
	}
	return unique
}
