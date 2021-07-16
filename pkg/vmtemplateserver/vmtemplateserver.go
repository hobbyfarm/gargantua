package vmtemplateserver

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	"github.com/hobbyfarm/gargantua/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"net/http"
	"strings"
)

type VirtualMachineTemplateServer struct {
	auth        *authclient.AuthClient
	hfClientSet hfClientset.Interface
}

func NewVirtualMachineTemplateServer(authClient *authclient.AuthClient, hfClientset hfClientset.Interface) (*VirtualMachineTemplateServer, error) {
	as := VirtualMachineTemplateServer{}

	as.hfClientSet = hfClientset
	as.auth = authClient

	return &as, nil
}

func (v VirtualMachineTemplateServer) getVirtualMachineTemplate(id string) (hfv1.VirtualMachineTemplate, error) {

	empty := hfv1.VirtualMachineTemplate{}

	if len(id) == 0 {
		return empty, fmt.Errorf("vm template id passed in was empty")
	}

	obj, err := v.hfClientSet.HobbyfarmV1().VirtualMachineTemplates().Get(id, metav1.GetOptions{})
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
	glog.V(2).Infof("set up routes for admin vmtemplate server")
}

type PreparedVMTemplate struct {
	hfv1.VirtualMachineTemplateSpec
}

func (v VirtualMachineTemplateServer) GetFunc(w http.ResponseWriter, r *http.Request) {
	_, err := v.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get vm template")
		return
	}

	vars := mux.Vars(r)

	vmtId := vars["id"]

	if len(vmtId) == 0 {
		util.ReturnHTTPMessage(w, r, 500, "error", "no vmt id passed in")
		return
	}

	vmt, err := v.getVirtualMachineTemplate(vmtId)

	if err != nil {
		glog.Errorf("error while retrieving virtual machine template %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "no virtual machine template found")
		return
	}

	preparedEnvironment := PreparedVMTemplate{vmt.Spec}

	encodedEnvironment, err := json.Marshal(preparedEnvironment)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedEnvironment)

	glog.V(2).Infof("retrieved vmt %s", vmt.Name)
}

func (v VirtualMachineTemplateServer) ListFunc(w http.ResponseWriter, r *http.Request) {
	_, err := v.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to list vmts")
		return
	}

	vmts, err := v.hfClientSet.HobbyfarmV1().VirtualMachineTemplates().List(metav1.ListOptions{})

	if err != nil {
		glog.Errorf("error while listing all vmts %v", err)
		util.ReturnHTTPMessage(w, r, 500, "error", "error listing all vmts")
		return
	}

	preparedVirtualMachineTemplates := []PreparedVMTemplate{}

	for _, vmt := range vmts.Items {
		preparedVirtualMachineTemplates = append(preparedVirtualMachineTemplates, PreparedVMTemplate{vmt.Spec})
	}

	encodedVirtualMachineTemplates, err := json.Marshal(preparedVirtualMachineTemplates)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedVirtualMachineTemplates)

	glog.V(2).Infof("retrieved list of all environments")
}

func (v VirtualMachineTemplateServer) CreateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := v.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to create vmt")
		return
	}

	name := r.PostFormValue("name")
	if name == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "missing name")
		return
	}

	image := r.PostFormValue("image")
	if image == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "missing image")
		return
	}

	resourcesRaw := r.PostFormValue("resources") // no validation, resources not required
	countMapRaw := r.PostFormValue("count_map") // no validation, count_map not required

	vmTemplate := &hfv1.VirtualMachineTemplate{Spec: hfv1.VirtualMachineTemplateSpec{}}

	resources := hfv1.CMSStruct{}
	countMap := map[string]string{}
	if resourcesRaw != "" {
		// attempt to decode if resources passed in
		err := json.Unmarshal([]byte(resourcesRaw), &resources)
		if err != nil {
			glog.Errorf("error while unmarshalling resources: %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing resources")
			return
		}
		// no error, assign to vmtemplate
		vmTemplate.Spec.Resources = resources
	}

	if countMapRaw != "" {
		// attempt to decode if count_map passed in
		err := json.Unmarshal([]byte(countMapRaw), &countMap)
		if err != nil {
			glog.Errorf("error while unmarshalling count_map: %v", err)
			util.ReturnHTTPMessage(w, r, 500, "internalerror", "error parsing count_map")
			return
		}
		// no error, assign to vmtemplate
		vmTemplate.Spec.CountMap = countMap
	}

	hasher := sha256.New()
	hasher.Write([]byte(name))
	sha := base32.StdEncoding.WithPadding(-1).EncodeToString(hasher.Sum(nil))[:10]
	vmTemplate.Name = "vmt-" + strings.ToLower(sha)
	vmTemplate.Spec.Id = vmTemplate.Name
	vmTemplate.Spec.Name = name
	vmTemplate.Spec.Image = image

	glog.V(2).Infof("user %s creating vmtemplate", user.Name)

	vmTemplate, err = v.hfClientSet.HobbyfarmV1().VirtualMachineTemplates().Create(vmTemplate)
	if err != nil {
		glog.Errorf("error creating vmtemplate %v", err)
		util.ReturnHTTPMessage(w, r, 500, "internalerror", "error creating vmtemplate")
		return
	}

	util.ReturnHTTPMessage(w, r, 201, "created", vmTemplate.Name)
	return
}

func (v VirtualMachineTemplateServer) UpdateFunc(w http.ResponseWriter, r *http.Request) {
	user, err := v.auth.AuthNAdmin(w, r)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to update vmt")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]
	glog.V(2).Infof("user %s updating vmtemplate %s", user.Name, id)
	if id == "" {
		util.ReturnHTTPMessage(w, r, 400, "badrequest", "no id passed in")
		return
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vmTemplate, err := v.hfClientSet.HobbyfarmV1().VirtualMachineTemplates().Get(id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			util.ReturnHTTPMessage(w, r, 400, "badrequest", "vmtemplate not found")
			return fmt.Errorf("bad")
		}

		name := r.PostFormValue("name")
		image := r.PostFormValue("image")
		resourcesRaw:= r.PostFormValue("resources")
		countMapRaw := r.PostFormValue("count_map")

		if name != "" {
			vmTemplate.Spec.Name = name
		}

		if image != "" {
			vmTemplate.Spec.Image = image
		}

		if resourcesRaw != "" {
			cms := hfv1.CMSStruct{}
			err := json.Unmarshal([]byte(resourcesRaw), &cms)
			if err != nil {
				glog.Error(err)
				return fmt.Errorf("bad")
			}
			vmTemplate.Spec.Resources = cms
		}

		if countMapRaw != "" {
			countMap := map[string]string{}
			err := json.Unmarshal([]byte(countMapRaw), &countMap)
			if err != nil {
				glog.Error(err)
				return fmt.Errorf("bad")
			}
			vmTemplate.Spec.CountMap = countMap
		}

		_, updateErr := v.hfClientSet.HobbyfarmV1().VirtualMachineTemplates().Update(vmTemplate)
		return updateErr
	})

	if retryErr != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error attempting to update vmtemplate")
		return
	}

	util.ReturnHTTPMessage(w, r, 200, "updated", "")
	return
}

func (v VirtualMachineTemplateServer) DeleteFunc(w http.ResponseWriter, r *http.Request) {
	// deleting a vmtemplate requires no existing VMs using it
	// nor any future SEs using it.
}