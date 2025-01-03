package costservice

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang/glog"
	v1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	costpb "github.com/hobbyfarm/gargantua/v3/protos/cost"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"strconv"
	"time"
)

const (
	LabelCostGroup = "cost-group"
	LabelBasePrice = "base-price"
	LabelTimeUnit  = "time-unit"
)

type costGroup struct {
	Id                string
	Kind              string
	CostGroup         string
	BasePrice         uint64
	TimeUnit          v1.TimeUnit
	CreationTimestamp int64
	DeletionTimestamp *int64
}

func newCostGroup(obj interface{}) (*costGroup, error) {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, errors.New("failed to cast object to *unstructured.Unstructured")
	}

	labels := unstructuredObj.GetLabels()

	costGroupLabel, found := labels[LabelCostGroup]
	if !found {
		return nil, fmt.Errorf("%s label not found", LabelCostGroup)
	}
	basePriceLabel, found := labels[LabelBasePrice]
	if !found {
		return nil, fmt.Errorf("%s label not found", LabelBasePrice)
	}
	basePrice, err := strconv.ParseUint(basePriceLabel, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%s label value is not an uint", LabelBasePrice)
	}
	timeUnitLabel, found := labels[LabelTimeUnit]
	if !found {
		return nil, fmt.Errorf("%s label not found", LabelTimeUnit)
	}
	timeUnit, err := v1.ParseTimeUnit(timeUnitLabel)
	if err != nil {
		return nil, err
	}

	var deletionTimestamp int64
	if unstructuredObj.GetDeletionTimestamp() != nil {
		deletionTimestamp = unstructuredObj.GetDeletionTimestamp().Unix()
	}

	return &costGroup{
		Id:                unstructuredObj.GetName(),
		Kind:              unstructuredObj.GetKind(),
		CostGroup:         costGroupLabel,
		BasePrice:         basePrice,
		TimeUnit:          timeUnit,
		CreationTimestamp: unstructuredObj.GetCreationTimestamp().Unix(),
		DeletionTimestamp: &deletionTimestamp,
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
		TimeUnit:              string(cg.TimeUnit),
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
		TimeUnit:              string(cg.TimeUnit),
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

	deletionTimestamp := time.Now().Unix()

	resp, err := li.internalCostServer.CreateOrUpdateCost(li.ctx, &costpb.CreateOrUpdateCostRequest{
		CostGroup:             cg.CostGroup,
		Kind:                  cg.Kind,
		BasePrice:             cg.BasePrice,
		TimeUnit:              string(cg.TimeUnit),
		Id:                    cg.Id,
		CreationUnixTimestamp: cg.CreationTimestamp,
		DeletionUnixTimestamp: &deletionTimestamp,
	})
	if err != nil {
		glog.Errorf("error processing delete event: %v", err)
		return
	}

	glog.Infof("resource %s deleted for cost group %s", resp.Id, cg.CostGroup)
}
