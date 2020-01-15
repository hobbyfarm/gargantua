package vmclient

import (
	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/vmclaimserver"
)

type VirtualMachineClaimClient struct {
	vmClaimServer *vmclaimserver.VMClaimServer
}

func NewVirtualMachineClaimClient(vmClaimServer *vmclaimserver.VMClaimServer) (*VirtualMachineClaimClient, error) {
	a := VirtualMachineClaimClient{}
	a.vmClaimServer = vmClaimServer
	return &a, nil
}

func (vmc VirtualMachineClaimClient) GetVirtualMachineById(id string) (hfv1.VirtualMachineClaim, error) {

	vmcResult, err := vmc.vmClaimServer.GetVirtualMachineClaimById(id)

	if err != nil {
		glog.Errorf("error while retrieving vmc by id %s %v", id, err)
		return vmcResult, err
	}

	return vmcResult, nil

}
