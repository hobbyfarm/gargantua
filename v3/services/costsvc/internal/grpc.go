package costservice

import (
	"context"
	"github.com/golang/glog"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfClientsetv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v1"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	listersv1 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	costpb "github.com/hobbyfarm/gargantua/v3/protos/cost"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"time"
)

type GrpcCostServer struct {
	costpb.UnimplementedCostSvcServer
	costClient hfClientsetv1.CostInterface
	costLister listersv1.CostLister
	costSynced cache.InformerSynced
	nowFunc    func() time.Time
}

func NewGrpcCostServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory) *GrpcCostServer {
	return &GrpcCostServer{
		costClient: hfClientSet.HobbyfarmV1().Costs(util.GetReleaseNamespace()),
		costLister: hfInformerFactory.Hobbyfarm().V1().Costs().Lister(),
		costSynced: hfInformerFactory.Hobbyfarm().V1().Costs().Informer().HasSynced,
		nowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (gcs *GrpcCostServer) CreateOrUpdateCost(ctx context.Context, req *costpb.CreateOrUpdateCostRequest) (*generalpb.ResourceId, error) {
	existing, err := gcs.costClient.Get(ctx, req.CostGroup, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			glog.Infof("creating new cost group %s", req.CostGroup)
			return gcs.createCost(ctx, req)
		}
		return &generalpb.ResourceId{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}

	glog.Infof("updating cost group %s", req.CostGroup)
	return gcs.updateCost(ctx, existing, req)
}

func (gcs *GrpcCostServer) createCost(ctx context.Context, req *costpb.CreateOrUpdateCostRequest) (*generalpb.ResourceId, error) {
	c := hfv1.Cost{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.CostGroup,
		},
		Spec: hfv1.CostSpec{
			CostGroup: req.CostGroup,
			Resources: []hfv1.CostResource{{
				Id:                    req.GetId(),
				Kind:                  req.GetKind(),
				BasePrice:             req.GetBasePrice(),
				TimeUnit:              hfv1.TimeUnit(req.GetTimeUnit()),
				CreationUnixTimestamp: req.GetCreationUnixTimestamp(),
				DeletionUnixTimestamp: req.GetDeletionUnixTimestamp(),
			}},
		},
	}

	resp, err := gcs.costClient.Create(ctx, &c, metav1.CreateOptions{})
	if err != nil {
		return &generalpb.ResourceId{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &generalpb.ResourceId{Id: resp.Name}, nil
}

func (gcs *GrpcCostServer) updateCost(ctx context.Context, existing *hfv1.Cost, req *costpb.CreateOrUpdateCostRequest) (*generalpb.ResourceId, error) {
	var found bool
outer:
	for i := range existing.Spec.Resources {
		resource := &existing.Spec.Resources[i]
		if resource.Kind == req.Kind && resource.Id == req.Id {
			resource.BasePrice = req.GetBasePrice()
			resource.TimeUnit = hfv1.TimeUnit(req.GetTimeUnit())
			resource.CreationUnixTimestamp = req.GetCreationUnixTimestamp()
			resource.DeletionUnixTimestamp = req.GetDeletionUnixTimestamp()
			found = true
			break outer
		}

	}

	if !found {
		existing.Spec.Resources = append(existing.Spec.Resources, hfv1.CostResource{
			Id:                    req.Id,
			Kind:                  req.GetKind(),
			BasePrice:             req.GetBasePrice(),
			TimeUnit:              hfv1.TimeUnit(req.GetTimeUnit()),
			CreationUnixTimestamp: req.GetCreationUnixTimestamp(),
			DeletionUnixTimestamp: req.GetDeletionUnixTimestamp(),
		})

	}

	resp, err := gcs.costClient.Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return &generalpb.ResourceId{}, hferrors.GrpcError(
			codes.Internal,
			err.Error(),
			req,
		)
	}
	return &generalpb.ResourceId{Id: resp.Name}, nil
}

func groupByKind(resources []hfv1.CostResource) map[string][]hfv1.CostResource {
	grouped := make(map[string][]hfv1.CostResource)

	for _, resource := range resources {
		grouped[resource.Kind] = append(grouped[resource.Kind], resource)
	}
	return grouped
}

func (gcs *GrpcCostServer) GetCostHistory(ctx context.Context, req *generalpb.GetRequest) (*costpb.Cost, error) {
	cost, err := util.GenericHfGetter(ctx, req, gcs.costClient, gcs.costLister.Costs(util.GetReleaseNamespace()), "cost", gcs.costSynced())
	if err != nil {
		return &costpb.Cost{}, err
	}

	var costSources []*costpb.CostSource

	var totalCost uint64
	now := gcs.nowFunc()

	grouped := groupByKind(cost.Spec.Resources)
	for kind, resources := range grouped {
		var costForKind uint64
		var count uint64

		for _, resource := range resources {
			if resource.DeletionUnixTimestamp == 0 {
				// skip resource which is still running
				continue
			}
			duration := resource.Duration(now)
			costForKind += resource.CalcCost(duration)
			count += 1
		}

		totalCost += costForKind
		costSources = append(costSources, &costpb.CostSource{
			Kind:  kind,
			Cost:  costForKind,
			Count: count,
		})
	}

	return &costpb.Cost{
		CostGroup: cost.Name,
		Total:     totalCost,
		Source:    costSources,
	}, nil
}

func (gcs *GrpcCostServer) GetCostPresent(ctx context.Context, req *generalpb.GetRequest) (*costpb.Cost, error) {
	cost, err := util.GenericHfGetter(ctx, req, gcs.costClient, gcs.costLister.Costs(util.GetReleaseNamespace()), "cost", gcs.costSynced())
	if err != nil {
		return &costpb.Cost{}, err
	}

	var costSources []*costpb.CostSource

	var totalCost uint64
	now := gcs.nowFunc()

	grouped := groupByKind(cost.Spec.Resources)
	for kind, resources := range grouped {
		var costForKind uint64
		var count uint64

		for _, resource := range resources {
			if resource.DeletionUnixTimestamp != 0 {
				// skip resource which is terminated
				continue
			}
			duration := resource.Duration(now)
			costForKind += resource.CalcCost(duration)
			count += 1
		}

		totalCost += costForKind
		costSources = append(costSources, &costpb.CostSource{
			Kind:  kind,
			Cost:  costForKind,
			Count: count,
		})
	}

	return &costpb.Cost{
		CostGroup: cost.Name,
		Total:     totalCost,
		Source:    costSources,
	}, nil
}

func (gcs *GrpcCostServer) GetCost(ctx context.Context, req *generalpb.GetRequest) (*costpb.Cost, error) {
	cost, err := util.GenericHfGetter(ctx, req, gcs.costClient, gcs.costLister.Costs(util.GetReleaseNamespace()), "cost", gcs.costSynced())
	if err != nil {
		return &costpb.Cost{}, err
	}

	return mapCost(cost, gcs.nowFunc()), nil
}

func mapCost(cost *hfv1.Cost, now time.Time) *costpb.Cost {
	var costSources []*costpb.CostSource

	var totalCost uint64

	grouped := groupByKind(cost.Spec.Resources)
	for kind, resources := range grouped {
		var costForKind uint64
		var count uint64

		for _, resource := range resources {
			duration := resource.Duration(now)
			costForKind += resource.CalcCost(duration)
			count += 1
		}

		totalCost += costForKind
		costSources = append(costSources, &costpb.CostSource{
			Kind:  kind,
			Cost:  costForKind,
			Count: count,
		})
	}

	return &costpb.Cost{
		CostGroup: cost.Name,
		Total:     totalCost,
		Source:    costSources,
	}
}

func (gcs *GrpcCostServer) DeleteCost(ctx context.Context, req *generalpb.ResourceId) (*emptypb.Empty, error) {
	return util.DeleteHfResource(ctx, req, gcs.costClient, "cost")
}

func (gcs *GrpcCostServer) ListCost(ctx context.Context, listOptions *generalpb.ListOptions) (*costpb.ListCostsResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var costs []hfv1.Cost
	var err error
	if !doLoadFromCache {
		var costList *hfv1.CostList
		costList, err = util.ListByHfClient(ctx, listOptions, gcs.costClient, "costs")
		if err == nil {
			costs = costList.Items
		}
	} else {
		costs, err = util.ListByCache(listOptions, gcs.costLister, "costs", gcs.costSynced())
	}
	if err != nil {
		glog.Error(err)
		return &costpb.ListCostsResponse{}, err
	}

	var preparedCosts []*costpb.Cost

	for _, cost := range costs {
		preparedCosts = append(preparedCosts, mapCost(&cost, gcs.nowFunc()))
	}

	return &costpb.ListCostsResponse{Costs: preparedCosts}, nil
}
