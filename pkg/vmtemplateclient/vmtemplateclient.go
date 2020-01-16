package vmtemplateclient

import (
	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/vmtemplateserver"
)

type VirtualMachineTemplateClient struct {
	vmTemplateServer *vmtemplateserver.VMTemplateServer
}

func NewVirtualMachineTemplateClient(vmTemplateServer *vmtemplateserver.VMTemplateServer) (*VirtualMachineTemplateClient, error) {
	a := VirtualMachineTemplateClient{}
	a.vmTemplateServer = vmTemplateServer
	return &a, nil
}

func (vmtc VirtualMachineTemplateClient) GetVirtualMachineTemplateById(id string) (hfv1.VirtualMachineTemplate, error) {

	vmtResult, err := vmtc.vmTemplateServer.GetVirtualMachineTemplateById(id)

	if err != nil {
		glog.Errorf("error while retrieving vmt by id %s %v", id, err)
		return vmtResult, err
	}

	return vmtResult, nil

}

func (vmtc VirtualMachineTemplateClient) GetVirtualMachineTemplateByName(name string) (hfv1.VirtualMachineTemplate, error) {

	vmtResult, err := vmtc.vmTemplateServer.GetVirtualMachineTemplateByName(name)

	if err != nil {
		glog.Errorf("error while retrieving vmt by name %s %v", name, err)
		return vmtResult, err
	}

	return vmtResult, nil

}
