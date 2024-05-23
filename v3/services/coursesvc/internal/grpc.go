package courseservice

import (
	"context"

	coursepb "github.com/hobbyfarm/gargantua/v3/protos/course"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"

	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfClientsetv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	listersv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

type GrpcCourseServer struct {
	coursepb.UnimplementedCourseSvcServer
	courseClient hfClientsetv1.CourseInterface
	courseLister listersv1.CourseLister
	courseSynced cache.InformerSynced
}

func NewGrpcCourseServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) *GrpcCourseServer {
	return &GrpcCourseServer{
		courseClient: hfClientSet.HobbyfarmV1().Courses(util.GetReleaseNamespace()),
		courseLister: hfInformerFactory.Hobbyfarm().V1().Courses().Lister(),
		courseSynced: hfInformerFactory.Hobbyfarm().V1().Courses().Informer().HasSynced,
	}
}

func (c *GrpcCourseServer) CreateCourse(ctx context.Context, req *coursepb.CreateCourseRequest) (*emptypb.Empty, error) {
	name := req.GetName()
	description := req.GetDescription()
	rawScenarios := req.GetRawScenarios()
	rawCategories := req.GetRawCategories()
	rawVirtualMachines := req.GetRawVms()
	keepaliveDuration := req.GetKeepaliveDuration()
	pauseDuration := req.GetPauseDuration()
	pausable := req.GetPausable()
	keepVm := req.GetKeepVm()

	requiredStringParams := map[string]string{
		"name":        name,
		"description": description,
	}
	for param, value := range requiredStringParams {
		if value == "" {
			return &emptypb.Empty{}, hferrors.GrpcNotSpecifiedError(req, param)
		}
	}

	id := util.GenerateResourceName("c", name, 10)

	course := &hfv1.Course{
		ObjectMeta: metav1.ObjectMeta{
			Name: id,
		},
		Spec: hfv1.CourseSpec{
			Name:              name,
			Description:       description,
			KeepAliveDuration: keepaliveDuration,
			PauseDuration:     pauseDuration,
			Pauseable:         pausable,
			KeepVM:            keepVm,
		},
	}

	if rawScenarios != "" {
		scenarios, err := util.GenericUnmarshal[[]string](rawScenarios, "rawScenarios")
		if err != nil {
			return &emptypb.Empty{}, hferrors.GrpcParsingError(req, "rawScenarios")
		}
		course.Spec.Scenarios = scenarios
	}
	if rawCategories != "" {
		categories, err := util.GenericUnmarshal[[]string](rawCategories, "rawCategories")
		if err != nil {
			return &emptypb.Empty{}, hferrors.GrpcParsingError(req, "rawCategories")
		}
		course.Spec.Categories = categories
	}
	if rawVirtualMachines != "" {
		vms, err := util.GenericUnmarshal[[]map[string]string](rawVirtualMachines, "rawVirtualMachines")
		if err != nil {
			return &emptypb.Empty{}, hferrors.GrpcParsingError(req, "rawVirtualMachines")
		}
		course.Spec.VirtualMachines = vms
	}

	_, err := c.courseClient.Create(ctx, course, metav1.CreateOptions{})
	if err != nil {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &emptypb.Empty{}, nil
}

func (c *GrpcCourseServer) GetCourse(ctx context.Context, req *generalpb.GetRequest) (*coursepb.Course, error) {
	course, err := util.GenericHfGetter(ctx, req, c.courseClient, c.courseLister.Courses(util.GetReleaseNamespace()), "course", c.courseSynced())
	if err != nil {
		return &coursepb.Course{}, err
	}

	vms := []*generalpb.StringMap{}
	for _, vm := range course.Spec.VirtualMachines {
		vms = append(vms, &generalpb.StringMap{Value: vm})
	}

	return &coursepb.Course{
		Id:                course.Name,
		Uid:               string(course.UID),
		Name:              course.Spec.Name,
		Description:       course.Spec.Description,
		Scenarios:         course.Spec.Scenarios,
		Categories:        course.Spec.Categories,
		Vms:               vms,
		KeepaliveDuration: course.Spec.KeepAliveDuration,
		PauseDuration:     course.Spec.PauseDuration,
		Pausable:          course.Spec.Pauseable,
		KeepVm:            course.Spec.KeepVM,
	}, nil
}

func (s *GrpcCourseServer) UpdateCourse(ctx context.Context, req *coursepb.UpdateCourseRequest) (*emptypb.Empty, error) {
	id := req.GetId()
	if len(id) == 0 {
		return &emptypb.Empty{}, hferrors.GrpcIdNotSpecifiedError(req)
	}
	name := req.GetName()
	description := req.GetDescription()
	rawScenarios := req.GetRawScenarios()
	rawCategories := req.GetRawCategories()
	rawVirtualMachines := req.GetRawVms()
	keepaliveDuration := req.GetKeepaliveDuration()
	pauseDuration := req.GetPauseDuration()
	pausable := req.GetPausable()
	keepVm := req.GetKeepVm()

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		course, err := s.courseClient.Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving course %s",
				req,
				req.GetId(),
			)
		}
		if name != "" {
			course.Spec.Name = name
		}
		if description != "" {
			course.Spec.Description = description
		}
		if keepaliveDuration != nil {
			course.Spec.KeepAliveDuration = keepaliveDuration.GetValue()
		}
		if pauseDuration != nil {
			course.Spec.PauseDuration = pauseDuration.GetValue()
		}
		if pausable != nil {
			course.Spec.Pauseable = pausable.GetValue()
		}
		if keepVm != nil {
			course.Spec.KeepVM = keepVm.GetValue()
		}
		if rawScenarios != "" {
			scenarios, err := util.GenericUnmarshal[[]string](rawScenarios, "rawScenarios")
			if err != nil {
				return hferrors.GrpcParsingError(req, "rawScenarios")
			}
			course.Spec.Scenarios = scenarios
		}
		if rawCategories != "" {
			categories, err := util.GenericUnmarshal[[]string](rawCategories, "rawCategories")
			if err != nil {
				return hferrors.GrpcParsingError(req, "rawCategories")
			}
			course.Spec.Categories = categories
		}
		if rawVirtualMachines != "" {
			vms, err := util.GenericUnmarshal[[]map[string]string](rawVirtualMachines, "rawVirtualMachines")
			if err != nil {
				return hferrors.GrpcParsingError(req, "rawVirtualMachines")
			}
			course.Spec.VirtualMachines = vms
		}

		_, updateErr := s.courseClient.Update(ctx, course, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error attempting to update",
			req,
		)
	}

	return &emptypb.Empty{}, nil
}

func (s *GrpcCourseServer) DeleteCourse(ctx context.Context, req *generalpb.ResourceId) (*emptypb.Empty, error) {
	return util.DeleteHfResource(ctx, req, s.courseClient, "course")
}

func (s *GrpcCourseServer) DeleteCollectionCourse(ctx context.Context, listOptions *generalpb.ListOptions) (*emptypb.Empty, error) {
	return util.DeleteHfCollection(ctx, listOptions, s.courseClient, "courses")
}

func (s *GrpcCourseServer) ListCourse(ctx context.Context, listOptions *generalpb.ListOptions) (*coursepb.ListCoursesResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var courses []hfv1.Course
	var err error
	if !doLoadFromCache {
		var courseList *hfv1.CourseList
		courseList, err = util.ListByHfClient(ctx, listOptions, s.courseClient, "courses")
		if err == nil {
			courses = courseList.Items
		}
	} else {
		courses, err = util.ListByCache(listOptions, s.courseLister, "courses", s.courseSynced())
	}
	if err != nil {
		glog.Error(err)
		return &coursepb.ListCoursesResponse{}, err
	}

	preparedCourses := []*coursepb.Course{}

	for _, course := range courses {

		vms := []*generalpb.StringMap{}
		for _, vm := range course.Spec.VirtualMachines {
			vms = append(vms, &generalpb.StringMap{Value: vm})
		}

		preparedCourses = append(preparedCourses, &coursepb.Course{
			Id:                course.Name,
			Uid:               string(course.UID),
			Name:              course.Spec.Name,
			Description:       course.Spec.Description,
			Scenarios:         course.Spec.Scenarios,
			Categories:        course.Spec.Categories,
			Vms:               vms,
			KeepaliveDuration: course.Spec.KeepAliveDuration,
			PauseDuration:     course.Spec.PauseDuration,
			Pausable:          course.Spec.Pauseable,
			KeepVm:            course.Spec.KeepVM,
		})
	}

	return &coursepb.ListCoursesResponse{Courses: preparedCourses}, nil
}
