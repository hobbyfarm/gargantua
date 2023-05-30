package scenarioclient

import (
	hfv2 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v2"
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

func (sc ScenarioClient) GetScenarioById(id string) (hfv2.Scenario, error) {

	sResult, err := sc.sServer.GetScenarioById(id)

	if err != nil {
		return hfv2.Scenario{}, err
	}

	return sResult, nil
}
