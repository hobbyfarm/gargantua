package vmtemplateserver

import (
	"fmt"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/authclient"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	"k8s.io/client-go/tools/cache"
)

const (
	idIndex   = "vmts.hobbyfarm.io/id-index"
	nameIndex = "vmts.hobbyfarm.io/name-index"
)

type VMTemplateServer struct {
	auth        *authclient.AuthClient
	hfClientSet *hfClientset.Clientset

	vmTemplateIndexer cache.Indexer
}

func NewVMTemplateServer(authClient *authclient.AuthClient, hfClientset *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*VMTemplateServer, error) {
	vmts := VMTemplateServer{}

	vmts.hfClientSet = hfClientset
	vmts.auth = authClient

	inf := hfInformerFactory.Hobbyfarm().V1().VirtualMachineTemplates().Informer()
	indexers := map[string]cache.IndexFunc{idIndex: vmtIdIndexer, nameIndex: vmtNameIndexer}
	inf.AddIndexers(indexers)
	vmts.vmTemplateIndexer = inf.GetIndexer()

	return &vmts, nil
}

func (vmts VMTemplateServer) GetVirtualMachineTemplateById(id string) (hfv1.VirtualMachineTemplate, error) {

	empty := hfv1.VirtualMachineTemplate{}

	if len(id) == 0 {
		return empty, fmt.Errorf("vm template id passed in was empty")
	}

	obj, err := vmts.vmTemplateIndexer.ByIndex(idIndex, id)
	if err != nil {
		return empty, fmt.Errorf("error while retrieving virtualmachinetemplate by id: %s with error: %v", id, err)
	}

	if len(obj) < 1 {
		return empty, fmt.Errorf("virtualmachinetemplate not found by id: %s", id)
	}

	result, ok := obj[0].(*hfv1.VirtualMachineTemplate)

	if !ok {
		return empty, fmt.Errorf("error while converting virtualmachinetemplate found by id to object: %s", id)
	}

	return *result, nil

}

func (vmts VMTemplateServer) GetVirtualMachineTemplateByName(name string) (hfv1.VirtualMachineTemplate, error) {

	empty := hfv1.VirtualMachineTemplate{}

	if len(name) == 0 {
		return empty, fmt.Errorf("vm template name passed in was empty")
	}

	obj, err := vmts.vmTemplateIndexer.ByIndex(nameIndex, name)
	if err != nil {
		return empty, fmt.Errorf("error while retrieving virtualmachinetemplate by name: %s with error: %v", name, err)
	}

	if len(obj) < 1 {
		return empty, fmt.Errorf("virtualmachinetemplate not found by name: %s", name)
	}

	result, ok := obj[0].(*hfv1.VirtualMachineTemplate)

	if !ok {
		return empty, fmt.Errorf("error while converting virtualmachinetemplate found by name to object: %s", name)
	}

	return *result, nil

}

func vmtIdIndexer(obj interface{}) ([]string, error) {
	vmt, ok := obj.(*hfv1.VirtualMachineTemplate)
	if !ok {
		return []string{}, nil
	}
	return []string{vmt.Spec.Id}, nil
}

func vmtNameIndexer(obj interface{}) ([]string, error) {
	vmt, ok := obj.(*hfv1.VirtualMachineTemplate)
	if !ok {
		return []string{}, nil
	}
	return []string{vmt.Spec.Name}, nil
}
