package coursesessionclient

import (
	hfv1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/coursesessionserver"
)

const (
	ssIndex = "ssc.hobbyfarm.io/coursesession-id-index"
)

type CourseSessionClient struct {
	ssServer *coursesessionserver.CourseSessionServer
}

func NewCourseSessionClient(ssServer *coursesessionserver.CourseSessionServer) (*CourseSessionClient, error) {
	a := CourseSessionClient{}

	a.ssServer = ssServer
	return &a, nil
}

func (ssc CourseSessionClient) GetCourseSessionById(id string) (hfv1.CourseSession, error) {

	ssResult, err := ssc.ssServer.GetCourseSessionById(id)

	if err != nil {
		return hfv1.CourseSession{}, err
	}
	return ssResult, nil
}
