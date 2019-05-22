package scenariosessionclient

import (
	"fmt"
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/pkg/client/clientset/versioned"
	hfInformers "github.com/hobbyfarm/gargantua/pkg/client/informers/externalversions"
	"k8s.io/client-go/tools/cache"
)

const (
	ssIndex = "ssc.hobbyfarm.io/scenariosession-id-index"
)

type ScenarioSessionClient struct {
	hfClientSet *hfClientset.Clientset
	ssIndexer cache.Indexer
}

func NewScenarioSessionClient(hfClientSet *hfClientset.Clientset, hfInformerFactory hfInformers.SharedInformerFactory) (*ScenarioSessionClient, error) {
	a := ScenarioSessionClient{}
	a.hfClientSet = hfClientSet
	inf := hfInformerFactory.Hobbyfarm().V1().ScenarioSessions().Informer()
	indexers := map[string]cache.IndexFunc{ssIndex: ssIdIndexer}
	inf.AddIndexers(indexers)
	a.ssIndexer = inf.GetIndexer()
	return &a, nil
}

func ssIdIndexer(obj interface{}) ([]string, error) {
	ss, ok := obj.(*hfv1.ScenarioSession)
	if !ok {
		return []string{}, nil
	}
	return []string{ss.Spec.Id}, nil
}

func (ss ScenarioSessionClient) GetScenarioSessionById(id string) (hfv1.ScenarioSession, error) {
	if len(id) == 0 {
		return hfv1.ScenarioSession{}, fmt.Errorf("email passed in was empty")
	}

	obj, err := ss.ssIndexer.ByIndex(ssIndex, id)
	if err != nil {
		return hfv1.ScenarioSession{}, fmt.Errorf("error while retrieving scenario session by id: %s with error: %v", id, err)
	}

	if len(obj) < 1 {
		return hfv1.ScenarioSession{}, fmt.Errorf("user not found by id: %s", id)
	}

	scenarioSession, ok := obj[0].(*hfv1.ScenarioSession)

	if !ok {
		return hfv1.ScenarioSession{}, fmt.Errorf("error while converting secnario session found by id to object: %s", id)
	}

	return *scenarioSession, nil

}