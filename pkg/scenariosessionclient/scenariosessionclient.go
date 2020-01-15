package scenariosessionclient

import (
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/scenariosessionserver"
)

const (
	ssIndex = "ssc.hobbyfarm.io/scenariosession-id-index"
)

type ScenarioSessionClient struct {
	ssServer *scenariosessionserver.ScenarioSessionServer
}

func NewScenarioSessionClient(ssServer *scenariosessionserver.ScenarioSessionServer) (*ScenarioSessionClient, error) {
	a := ScenarioSessionClient{}

	a.ssServer = ssServer
	return &a, nil
}

func (ssc ScenarioSessionClient) GetScenarioSessionById(id string) (hfv1.ScenarioSession, error) {

	ssResult, err := ssc.ssServer.GetScenarioSessionById(id)

	if err != nil {
		return hfv1.ScenarioSession{}, err
	}
	return ssResult, nil
}
