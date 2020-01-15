package vmclient

import (
	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/vmserver"
)

type VirtualMachineClient struct {
	vmServer *vmserver.VMServer
}

func NewVirtualMachineClient(vmServer *vmserver.VMServer) (*VirtualMachineClient, error) {
	a := VirtualMachineClient{}
	a.vmServer = vmServer
	return &a, nil
}

func (vm VirtualMachineClient) GetVirtualMachineById(id string) (hfv1.VirtualMachine, error) {

	vmResult, err := vm.vmServer.GetVirtualMachineById(id)

	if err != nil {
		glog.Errorf("error while retrieving vm by id %s %v", id, err)
		return vmResult, err
	}

	return vmResult, nil

}
