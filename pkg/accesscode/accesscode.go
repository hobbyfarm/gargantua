package accesscode

import (
	"fmt"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	"k8s.io/client-go/tools/cache"
)

const (
	acIndex = "acc.hobbyfarm.io/code-index"
)

type AccessCodeClient struct {

	hfClientSet *hfClientset.Clientset
	accessCodeIndexer cache.Indexer

}

func NewAccessCodeClient(hfClientset *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*AccessCodeClient, error) {
	acc := AccessCodeClient{}

	acc.hfClientSet = hfClientset
	inf := hfInformerFactory.Hobbyfarm().V1().AccessCodes().Informer()
	indexers := map[string]cache.IndexFunc{acIndex: acIndexer}
	inf.AddIndexers(indexers)
	acc.accessCodeIndexer = inf.GetIndexer()
	return &acc, nil
}

func (acc AccessCodeClient) GetScenarioIds(code string) ([]string, error) {
	var ids []string

	if len(code) == 0 {
		return ids, fmt.Errorf("code was empty")
	}

	obj, err := acc.accessCodeIndexer.ByIndex(acIndex, code)

	if err != nil {
		return ids, fmt.Errorf("error while retrieving access code %s %v", code, err)
	}

	if len(obj) < 1 {
		return ids, fmt.Errorf("error while retrieving access code %s", code)
	}

	accessCode, ok := obj[0].(*hfv1.AccessCode)

	if !ok {
		return ids, fmt.Errorf("error while retrieving access code %s %v", code, ok)
	}

	return accessCode.Spec.Scenarios, nil
}

func acIndexer(obj interface{}) ([]string, error) {
	accessCode, ok := obj.(*hfv1.AccessCode)
	if !ok {
		return []string{}, nil
	}
	return []string{accessCode.Spec.Code}, nil
}