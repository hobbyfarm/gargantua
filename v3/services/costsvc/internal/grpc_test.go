package costservice

import (
	"context"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	faketyped "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v1/fake"
	fakelisters "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1/fake"
	costpb "github.com/hobbyfarm/gargantua/v3/protos/cost"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"
	"reflect"
	"sort"
	"testing"
	"time"
)

func Test_groupByKind(t *testing.T) {
	tests := []struct {
		name  string
		input []hfv1.CostResource
		want  map[string][]hfv1.CostResource
	}{
		{
			name: "ok",
			input: []hfv1.CostResource{
				{Id: "a", Kind: "Pod"},
				{Id: "x", Kind: "Deployment"},
				{Id: "b", Kind: "Pod"},
				{Id: "y", Kind: "Deployment"},
				{Id: "c", Kind: "Pod"},
				{Id: "1", Kind: "VirtualMachine"},
			},
			want: map[string][]hfv1.CostResource{
				"Pod": {
					{Id: "a", Kind: "Pod"},
					{Id: "b", Kind: "Pod"},
					{Id: "c", Kind: "Pod"},
				},
				"Deployment": {
					{Id: "x", Kind: "Deployment"},
					{Id: "y", Kind: "Deployment"},
				},
				"VirtualMachine": {
					{Id: "1", Kind: "VirtualMachine"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := groupByKind(tt.input)

			// Sort the slices for deterministic comparison
			for k := range got {
				sort.Slice(got[k], func(i, j int) bool {
					return got[k][i].Id < got[k][j].Id
				})
			}
			for k := range tt.want {
				sort.Slice(tt.want[k], func(i, j int) bool {
					return tt.want[k][i].Id < tt.want[k][j].Id
				})
			}

			// Perform the comparison
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("groupByKind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGrpcCostServer_GetCostHistory(t *testing.T) {
	fakeClient := &faketyped.FakeHobbyfarmV1{Fake: &k8stesting.Fake{}}
	fakeCosts := &faketyped.FakeCosts{Fake: fakeClient}
	fakeCostLister := &fakelisters.FakeCostLister{}
	fakeCostLister.On("Costs", mock.Anything).Return(nil)
	server := GrpcCostServer{
		costClient: fakeCosts,
		costLister: fakeCostLister,
		costSynced: func() bool { return true },
		nowFunc:    time.Now,
	}
	expectedCost := &hfv1.Cost{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-cost-group",
		},
		Spec: hfv1.CostSpec{
			CostGroup: "my-cost-group",
			Resources: []hfv1.CostResource{
				{
					Id:                    "pod-a",
					Kind:                  "Pod",
					BasePrice:             1,
					TimeUnit:              hfv1.TimeUnitSeconds,
					CreationUnixTimestamp: 0, // ran 10 seconds and should cost 10
					DeletionUnixTimestamp: 10,
				},
				{
					Id:                    "pod-b",
					Kind:                  "Pod",
					BasePrice:             10,
					TimeUnit:              hfv1.TimeUnitMinutes,
					CreationUnixTimestamp: 0, // ran less than a minute so base price, total is 10
					DeletionUnixTimestamp: 10,
				},
				{
					Id:                    "pod-still-running",
					Kind:                  "Pod",
					BasePrice:             10,
					TimeUnit:              hfv1.TimeUnitMinutes,
					CreationUnixTimestamp: 1,
					DeletionUnixTimestamp: 0, // still running, should be filtered out
				},
				{
					Id:                    "vm-x",
					Kind:                  "VirtualMachine",
					BasePrice:             2,
					TimeUnit:              hfv1.TimeUnitSeconds,
					CreationUnixTimestamp: 10, // ran 5 seconds and should cost 10
					DeletionUnixTimestamp: 15,
				},
				{
					Id:                    "vm-y",
					Kind:                  "VirtualMachine",
					BasePrice:             50,
					TimeUnit:              hfv1.TimeUnitHours,
					CreationUnixTimestamp: 1, // ran less than an hour so base price, total is 50
					DeletionUnixTimestamp: 3,
				},
				{
					Id:                    "vm-still-running",
					Kind:                  "VirtualMachine",
					BasePrice:             123213,
					TimeUnit:              hfv1.TimeUnitHours,
					CreationUnixTimestamp: 1,
					DeletionUnixTimestamp: 0, // still running, should be filtered out
				},
			},
		},
	}
	fakeClient.Fake.PrependReactor("get", "costs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, expectedCost, nil
	})

	costs, err := server.GetCostHistory(context.TODO(), &generalpb.GetRequest{Id: "my-cost-group"})
	assert.NoError(t, err)
	assert.Equal(t, costs.GetCostGroup(), expectedCost.Name, "cost group matches")
	assert.Equal(t, costs.GetTotal(), uint64(10+10+10+50), "cost group total")

	assert.Len(t, costs.GetSource(), 2, "size of cost source")
	for _, source := range costs.Source {
		switch source.GetKind() {
		case "Pod":
			assert.Equal(t, source.GetCount(), uint64(2), "pod count")
			assert.Equal(t, source.GetCost(), uint64(20), "pod costs")
		case "VirtualMachine":
			assert.Equal(t, source.GetCount(), uint64(2), "virtual machine count")
			assert.Equal(t, source.GetCost(), uint64(60), "virtual machine costs")
		default:
			t.Errorf("unkown source kind = %s; want Pod or VirtualMachine", source.Kind)
		}
	}
}

func TestGrpcCostServer_GetCostPresent(t *testing.T) {
	now := time.Unix(10, 0)

	fakeClient := &faketyped.FakeHobbyfarmV1{Fake: &k8stesting.Fake{}}
	fakeCosts := &faketyped.FakeCosts{Fake: fakeClient}
	fakeCostLister := &fakelisters.FakeCostLister{}
	fakeCostLister.On("Costs", mock.Anything).Return(nil)
	server := GrpcCostServer{
		costClient: fakeCosts,
		costLister: fakeCostLister,
		costSynced: func() bool { return true },
		nowFunc:    func() time.Time { return now },
	}
	expectedCost := &hfv1.Cost{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-cost-group",
		},
		Spec: hfv1.CostSpec{
			CostGroup: "my-cost-group",
			Resources: []hfv1.CostResource{
				{
					Id:                    "pod-a",
					Kind:                  "Pod",
					BasePrice:             1,
					TimeUnit:              hfv1.TimeUnitSeconds,
					CreationUnixTimestamp: 0, // ran 10 seconds and should cost 10
					DeletionUnixTimestamp: 0,
				},
				{
					Id:                    "pod-b",
					Kind:                  "Pod",
					BasePrice:             10,
					TimeUnit:              hfv1.TimeUnitMinutes,
					CreationUnixTimestamp: 0, // ran less than a minute so base price, total is 10
					DeletionUnixTimestamp: 0,
				},
				{
					Id:                    "pod-terminated",
					Kind:                  "Pod",
					BasePrice:             10,
					TimeUnit:              hfv1.TimeUnitMinutes,
					CreationUnixTimestamp: 1,
					DeletionUnixTimestamp: 10, // terminated, should be filtered out
				},
				{
					Id:                    "vm-x",
					Kind:                  "VirtualMachine",
					BasePrice:             2,
					TimeUnit:              hfv1.TimeUnitSeconds,
					CreationUnixTimestamp: 5, // ran 5 seconds and should cost 10
					DeletionUnixTimestamp: 0,
				},
				{
					Id:                    "vm-y",
					Kind:                  "VirtualMachine",
					BasePrice:             50,
					TimeUnit:              hfv1.TimeUnitHours,
					CreationUnixTimestamp: 1, // ran less than an hour so base price, total is 50
					DeletionUnixTimestamp: 0,
				},
				{
					Id:                    "vm-terminated",
					Kind:                  "VirtualMachine",
					BasePrice:             123213,
					TimeUnit:              hfv1.TimeUnitHours,
					CreationUnixTimestamp: 1,
					DeletionUnixTimestamp: 10, // terminated, should be filtered out
				},
			},
		},
	}
	fakeClient.Fake.PrependReactor("get", "costs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, expectedCost, nil
	})

	costs, err := server.GetCostPresent(context.TODO(), &generalpb.GetRequest{Id: "my-cost-group"})
	assert.NoError(t, err)
	assert.Equal(t, costs.GetCostGroup(), expectedCost.Name, "cost group matches")
	assert.Equal(t, costs.GetTotal(), uint64(10+10+10+50), "cost group total")

	assert.Len(t, costs.GetSource(), 2, "size of cost source")
	for _, source := range costs.Source {
		switch source.GetKind() {
		case "Pod":
			assert.Equal(t, source.GetCount(), uint64(2), "pod count")
			assert.Equal(t, source.GetCost(), uint64(20), "pod costs")
		case "VirtualMachine":
			assert.Equal(t, source.GetCount(), uint64(2), "virtual machine count")
			assert.Equal(t, source.GetCost(), uint64(60), "virtual machine costs")
		default:
			t.Errorf("unkown source kind = %s; want Pod or VirtualMachine", source.Kind)
		}
	}
}

func TestGrpcCostServer_GetCost(t *testing.T) {
	now := time.Unix(10, 0)

	fakeClient := &faketyped.FakeHobbyfarmV1{Fake: &k8stesting.Fake{}}
	fakeCosts := &faketyped.FakeCosts{Fake: fakeClient}
	fakeCostLister := &fakelisters.FakeCostLister{}
	fakeCostLister.On("Costs", mock.Anything).Return(nil)
	server := GrpcCostServer{
		costClient: fakeCosts,
		costLister: fakeCostLister,
		costSynced: func() bool { return true },
		nowFunc:    func() time.Time { return now },
	}
	expectedCost := &hfv1.Cost{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-cost-group",
		},
		Spec: hfv1.CostSpec{
			CostGroup: "my-cost-group",
			Resources: []hfv1.CostResource{
				{
					Id:                    "pod-a",
					Kind:                  "Pod",
					BasePrice:             1,
					TimeUnit:              hfv1.TimeUnitSeconds,
					CreationUnixTimestamp: 0, // ran 10 seconds and should cost 10
					DeletionUnixTimestamp: 10,
				},
				{
					Id:                    "pod-b",
					Kind:                  "Pod",
					BasePrice:             10,
					TimeUnit:              hfv1.TimeUnitMinutes,
					CreationUnixTimestamp: 0, // ran less than a minute so base price, total is 10
					DeletionUnixTimestamp: 10,
				},
				{
					Id:                    "pod-c",
					Kind:                  "Pod",
					BasePrice:             1,
					TimeUnit:              hfv1.TimeUnitSeconds,
					CreationUnixTimestamp: 0,
					DeletionUnixTimestamp: 0, // still running, total is 10
				},
				{
					Id:                    "vm-x",
					Kind:                  "VirtualMachine",
					BasePrice:             2,
					TimeUnit:              hfv1.TimeUnitSeconds,
					CreationUnixTimestamp: 10, // ran 5 seconds and should cost 10
					DeletionUnixTimestamp: 15,
				},
				{
					Id:                    "vm-y",
					Kind:                  "VirtualMachine",
					BasePrice:             50,
					TimeUnit:              hfv1.TimeUnitHours,
					CreationUnixTimestamp: 1, // ran less than an hour so base price, total is 50
					DeletionUnixTimestamp: 3,
				},
				{
					Id:                    "vm-z",
					Kind:                  "VirtualMachine",
					BasePrice:             2,
					TimeUnit:              hfv1.TimeUnitSeconds,
					CreationUnixTimestamp: 5,
					DeletionUnixTimestamp: 0, // still running, total is 10
				},
			},
		},
	}
	fakeClient.Fake.PrependReactor("get", "costs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, expectedCost, nil
	})

	costs, err := server.GetCost(context.TODO(), &generalpb.GetRequest{Id: "my-cost-group"})
	assert.NoError(t, err)
	assert.Equal(t, costs.GetCostGroup(), expectedCost.Name, "cost group matches")
	assert.Equal(t, costs.GetTotal(), uint64(10+10+10+10+50+10), "cost group total")

	assert.Len(t, costs.GetSource(), 2, "size of cost source")
	for _, source := range costs.Source {
		switch source.GetKind() {
		case "Pod":
			assert.Equal(t, source.GetCount(), uint64(3), "pod count")
			assert.Equal(t, source.GetCost(), uint64(30), "pod costs")
		case "VirtualMachine":
			assert.Equal(t, source.GetCount(), uint64(3), "virtual machine count")
			assert.Equal(t, source.GetCost(), uint64(70), "virtual machine costs")
		default:
			t.Errorf("unkown source kind = %s; want Pod or VirtualMachine", source.Kind)
		}
	}
}

func TestGrpcCostServer_ListCost(t *testing.T) {
	now := time.Unix(10, 0)

	fakeClient := &faketyped.FakeHobbyfarmV1{Fake: &k8stesting.Fake{}}
	fakeCosts := &faketyped.FakeCosts{Fake: fakeClient}
	fakeCostLister := &fakelisters.FakeCostLister{}
	fakeCostLister.On("Costs", mock.Anything).Return(nil)
	server := GrpcCostServer{
		costClient: fakeCosts,
		costLister: fakeCostLister,
		costSynced: func() bool { return true },
		nowFunc:    func() time.Time { return now },
	}
	expectedCosts := &hfv1.CostList{
		TypeMeta: metav1.TypeMeta{},
		ListMeta: metav1.ListMeta{},
		Items: []hfv1.Cost{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "first",
				},
				Spec: hfv1.CostSpec{
					CostGroup: "first",
					Resources: []hfv1.CostResource{
						{
							Id:                    "pod-a",
							Kind:                  "Pod",
							BasePrice:             1,
							TimeUnit:              hfv1.TimeUnitSeconds,
							CreationUnixTimestamp: 0, // ran 10 seconds and should cost 10
							DeletionUnixTimestamp: 10,
						},
						{
							Id:                    "pod-b",
							Kind:                  "Pod",
							BasePrice:             10,
							TimeUnit:              hfv1.TimeUnitMinutes,
							CreationUnixTimestamp: 0, // ran less than a minute so base price, total is 10
							DeletionUnixTimestamp: 10,
						},
						{
							Id:                    "pod-still-running",
							Kind:                  "Pod",
							BasePrice:             1,
							TimeUnit:              hfv1.TimeUnitSeconds,
							CreationUnixTimestamp: 0,
							DeletionUnixTimestamp: 0, // still running, total is 10
						},
						{
							Id:                    "vm-x",
							Kind:                  "VirtualMachine",
							BasePrice:             2,
							TimeUnit:              hfv1.TimeUnitSeconds,
							CreationUnixTimestamp: 10, // ran 5 seconds and should cost 10
							DeletionUnixTimestamp: 15,
						},
						{
							Id:                    "vm-y",
							Kind:                  "VirtualMachine",
							BasePrice:             50,
							TimeUnit:              hfv1.TimeUnitHours,
							CreationUnixTimestamp: 1, // ran less than an hour so base price, total is 50
							DeletionUnixTimestamp: 3,
						},
						{
							Id:                    "vm-z",
							Kind:                  "VirtualMachine",
							BasePrice:             2,
							TimeUnit:              hfv1.TimeUnitSeconds,
							CreationUnixTimestamp: 5,
							DeletionUnixTimestamp: 0, // still running, total is 10
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "second",
				},
				Spec: hfv1.CostSpec{
					CostGroup: "second",
					Resources: []hfv1.CostResource{
						{
							Id:                    "vm-f",
							Kind:                  "VirtualMachine",
							BasePrice:             2,
							TimeUnit:              hfv1.TimeUnitSeconds,
							CreationUnixTimestamp: 10, // ran 5 seconds and should cost 10
							DeletionUnixTimestamp: 15,
						},
						{
							Id:                    "vm-g",
							Kind:                  "VirtualMachine",
							BasePrice:             50,
							TimeUnit:              hfv1.TimeUnitHours,
							CreationUnixTimestamp: 1, // ran less than an hour so base price, total is 50
							DeletionUnixTimestamp: 3,
						},
						{
							Id:                    "vm-h",
							Kind:                  "VirtualMachine",
							BasePrice:             2,
							TimeUnit:              hfv1.TimeUnitSeconds,
							CreationUnixTimestamp: 5,
							DeletionUnixTimestamp: 0, // still running, total is 10
						},
					},
				},
			},
		},
	}
	fakeClient.Fake.PrependReactor("list", "costs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, expectedCosts, nil
	})

	costs, err := server.ListCost(context.TODO(), &generalpb.ListOptions{})
	assert.NoError(t, err)

	assert.Len(t, costs.GetCosts(), 2, "size of cost groups")
	for _, cg := range costs.GetCosts() {
		switch cg.CostGroup {
		case "first":
			for _, source := range cg.Source {
				switch source.GetKind() {
				case "Pod":
					assert.Equal(t, source.GetCount(), uint64(3), "pod count")
					assert.Equal(t, source.GetCost(), uint64(30), "pod costs")
				case "VirtualMachine":
					assert.Equal(t, source.GetCount(), uint64(3), "virtual machine count")
					assert.Equal(t, source.GetCost(), uint64(70), "virtual machine costs")
				default:
					t.Errorf("unkown source kind = %s; want Pod or VirtualMachine", source.Kind)
				}
			}
		case "second":
			for _, source := range cg.Source {
				switch source.GetKind() {
				case "VirtualMachine":
					assert.Equal(t, source.GetCount(), uint64(3), "virtual machine count")
					assert.Equal(t, source.GetCost(), uint64(70), "virtual machine costs")
				default:
					t.Errorf("unkown source kind = %s; want Pod or VirtualMachine", source.Kind)
				}
			}
		default:
			t.Errorf("unkown cost group = %s; want first or second", cg.CostGroup)
		}
	}
}

func TestGrpcCostServer_CreateOrUpdateCost_create(t *testing.T) {
	fakeClient := &faketyped.FakeHobbyfarmV1{Fake: &k8stesting.Fake{}}
	fakeCosts := &faketyped.FakeCosts{Fake: fakeClient}
	gcs := &GrpcCostServer{costClient: fakeCosts}

	deletionTimestamp := int64(100)

	tests := []struct {
		name  string
		input *costpb.CreateOrUpdateCostRequest
		want  *hfv1.Cost
	}{
		{
			name: "no deletion timestamp",
			input: &costpb.CreateOrUpdateCostRequest{
				CostGroup:             "test-cost-group",
				Id:                    "test-resource-id",
				Kind:                  "VirtualMachine",
				BasePrice:             100,
				TimeUnit:              string(hfv1.TimeUnitSeconds),
				CreationUnixTimestamp: 10,
				DeletionUnixTimestamp: nil,
			},
			want: &hfv1.Cost{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cost-group",
				},
				Spec: hfv1.CostSpec{
					CostGroup: "test-cost-group",
					Resources: []hfv1.CostResource{{
						Id:                    "test-resource-id",
						Kind:                  "VirtualMachine",
						BasePrice:             100,
						TimeUnit:              hfv1.TimeUnitSeconds,
						CreationUnixTimestamp: 10,
						DeletionUnixTimestamp: 0,
					}},
				},
			},
		},
		{
			name: "with deletion timestamp",
			input: &costpb.CreateOrUpdateCostRequest{
				CostGroup:             "test-cost-group",
				Id:                    "test-resource-id",
				Kind:                  "VirtualMachine",
				BasePrice:             100,
				TimeUnit:              string(hfv1.TimeUnitSeconds),
				CreationUnixTimestamp: 10,
				DeletionUnixTimestamp: &deletionTimestamp,
			},
			want: &hfv1.Cost{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cost-group",
				},
				Spec: hfv1.CostSpec{
					CostGroup: "test-cost-group",
					Resources: []hfv1.CostResource{{
						Id:                    "test-resource-id",
						Kind:                  "VirtualMachine",
						BasePrice:             100,
						TimeUnit:              hfv1.TimeUnitSeconds,
						CreationUnixTimestamp: 10,
						DeletionUnixTimestamp: deletionTimestamp,
					}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient.Fake.PrependReactor("get", "costs", func(action k8stesting.Action) (bool, runtime.Object, error) {
				getAction := action.(k8stesting.GetAction)
				return true, nil, errors.NewNotFound(schema.GroupResource{}, getAction.GetName())
			})
			fakeClient.Fake.PrependReactor("create", "costs", func(action k8stesting.Action) (bool, runtime.Object, error) {
				createAction := action.(k8stesting.CreateAction)
				createdObj := createAction.GetObject().(*hfv1.Cost)

				assert.Equal(t, tt.want, createdObj, "created cost resource matches")

				return true, createdObj, nil
			})

			ctx := context.TODO()
			resp, err := gcs.CreateOrUpdateCost(ctx, tt.input)

			assert.NoError(t, err)
			assert.NotNil(t, resp, "response should not be nil")
			assert.Equal(t, "test-cost-group", resp.Id, "response id matches")
		})
	}
}

func TestGrpcCostServer_CreateOrUpdateCost_newResource(t *testing.T) {
	existing := &hfv1.Cost{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cost-group",
		},
		Spec: hfv1.CostSpec{
			CostGroup: "test-cost-group",
			Resources: []hfv1.CostResource{{
				Id:                    "vm-existing",
				Kind:                  "VirtualMachine",
				BasePrice:             1,
				TimeUnit:              hfv1.TimeUnitHours,
				CreationUnixTimestamp: 10,
				DeletionUnixTimestamp: 100,
			}},
		},
	}

	deletionTimestamp := int64(100)

	input := &costpb.CreateOrUpdateCostRequest{
		CostGroup:             "test-cost-group",
		Id:                    "vm-new",
		Kind:                  "VirtualMachine",
		BasePrice:             100,
		TimeUnit:              string(hfv1.TimeUnitSeconds),
		CreationUnixTimestamp: 10,
		DeletionUnixTimestamp: &deletionTimestamp,
	}

	want := &hfv1.Cost{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cost-group",
		},
		Spec: hfv1.CostSpec{
			CostGroup: "test-cost-group",
			Resources: []hfv1.CostResource{
				{
					Id:                    "vm-existing",
					Kind:                  "VirtualMachine",
					BasePrice:             1,
					TimeUnit:              hfv1.TimeUnitHours,
					CreationUnixTimestamp: 10,
					DeletionUnixTimestamp: 100,
				},
				{
					Id:                    "vm-new",
					Kind:                  "VirtualMachine",
					BasePrice:             100,
					TimeUnit:              hfv1.TimeUnitSeconds,
					CreationUnixTimestamp: 10,
					DeletionUnixTimestamp: deletionTimestamp,
				},
			},
		},
	}

	fakeClient := &faketyped.FakeHobbyfarmV1{Fake: &k8stesting.Fake{}}
	fakeCosts := &faketyped.FakeCosts{Fake: fakeClient}
	gcs := &GrpcCostServer{costClient: fakeCosts}

	fakeClient.Fake.PrependReactor("get", "costs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		getAction := action.(k8stesting.GetAction)
		assert.Equal(t, existing.Name, getAction.GetName(), "created cost resource matches")

		return true, existing, nil
	})
	fakeClient.Fake.PrependReactor("update", "costs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		// Correctly type assert to UpdateAction
		updateAction, ok := action.(k8stesting.UpdateAction)
		if !ok {
			t.Fatalf("Expected UpdateAction, got %T", action)
		}

		// Get the updated object
		updatedObj, ok := updateAction.GetObject().(*hfv1.Cost)
		if !ok {
			t.Fatalf("Expected *hfv1.Cost, got %T", updateAction.GetObject())
		}
		assert.Equal(t, want.Name, updatedObj.Name, "name matches")
		assert.Equal(t, want.Spec.CostGroup, updatedObj.Spec.CostGroup, "cost group matches")
		assert.ElementsMatch(t, want.Spec.Resources, updatedObj.Spec.Resources, "cost resource matches")

		return true, updatedObj, nil
	})

	ctx := context.TODO()
	resp, err := gcs.CreateOrUpdateCost(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, resp, "response should not be nil")
	assert.Equal(t, want.Name, resp.Id)
}

func TestGrpcCostServer_CreateOrUpdateCost_updateResource(t *testing.T) {
	existing := &hfv1.Cost{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cost-group",
		},
		Spec: hfv1.CostSpec{
			CostGroup: "test-cost-group",
			Resources: []hfv1.CostResource{{
				Id:                    "vm-existing",
				Kind:                  "VirtualMachine",
				BasePrice:             1,
				TimeUnit:              hfv1.TimeUnitHours,
				CreationUnixTimestamp: 10,
				DeletionUnixTimestamp: 100,
			}},
		},
	}

	deletionTimestamp := int64(100)

	input := &costpb.CreateOrUpdateCostRequest{
		CostGroup:             "test-cost-group",
		Id:                    "vm-existing",
		Kind:                  "VirtualMachine",
		BasePrice:             100,
		TimeUnit:              string(hfv1.TimeUnitSeconds),
		CreationUnixTimestamp: 10,
		DeletionUnixTimestamp: &deletionTimestamp,
	}

	want := &hfv1.Cost{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cost-group",
		},
		Spec: hfv1.CostSpec{
			CostGroup: "test-cost-group",
			Resources: []hfv1.CostResource{
				{
					Id:                    "vm-existing",
					Kind:                  "VirtualMachine",
					BasePrice:             100,
					TimeUnit:              hfv1.TimeUnitSeconds,
					CreationUnixTimestamp: 10,
					DeletionUnixTimestamp: deletionTimestamp,
				},
			},
		},
	}

	fakeClient := &faketyped.FakeHobbyfarmV1{Fake: &k8stesting.Fake{}}
	fakeCosts := &faketyped.FakeCosts{Fake: fakeClient}
	gcs := &GrpcCostServer{costClient: fakeCosts}

	fakeClient.Fake.PrependReactor("get", "costs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		getAction := action.(k8stesting.GetAction)
		assert.Equal(t, existing.Name, getAction.GetName(), "created cost resource matches")

		return true, existing, nil
	})
	fakeClient.Fake.PrependReactor("update", "costs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		// Correctly type assert to UpdateAction
		updateAction, ok := action.(k8stesting.UpdateAction)
		if !ok {
			t.Fatalf("Expected UpdateAction, got %T", action)
		}

		// Get the updated object
		updatedObj, ok := updateAction.GetObject().(*hfv1.Cost)
		if !ok {
			t.Fatalf("Expected *hfv1.Cost, got %T", updateAction.GetObject())
		}
		assert.Equal(t, want.Name, updatedObj.Name, "name matches")
		assert.Equal(t, want.Spec.CostGroup, updatedObj.Spec.CostGroup, "cost group matches")
		assert.ElementsMatch(t, want.Spec.Resources, updatedObj.Spec.Resources, "cost resource matches")

		return true, updatedObj, nil
	})

	ctx := context.TODO()
	resp, err := gcs.CreateOrUpdateCost(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, resp, "response should not be nil")
	assert.Equal(t, want.Name, resp.Id)
}
