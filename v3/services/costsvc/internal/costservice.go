package costservice

import (
	"encoding/json"
	"fmt"
	costpb "github.com/hobbyfarm/gargantua/v3/protos/cost"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"github.com/hobbyfarm/gargantua/v3/pkg/rbac"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
)

type PreparedCost struct {
	CostGroup string               `json:"cost_group"`
	Total     float64              `json:"total"`
	Sources   []PreparedCostSource `json:"source"`
}

type PreparedCostSource struct {
	Kind  string  `json:"kind"`
	Cost  float64 `json:"cost"`
	Count uint64  `json:"count"`
}

func NewPreparedCost(cost *costpb.Cost) PreparedCost {
	sources := make([]PreparedCostSource, len(cost.GetSource()))
	for i, source := range cost.GetSource() {
		sources[i] = PreparedCostSource{
			Kind:  source.GetKind(),
			Cost:  source.GetCost(),
			Count: source.GetCount(),
		}
	}
	return PreparedCost{
		CostGroup: cost.GetCostGroup(),
		Total:     cost.GetTotal(),
		Sources:   sources,
	}
}

type PreparedCostDetail struct {
	CostGroup string                     `json:"cost_group"`
	Sources   []PreparedCostDetailSource `json:"source"`
}

type PreparedCostDetailSource struct {
	Kind                  string  `json:"kind"`
	BasePrice             float64 `json:"base_price"`
	TimeUnit              string  `json:"time_unit"`
	ID                    string  `json:"id"`
	CreationUnixTimestamp int64   `json:"creation_unix_timestamp"`
	DeletionUnixTimestamp int64   `json:"deletion_unix_timestamp,omitempty"`
}

func NewPreparedCostDetail(costDetail *costpb.CostDetail) PreparedCostDetail {
	sources := make([]PreparedCostDetailSource, len(costDetail.GetSource()))
	for i, source := range costDetail.GetSource() {
		sources[i] = PreparedCostDetailSource{
			Kind:                  source.GetKind(),
			BasePrice:             source.GetBasePrice(),
			TimeUnit:              source.TimeUnit,
			ID:                    source.GetId(),
			CreationUnixTimestamp: source.GetCreationUnixTimestamp(),
			DeletionUnixTimestamp: source.GetDeletionUnixTimestamp(),
		}
	}
	return PreparedCostDetail{
		CostGroup: costDetail.GetCostGroup(),
		Sources:   sources,
	}
}

func (cs CostServer) GetCostFunc(w http.ResponseWriter, r *http.Request) {
	_, err := rbac.AuthenticateRequest(r, cs.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get cost")
		return
	}

	vars := mux.Vars(r)

	cg := vars["cost_group"]

	if len(cg) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no cost group passed in")
		return
	}

	cost, err := cs.internalCostServer.GetCost(r.Context(), &generalpb.GetRequest{Id: cg, LoadFromCache: true})
	if err != nil {
		glog.Errorf("error retrieving cost group %s from cache: %s", cg, hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("cost group %s not found", cg)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error retrieving cost croup %s", cg)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	preparedCost := NewPreparedCost(cost)
	encodedCost, err := json.Marshal(preparedCost)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedCost)

	glog.V(2).Infof("retrieved cost %s", cost.GetCostGroup())
}

func (cs CostServer) GetCostHistoryFunc(w http.ResponseWriter, r *http.Request) {
	_, err := rbac.AuthenticateRequest(r, cs.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get cost history")
		return
	}

	vars := mux.Vars(r)

	cg := vars["cost_group"]

	if len(cg) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no cost group passed in")
		return
	}

	cost, err := cs.internalCostServer.GetCostHistory(r.Context(), &generalpb.GetRequest{Id: cg, LoadFromCache: true})
	if err != nil {
		glog.Errorf("error retrieving cost group %s from cache: %s", cg, hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("cost group %s not found", cg)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error retrieving cost croup %s", cg)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	preparedCost := NewPreparedCost(cost)
	encodedCost, err := json.Marshal(preparedCost)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedCost)

	glog.V(2).Infof("retrieved cost history %s", cost.GetCostGroup())
}

func (cs CostServer) GetCostPresentFunc(w http.ResponseWriter, r *http.Request) {
	_, err := rbac.AuthenticateRequest(r, cs.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get cost present")
		return
	}

	vars := mux.Vars(r)

	cg := vars["cost_group"]

	if len(cg) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no cost group passed in")
		return
	}

	cost, err := cs.internalCostServer.GetCostPresent(r.Context(), &generalpb.GetRequest{Id: cg, LoadFromCache: true})
	if err != nil {
		glog.Errorf("error retrieving cost group %s from cache: %s", cg, hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("cost group %s not found", cg)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error retrieving cost croup %s", cg)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	preparedCost := NewPreparedCost(cost)
	encodedCost, err := json.Marshal(preparedCost)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedCost)

	glog.V(2).Infof("retrieved cost present %s", cost.GetCostGroup())
}

func (cs CostServer) GetCostDetailFunc(w http.ResponseWriter, r *http.Request) {
	_, err := rbac.AuthenticateRequest(r, cs.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 403, "forbidden", "no access to get cost detail")
		return
	}

	vars := mux.Vars(r)

	cg := vars["cost_group"]

	if len(cg) == 0 {
		util.ReturnHTTPMessage(w, r, 400, "bad request", "no cost group passed in")
		return
	}

	costDetail, err := cs.internalCostServer.GetCostDetail(r.Context(), &generalpb.GetRequest{Id: cg, LoadFromCache: true})
	if err != nil {
		glog.Errorf("error retrieving cost group %s from cache: %s", cg, hferrors.GetErrorMessage(err))
		if hferrors.IsGrpcNotFound(err) {
			errMsg := fmt.Sprintf("cost group %s not found", cg)
			util.ReturnHTTPMessage(w, r, http.StatusNotFound, "not found", errMsg)
			return
		}
		errMsg := fmt.Sprintf("error retrieving cost croup %s", cg)
		util.ReturnHTTPMessage(w, r, http.StatusInternalServerError, "error", errMsg)
		return
	}

	preparedCostDetail := NewPreparedCostDetail(costDetail)
	encodedCost, err := json.Marshal(preparedCostDetail)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedCost)

	glog.V(2).Infof("retrieved cost detail %s", costDetail.GetCostGroup())
}

func (cs CostServer) GetAllCostListFunc(w http.ResponseWriter, r *http.Request) {
	_, err := rbac.AuthenticateRequest(r, cs.authnClient)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 401, "unauthorized", "authentication failed")
		return
	}

	costList, err := cs.internalCostServer.ListCost(r.Context(), &generalpb.ListOptions{})
	if err != nil {
		glog.Errorf("error while retrieving costs %s", hferrors.GetErrorMessage(err))
		util.ReturnHTTPMessage(w, r, 500, "error", "error retrieving costs")
		return
	}

	preparedCosts := make([]PreparedCost, len(costList.GetCosts()))
	for i, cost := range costList.GetCosts() {
		preparedCosts[i] = NewPreparedCost(cost)
	}

	encodedCosts, err := json.Marshal(preparedCosts)
	if err != nil {
		glog.Error(err)
	}
	util.ReturnHTTPContent(w, r, 200, "success", encodedCosts)

	glog.V(2).Info("retrieved cost list")
}
