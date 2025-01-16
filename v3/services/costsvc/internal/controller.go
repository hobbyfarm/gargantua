package costservice

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	costpb "github.com/hobbyfarm/gargantua/v3/protos/cost"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"strconv"
	"time"
)

type costGroup struct {
	Id                string
	Kind              string
	CostGroup         string
	BasePrice         float64
	TimeUnit          util.TimeUnit
	CreationTimestamp int64
}

func newCostGroup(obj interface{}) (*costGroup, error) {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, errors.New("failed to cast object to *unstructured.Unstructured")
	}

	objLabels := unstructuredObj.GetLabels()

	costGroupLabel, found := objLabels[labels.CostGroup]
	if !found {
		return nil, fmt.Errorf("%s label not found", labels.CostGroup)
	}
	basePriceLabel, found := objLabels[labels.CostBasePrice]
	if !found {
		return nil, fmt.Errorf("%s label not found", labels.CostBasePrice)
	}
	basePrice, err := strconv.ParseFloat(basePriceLabel, 64)
	if err != nil {
		return nil, fmt.Errorf("%s label value is not a float64", labels.CostBasePrice)
	}
	timeUnitLabel, found := objLabels[labels.CostTimeUnit]
	if !found {
		return nil, fmt.Errorf("%s label not found", labels.CostTimeUnit)
	}
	timeUnit, err := util.ParseTimeUnit(timeUnitLabel)
	if err != nil {
		return nil, fmt.Errorf("%s label value is not a valid time unit", labels.CostTimeUnit)
	}

	return &costGroup{
		Id:                unstructuredObj.GetName(),
		Kind:              unstructuredObj.GetKind(),
		CostGroup:         costGroupLabel,
		BasePrice:         basePrice,
		TimeUnit:          timeUnit,
		CreationTimestamp: unstructuredObj.GetCreationTimestamp().Unix(),
	}, nil
}

type CostController struct {
	internalCostServer *GrpcCostServer
	ctx                context.Context
}

func NewCostController(
	costServer *GrpcCostServer,
	dynamicInformerFactory dynamicinformer.DynamicSharedInformerFactory,
	ctx context.Context,
	resources ...schema.GroupVersionResource,
) *CostController {
	costController := &CostController{
		internalCostServer: costServer,
		ctx:                ctx,
	}

	for _, resource := range resources {
		informer := dynamicInformerFactory.ForResource(resource).Informer()
		_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    costController.add,
			UpdateFunc: costController.update,
			DeleteFunc: costController.delete,
		})
		if err != nil {
			glog.Fatalf("Error building label informer: %s", err.Error())
		}

	}
	return costController
}

func (li CostController) add(obj interface{}) {
	cg, err := newCostGroup(obj)
	if err != nil {
		glog.Errorf("error processing add event: %v", err)
		return
	}

	resp, err := li.internalCostServer.CreateOrUpdateCost(li.ctx, &costpb.CreateOrUpdateCostRequest{
		CostGroup:             cg.CostGroup,
		Kind:                  cg.Kind,
		BasePrice:             cg.BasePrice,
		TimeUnit:              cg.TimeUnit,
		Id:                    cg.Id,
		CreationUnixTimestamp: cg.CreationTimestamp,
		DeletionUnixTimestamp: nil,
	})
	if err != nil {
		glog.Errorf("error processing add event: %v", err)
		return
	}

	glog.Infof("resource %s created for cost group %s", resp.Id, cg.CostGroup)
}

func (li CostController) update(_, newObj interface{}) {
	cg, err := newCostGroup(newObj)
	if err != nil {
		glog.Errorf("error processing update event: %v", err)
		return
	}

	resp, err := li.internalCostServer.CreateOrUpdateCost(li.ctx, &costpb.CreateOrUpdateCostRequest{
		CostGroup:             cg.CostGroup,
		Kind:                  cg.Kind,
		BasePrice:             cg.BasePrice,
		TimeUnit:              cg.TimeUnit,
		Id:                    cg.Id,
		CreationUnixTimestamp: cg.CreationTimestamp,
		DeletionUnixTimestamp: nil,
	})
	if err != nil {
		glog.Errorf("error processing update event: %v", err)
		return
	}

	glog.Infof("resource %s updated for cost group %s", resp.Id, cg.CostGroup)
}

func (li CostController) delete(obj interface{}) {
	cg, err := newCostGroup(obj)
	if err != nil {
		glog.Errorf("error processing delete event: %v", err)
		return
	}

	resp, err := li.internalCostServer.CreateOrUpdateCost(li.ctx, &costpb.CreateOrUpdateCostRequest{
		CostGroup:             cg.CostGroup,
		Kind:                  cg.Kind,
		BasePrice:             cg.BasePrice,
		TimeUnit:              cg.TimeUnit,
		Id:                    cg.Id,
		CreationUnixTimestamp: cg.CreationTimestamp,
		DeletionUnixTimestamp: util.Ref(time.Now().Unix()),
	})
	if err != nil {
		glog.Errorf("error processing delete event: %v", err)
		return
	}

	glog.Infof("resource %s deleted for cost group %s", resp.Id, cg.CostGroup)
}
