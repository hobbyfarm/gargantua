package scenarioclient

import (
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/scenarioserver"
)

type ScenarioClient struct {
	sServer *scenarioserver.ScenarioServer
}

func NewScenarioClient(sServer *scenarioserver.ScenarioServer) (*ScenarioClient, error) {
	a := ScenarioClient{}

	a.sServer = sServer
	return &a, nil
}

func (sc ScenarioClient) GetScenarioById(id string) (hfv1.Scenario, error) {

	sResult, err := sc.sServer.GetScenarioById(id)

	if err != nil {
		return hfv1.Scenario{}, err
	}

	return sResult, nil
}
