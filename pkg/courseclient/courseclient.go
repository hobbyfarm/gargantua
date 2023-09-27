package courseclient

import (
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/v3/pkg/courseserver"
)

type CourseClient struct {
	cServer *courseserver.CourseServer
}

func NewCourseClient(cServer *courseserver.CourseServer) (*CourseClient, error) {
	a := CourseClient{}

	a.cServer = cServer
	return &a, nil
}

func (cc CourseClient) GetCourseById(id string) (hfv1.Course, error) {

	cResult, err := cc.cServer.GetCourseById(id)

	if err != nil {
		return hfv1.Course{}, err
	}

	return cResult, nil
}

func (cc CourseClient) AppendDynamicScenariosByCategories(scenariosList []string, categories []string) []string {

	return cc.cServer.AppendDynamicScenariosByCategories(scenariosList, categories)
}
