package sessionclient

import (
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/sessionserver"
)

const (
	ssIndex = "ssc.hobbyfarm.io/session-id-index"
)

type SessionClient struct {
	ssServer *sessionserver.SessionServer
}

func NewSessionClient(ssServer *sessionserver.SessionServer) (*SessionClient, error) {
	a := SessionClient{}

	a.ssServer = ssServer
	return &a, nil
}

func (ssc SessionClient) GetSessionById(id string) (hfv1.Session, error) {

	ssResult, err := ssc.ssServer.GetSessionById(id)

	if err != nil {
		return hfv1.Session{}, err
	}
	return ssResult, nil
}
