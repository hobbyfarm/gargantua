package vmtemplateserver

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	rbac2 "github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	util2 "github.com/hobbyfarm/gargantua/v3/pkg/util"

	authnpb "github.com/hobbyfarm/gargantua/v3/protos/authn"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

const (
	resourcePlural = rbac2.ResourcePluralVMTemplate
)

type VirtualMachineTemplateServer struct {
	authnClient authnpb.AuthNClient
	authrClient authrpb.AuthRClient
	hfClientSet hfClientset.Interface
	ctx         context.Context
}

func NewVirtualMachineTemplateServer(authnClient authnpb.AuthNClient, authrClient authrpb.AuthRClient, hfClientset hfClientset.Interface, ctx context.Context) (*VirtualMachineTemplateServer, error) {
	as := VirtualMachineTemplateServer{}

	as.authnClient = authnClient
	as.authrClient = authrClient
	as.hfClientSet = hfClientset
	as.ctx = ctx
	return &as, nil
}

func (v VirtualMachineTemplateServer) getVirtualMachineTemplate(id string) (hfv1.VirtualMachineTemplate, error) {

	empty := hfv1.VirtualMachineTemplate{}

	if len(id) == 0 {
		return empty, fmt.Errorf("vm template id passed in was empty")
	}

	obj, err := v.hfClientSet.HobbyfarmV1().VirtualMachineTemplates(util2.GetReleaseNamespace()).Get(v.ctx, id, metav1.GetOptions{})
	if err != nil {
		return empty, fmt.Errorf("error while retrieving Virtual Machine Template by id: %s with error: %v", id, err)
	}

	return *obj, nil

}

func (v VirtualMachineTemplateServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/a/vmtemplate/list", v.ListFunc).Methods("GET")
	r.HandleFunc("/a/vmtemplate/{id}", v.GetFunc).Methods("GET")
	r.HandleFunc("/a/vmtemplate/create", v.CreateFunc).Methods("POST")
	r.HandleFunc("/a/vmtemplate/{id}/update", v.UpdateFunc).Methods("PUT")
	r.HandleFunc("/a/vmtemplate/{id}/delete", v.DeleteFunc).Methods("DELETE")
	glog.V(2).Infof("set up routes for admin vmtemplate server")
}

type PreparedVMTemplate struct {
	ID string `json:"id"`
	hfv1.VirtualMachineTemplateSpec
}

type PreparedVMTemplateList struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"`
}

func (v VirtualMachineTemplateServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac2.AuthenticateRequest(r, v.authnClient)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac2.AuthorizeSimple(r, v.authrClient, impersonatedUserId, rbac2.HobbyfarmPermission(resourcePlural, rbac2.VerbGet))
	if err != nil || !authrResponse.Success {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm template")
		return
	}

	vars := mux.Vars(r)

	vmtId := vars["id"]

	if len(vmtId) == 0 {
		util2.ReturnHTTPMessage(w, r, 500, "error", "no vmt id passed in")
		return
	}

	vmt, err := v.getVirtualMachineTemplate(vmtId)

	if err != nil {
		glog.Errorf("error while retrieving virtual machine template %v", err)
		util2.ReturnHTTPMessage(w, r, 500, "error", "no virtual machine template found")
		return
	}

	preparedEnvironment := PreparedVMTemplate{vmt.Name, vmt.Spec}

	encodedEnvironment, err := json.Marshal(preparedEnvironment)
	if err != nil {
		glog.Error(err)
	}
	util2.ReturnHTTPContent(w, r, 200, "success", encodedEnvironment)

	glog.V(2).Infof("retrieved vmt %s", vmt.Name)
}

func (v VirtualMachineTemplateServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac2.AuthenticateRequest(r, v.authnClient)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac2.AuthorizeSimple(r, v.authrClient, impersonatedUserId, rbac2.HobbyfarmPermission(resourcePlural, rbac2.VerbList))
	if err != nil || !authrResponse.Success {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list vmts")
		return
	}

	vmts, err := v.hfClientSet.HobbyfarmV1().VirtualMachineTemplates(util2.GetReleaseNamespace()).List(v.ctx, metav1.ListOptions{})

	if err != nil {
		glog.Errorf("error while listing all vmts %v", err)
		util2.ReturnHTTPMessage(w, r, 500, "error", "error listing all vmts")
		return
	}

	preparedVirtualMachineTemplates := []PreparedVMTemplateList{}

	for _, vmt := range vmts.Items {
		preparedVirtualMachineTemplates = append(preparedVirtualMachineTemplates, PreparedVMTemplateList{vmt.Name, vmt.Spec.Name, vmt.Spec.Image})
	}

	encodedVirtualMachineTemplates, err := json.Marshal(preparedVirtualMachineTemplates)
	if err != nil {
		glog.Error(err)
	}
	util2.ReturnHTTPContent(w, r, 200, "success", encodedVirtualMachineTemplates)

	glog.V(2).Infof("retrieved list of all environments")
}

func (v VirtualMachineTemplateServer) CreateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac2.AuthenticateRequest(r, v.authnClient)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac2.AuthorizeSimple(r, v.authrClient, impersonatedUserId, rbac2.HobbyfarmPermission(resourcePlural, rbac2.VerbCreate))
	if err != nil || !authrResponse.Success {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create vmt")
		return
	}

	name := r.PostFormValue("name")
	if name == "" {
		util2.ReturnHTTPMessage(w, r, 400, "badrequest", "missing name")
		return
	}

	image := r.PostFormValue("image")
	if image == "" {
		util2.ReturnHTTPMessage(w, r, 400, "badrequest", "missing image")
		return
	}

	configMapRaw := r.PostFormValue("config_map") // no validation, config_map not required

	vmTemplate := &hfv1.VirtualMachineTemplate{Spec: hfv1.VirtualMachineTemplateSpec{}}

	configMap := map[string]string{}
	if configMapRaw != "" {
		// attempt to decode if config_map passed in
		err := json.Unmarshal([]byte(configMapRaw), &configMap)
		if err != nil {
			glog.Errorf("error while unmarshalling config_map: %v", err)
			util2.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing config_map")
			return
		}
		// no error, assign to vmtemplate
		vmTemplate.Spec.ConfigMap = configMap
	}

	hasher := sha256.New()
	hasher.Write([]byte(name))
	sha := base32.StdEncoding.WithPadding(-1).EncodeToString(hasher.Sum(nil))[:10]
	vmTemplate.Name = "vmt-" + strings.ToLower(sha)
	vmTemplate.Spec.Name = name
	vmTemplate.Spec.Image = image

	glog.V(2).Infof("user %s creating vmtemplate", user.GetId())

	vmTemplate, err = v.hfClientSet.HobbyfarmV1().VirtualMachineTemplates(util2.GetReleaseNamespace()).Create(v.ctx, vmTemplate, metav1.CreateOptions{})
	if err != nil {
		glog.Errorf("error creating vmtemplate %v", err)
		util2.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating vmtemplate")
		return
	}

	util2.ReturnHTTPMessage(w, r, 201, "created", vmTemplate.Name)
	return
}

func (v VirtualMachineTemplateServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := rbac2.AuthenticateRequest(r, v.authnClient)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac2.AuthorizeSimple(r, v.authrClient, impersonatedUserId, rbac2.HobbyfarmPermission(resourcePlural, rbac2.VerbUpdate))
	if err != nil || !authrResponse.Success {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update vmt")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]
	if id == "" {
		util2.ReturnHTTPMessage(w, r, 400, "badrequest", "no id passed in")
		return
	}

	glog.V(2).Infof("user %s updating vmtemplate %s", user.GetId(), id)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vmTemplate, err := v.hfClientSet.HobbyfarmV1().VirtualMachineTemplates(util2.GetReleaseNamespace()).Get(v.ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			util2.ReturnHTTPMessage(w, r, http.StatusNotFound, "badrequest", "vmtemplate not found with given ID")
			return fmt.Errorf("bad")
		}

		name := r.PostFormValue("name")
		image := r.PostFormValue("image")
		configMapRaw := r.PostFormValue("config_map")

		if name != "" {
			vmTemplate.Spec.Name = name
		}

		if image != "" {
			vmTemplate.Spec.Image = image
		}

		if configMapRaw != "" {
			configMap := map[string]string{}
			err := json.Unmarshal([]byte(configMapRaw), &configMap)
			if err != nil {
				glog.Error(err)
				return fmt.Errorf("bad")
			}
			vmTemplate.Spec.ConfigMap = configMap
		}

		_, updateErr := v.hfClientSet.HobbyfarmV1().VirtualMachineTemplates(util2.GetReleaseNamespace()).Update(v.ctx, vmTemplate, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		util2.ReturnHTTPMessage(w, r, 500, "error", "error attempting to update vmtemplate")
		return
	}

	util2.ReturnHTTPMessage(w, r, 200, "updated", "")
	return
}

func (v VirtualMachineTemplateServer) DeleteFunc(w http.ResponseWriter, r *http.Request) {
	// deleting a vmtemplate requires none of the following objects having reference to it
	// - future scheduled events
	// - virtualmachines
	// - virtualmachineclaims
	// - virtualmachinesets
	user, err := rbac2.AuthenticateRequest(r, v.authnClient)
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	impersonatedUserId := user.GetId()
	authrResponse, err := rbac2.AuthorizeSimple(r, v.authrClient, impersonatedUserId, rbac2.HobbyfarmPermission(resourcePlural, rbac2.VerbDelete))
	if err != nil || !authrResponse.Success {
		util2.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to delete vmt")
		return
	}

	// first, check if the vmt exists
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		util2.ReturnHTTPMessage(w, r, 400, "badrequest", "no id passed in")
		return
	}

	glog.V(2).Infof("user %s deleting vmtemplate %s", user.GetId(), id)

	vmt, err := v.hfClientSet.HobbyfarmV1().VirtualMachineTemplates(util2.GetReleaseNamespace()).Get(v.ctx, id, metav1.GetOptions{})
	if err != nil {
		util2.ReturnHTTPMessage(w, r, http.StatusNotFound, "notfound", "no vmt found with given ID")
		return
	}

	// vmt exists, now we need to check all other objects for references
	// start with virtualmachines
	virtualmachines, err := v.hfClientSet.HobbyfarmV1().VirtualMachines(util2.GetReleaseNamespace()).List(v.ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", hflabels.VirtualMachineTemplate, vmt.Name)})
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 500, "internalerror",
			"error listing virtual machines while attempting vmt deletion")
		return
	}

	if len(virtualmachines.Items) > 0 {
		util2.ReturnHTTPMessage(w, r, 409, "conflict", "existing virtual machines reference this vmtemplate")
		return
	}

	// now check scheduledevents
	scheduledEvents, err := v.hfClientSet.HobbyfarmV1().ScheduledEvents(util2.GetReleaseNamespace()).List(v.ctx, metav1.ListOptions{})
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 500, "internalerror",
			"error listing scheduled events while attempting vmt deletion")
		return
	}

	if len(scheduledEvents.Items) > 0 {
		for _, v := range scheduledEvents.Items {
			if v.Status.Finished != true {
				// unfinished SE. Is it going on now or in the future?
				startTime, err := time.Parse(time.UnixDate, v.Spec.StartTime)
				if err != nil {
					util2.ReturnHTTPMessage(w, r, 500, "internalerror",
						"error parsing time while checking scheduledevent for conflict")
					return
				}
				endTime, err := time.Parse(time.UnixDate, v.Spec.EndTime)
				if err != nil {
					util2.ReturnHTTPMessage(w, r, 500, "internalerror",
						"error parsing time while checking scheduledevent for conflict")
					return
				}

				// if this starts in the future, or hasn't ended
				if startTime.After(time.Now()) || endTime.After(time.Now()) {
					// check for template existence
					if exists := searchForTemplateInRequiredVMs(v.Spec.RequiredVirtualMachines, vmt.Name); exists {
						// if template exists in this to-be-happening SE, we can't delete it
						util2.ReturnHTTPMessage(w, r, 409, "conflict",
							"existing or future scheduled event references this vmtemplate")
					}
				}
			}
		}
	}

	// now check virtul machine claims
	vmcList, err := v.hfClientSet.HobbyfarmV1().VirtualMachineClaims(util2.GetReleaseNamespace()).List(v.ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("virtualmachinetemplate.hobbyfarm.io/%s=%s", vmt.Name, "true"),
	})
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 500, "internalerror",
			"error listing virtual machine claims while attempting vmt deletion")
		return
	}

	if len(vmcList.Items) > 0 {
		util2.ReturnHTTPMessage(w, r, 409, "conflict",
			"existing virtual machine claims reference this vmtemplate")
		return
	}

	// now check virtualmachinesets (theoretically the VM checks above should catch this, but let's be safe)
	vmsetList, err := v.hfClientSet.HobbyfarmV1().VirtualMachineSets(util2.GetReleaseNamespace()).List(v.ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("virtualmachinetemplate.hobbyfarm.io/%s=%s", vmt.Name, "true"),
	})
	if err != nil {
		util2.ReturnHTTPMessage(w, r, 500, "internalerror",
			"error listing virtual machine sets while attempting vmt deletion")
		return
	}

	if len(vmsetList.Items) > 0 {
		util2.ReturnHTTPMessage(w, r, 409, "conflict",
			"existing virtual machine sets reference this vmtemplate")
		return
	}

	// if we get here, shouldn't be anything in our path stopping us from deleting the vmtemplate
	// so do it!
	err = v.hfClientSet.HobbyfarmV1().VirtualMachineTemplates(util2.GetReleaseNamespace()).Delete(v.ctx, vmt.Name, metav1.DeleteOptions{})
	if err != nil {
		glog.Errorf("error deleting vmtemplate: %v", err)
		util2.ReturnHTTPMessage(w, r, 500, "internalerror", "error deleting vmtemplate")
		return
	}

	util2.ReturnHTTPMessage(w, r, 200, "deleted", "vmtemplate deleted")
}

func searchForTemplateInRequiredVMs(req map[string]map[string]int, template string) bool {
	for _, v := range req {
		// k is environment, v is map[string]string
		for kk, _ := range v {
			// kk is vmtemplate, vv is count
			if kk == template {
				return true
			}
		}
	}

	return false
}
