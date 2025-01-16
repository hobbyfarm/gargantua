package costservice

import (
	"context"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	faketyped "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v1/fake"
	fakelisters "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v1/fake"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	costpb "github.com/hobbyfarm/gargantua/v3/protos/cost"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"
	"testing"
	"time"
)

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
					TimeUnit:              util.TimeUnitSeconds,
					CreationUnixTimestamp: 0, // ran 10 seconds and should cost 10
					DeletionUnixTimestamp: 10,
				},
				{
					Id:                    "pod-b",
					Kind:                  "Pod",
					BasePrice:             10,
					TimeUnit:              util.TimeUnitMinutes,
					CreationUnixTimestamp: 0, // ran less than a minute so base price, total is 10
					DeletionUnixTimestamp: 10,
				},
				{
					Id:                    "pod-still-running",
					Kind:                  "Pod",
					BasePrice:             10,
					TimeUnit:              util.TimeUnitMinutes,
					CreationUnixTimestamp: 1,
					DeletionUnixTimestamp: 0, // still running, should be filtered out
				},
				{
					Id:                    "vm-x",
					Kind:                  "VirtualMachine",
					BasePrice:             2,
					TimeUnit:              util.TimeUnitSeconds,
					CreationUnixTimestamp: 10, // ran 5 seconds and should cost 10
					DeletionUnixTimestamp: 15,
				},
				{
					Id:                    "vm-y",
					Kind:                  "VirtualMachine",
					BasePrice:             50,
					TimeUnit:              util.TimeUnitHours,
					CreationUnixTimestamp: 1, // ran less than an hour so base price, total is 50
					DeletionUnixTimestamp: 3,
				},
				{
					Id:                    "vm-still-running",
					Kind:                  "VirtualMachine",
					BasePrice:             123213,
					TimeUnit:              util.TimeUnitHours,
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
	assert.Equal(t, costs.GetTotal(), float64(10+10+10+50), "cost group total")

	assert.Len(t, costs.GetSource(), 2, "size of cost source")
	for _, source := range costs.Source {
		switch source.GetKind() {
		case "Pod":
			assert.Equal(t, source.GetCount(), uint64(2), "pod count")
			assert.Equal(t, source.GetCost(), float64(20), "pod costs")
		case "VirtualMachine":
			assert.Equal(t, source.GetCount(), uint64(2), "virtual machine count")
			assert.Equal(t, source.GetCost(), float64(60), "virtual machine costs")
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
					TimeUnit:              util.TimeUnitSeconds,
					CreationUnixTimestamp: 0, // ran 10 seconds and should cost 10
					DeletionUnixTimestamp: 0,
				},
				{
					Id:                    "pod-b",
					Kind:                  "Pod",
					BasePrice:             10,
					TimeUnit:              util.TimeUnitMinutes,
					CreationUnixTimestamp: 0, // ran less than a minute so base price, total is 10
					DeletionUnixTimestamp: 0,
				},
				{
					Id:                    "pod-terminated",
					Kind:                  "Pod",
					BasePrice:             10,
					TimeUnit:              util.TimeUnitMinutes,
					CreationUnixTimestamp: 1,
					DeletionUnixTimestamp: 10, // terminated, should be filtered out
				},
				{
					Id:                    "vm-x",
					Kind:                  "VirtualMachine",
					BasePrice:             2,
					TimeUnit:              util.TimeUnitSeconds,
					CreationUnixTimestamp: 5, // ran 5 seconds and should cost 10
					DeletionUnixTimestamp: 0,
				},
				{
					Id:                    "vm-y",
					Kind:                  "VirtualMachine",
					BasePrice:             50,
					TimeUnit:              util.TimeUnitHours,
					CreationUnixTimestamp: 1, // ran less than an hour so base price, total is 50
					DeletionUnixTimestamp: 0,
				},
				{
					Id:                    "vm-terminated",
					Kind:                  "VirtualMachine",
					BasePrice:             123213,
					TimeUnit:              util.TimeUnitHours,
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
	assert.Equal(t, costs.GetTotal(), float64(10+10+10+50), "cost group total")

	assert.Len(t, costs.GetSource(), 2, "size of cost source")
	for _, source := range costs.Source {
		switch source.GetKind() {
		case "Pod":
			assert.Equal(t, source.GetCount(), uint64(2), "pod count")
			assert.Equal(t, source.GetCost(), float64(20), "pod costs")
		case "VirtualMachine":
			assert.Equal(t, source.GetCount(), uint64(2), "virtual machine count")
			assert.Equal(t, source.GetCost(), float64(60), "virtual machine costs")
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
					TimeUnit:              util.TimeUnitSeconds,
					CreationUnixTimestamp: 0, // ran 10 seconds and should cost 10
					DeletionUnixTimestamp: 10,
				},
				{
					Id:                    "pod-b",
					Kind:                  "Pod",
					BasePrice:             10,
					TimeUnit:              util.TimeUnitMinutes,
					CreationUnixTimestamp: 0, // ran less than a minute so base price, total is 10
					DeletionUnixTimestamp: 10,
				},
				{
					Id:                    "pod-c",
					Kind:                  "Pod",
					BasePrice:             1,
					TimeUnit:              util.TimeUnitSeconds,
					CreationUnixTimestamp: 0,
					DeletionUnixTimestamp: 0, // still running, total is 10
				},
				{
					Id:                    "vm-x",
					Kind:                  "VirtualMachine",
					BasePrice:             2,
					TimeUnit:              util.TimeUnitSeconds,
					CreationUnixTimestamp: 10, // ran 5 seconds and should cost 10
					DeletionUnixTimestamp: 15,
				},
				{
					Id:                    "vm-y",
					Kind:                  "VirtualMachine",
					BasePrice:             50,
					TimeUnit:              util.TimeUnitHours,
					CreationUnixTimestamp: 1, // ran less than an hour so base price, total is 50
					DeletionUnixTimestamp: 3,
				},
				{
					Id:                    "vm-z",
					Kind:                  "VirtualMachine",
					BasePrice:             2,
					TimeUnit:              util.TimeUnitSeconds,
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
	assert.Equal(t, costs.GetTotal(), float64(10+10+10+10+50+10), "cost group total")

	assert.Len(t, costs.GetSource(), 2, "size of cost source")
	for _, source := range costs.Source {
		switch source.GetKind() {
		case "Pod":
			assert.Equal(t, source.GetCount(), uint64(3), "pod count")
			assert.Equal(t, source.GetCost(), float64(30), "pod costs")
		case "VirtualMachine":
			assert.Equal(t, source.GetCount(), uint64(3), "virtual machine count")
			assert.Equal(t, source.GetCost(), float64(70), "virtual machine costs")
		default:
			t.Errorf("unkown source kind = %s; want Pod or VirtualMachine", source.Kind)
		}
	}
}

func TestGrpcCostServer_GetCostDetail(t *testing.T) {
	fakeClient := &faketyped.FakeHobbyfarmV1{Fake: &k8stesting.Fake{}}
	fakeCosts := &faketyped.FakeCosts{Fake: fakeClient}
	fakeCostLister := &fakelisters.FakeCostLister{}
	fakeCostLister.On("Costs", mock.Anything).Return(nil)
	server := GrpcCostServer{
		costClient: fakeCosts,
		costLister: fakeCostLister,
		costSynced: func() bool { return true },
	}
	givenCost := &hfv1.Cost{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-cost-group",
		},
		Spec: hfv1.CostSpec{
			CostGroup: "my-cost-group",
			Resources: []hfv1.CostResource{
				{
					Id:                    "pod-a",
					Kind:                  "Pod",
					BasePrice:             0.1,
					TimeUnit:              util.TimeUnitSeconds,
					CreationUnixTimestamp: 10,
					DeletionUnixTimestamp: 0,
				},
				{
					Id:                    "vm-a",
					Kind:                  "VirtualMachine",
					BasePrice:             10.111,
					TimeUnit:              util.TimeUnitMinutes,
					CreationUnixTimestamp: 20,
					DeletionUnixTimestamp: 200,
				},
			},
		},
	}
	expectedCostDetail := costpb.CostDetail{
		CostGroup: "my-cost-group",
		Source: []*costpb.CostDetailSource{
			{
				Kind:                  "Pod",
				BasePrice:             0.1,
				TimeUnit:              util.TimeUnitSeconds,
				Id:                    "pod-a",
				CreationUnixTimestamp: 10,
				DeletionUnixTimestamp: nil,
			},
			{
				Kind:                  "VirtualMachine",
				BasePrice:             10.111,
				TimeUnit:              util.TimeUnitMinutes,
				Id:                    "vm-a",
				CreationUnixTimestamp: 20,
				DeletionUnixTimestamp: util.Ref(int64(200)),
			},
		},
	}
	fakeClient.Fake.PrependReactor("get", "costs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, givenCost, nil
	})

	costs, err := server.GetCostDetail(context.TODO(), &generalpb.GetRequest{Id: "my-cost-group"})
	assert.NoError(t, err)
	assert.Equal(t, costs.GetCostGroup(), expectedCostDetail.CostGroup, "cost group matches")
	assert.ElementsMatch(t, expectedCostDetail.GetSource(), costs.GetSource(), "source matches")
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
							TimeUnit:              util.TimeUnitSeconds,
							CreationUnixTimestamp: 0, // ran 10 seconds and should cost 10
							DeletionUnixTimestamp: 10,
						},
						{
							Id:                    "pod-b",
							Kind:                  "Pod",
							BasePrice:             10,
							TimeUnit:              util.TimeUnitMinutes,
							CreationUnixTimestamp: 0, // ran less than a minute so base price, total is 10
							DeletionUnixTimestamp: 10,
						},
						{
							Id:                    "pod-still-running",
							Kind:                  "Pod",
							BasePrice:             1,
							TimeUnit:              util.TimeUnitSeconds,
							CreationUnixTimestamp: 0,
							DeletionUnixTimestamp: 0, // still running, total is 10
						},
						{
							Id:                    "vm-x",
							Kind:                  "VirtualMachine",
							BasePrice:             2,
							TimeUnit:              util.TimeUnitSeconds,
							CreationUnixTimestamp: 10, // ran 5 seconds and should cost 10
							DeletionUnixTimestamp: 15,
						},
						{
							Id:                    "vm-y",
							Kind:                  "VirtualMachine",
							BasePrice:             50,
							TimeUnit:              util.TimeUnitHours,
							CreationUnixTimestamp: 1, // ran less than an hour so base price, total is 50
							DeletionUnixTimestamp: 3,
						},
						{
							Id:                    "vm-z",
							Kind:                  "VirtualMachine",
							BasePrice:             2,
							TimeUnit:              util.TimeUnitSeconds,
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
							TimeUnit:              util.TimeUnitSeconds,
							CreationUnixTimestamp: 10, // ran 5 seconds and should cost 10
							DeletionUnixTimestamp: 15,
						},
						{
							Id:                    "vm-g",
							Kind:                  "VirtualMachine",
							BasePrice:             50,
							TimeUnit:              util.TimeUnitHours,
							CreationUnixTimestamp: 1, // ran less than an hour so base price, total is 50
							DeletionUnixTimestamp: 3,
						},
						{
							Id:                    "vm-h",
							Kind:                  "VirtualMachine",
							BasePrice:             2,
							TimeUnit:              util.TimeUnitSeconds,
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
					assert.Equal(t, source.GetCost(), float64(30), "pod costs")
				case "VirtualMachine":
					assert.Equal(t, source.GetCount(), uint64(3), "virtual machine count")
					assert.Equal(t, source.GetCost(), float64(70), "virtual machine costs")
				default:
					t.Errorf("unkown source kind = %s; want Pod or VirtualMachine", source.Kind)
				}
			}
		case "second":
			for _, source := range cg.Source {
				switch source.GetKind() {
				case "VirtualMachine":
					assert.Equal(t, source.GetCount(), uint64(3), "virtual machine count")
					assert.Equal(t, source.GetCost(), float64(70), "virtual machine costs")
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
				TimeUnit:              util.TimeUnitSeconds,
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
						TimeUnit:              util.TimeUnitSeconds,
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
				TimeUnit:              util.TimeUnitSeconds,
				CreationUnixTimestamp: 10,
				DeletionUnixTimestamp: util.Ref(int64(100)),
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
						TimeUnit:              util.TimeUnitSeconds,
						CreationUnixTimestamp: 10,
						DeletionUnixTimestamp: 100,
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
				TimeUnit:              util.TimeUnitHours,
				CreationUnixTimestamp: 10,
				DeletionUnixTimestamp: 100,
			}},
		},
	}

	input := &costpb.CreateOrUpdateCostRequest{
		CostGroup:             "test-cost-group",
		Id:                    "vm-new",
		Kind:                  "VirtualMachine",
		BasePrice:             100,
		TimeUnit:              util.TimeUnitSeconds,
		CreationUnixTimestamp: 10,
		DeletionUnixTimestamp: util.Ref(int64(100)),
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
					TimeUnit:              util.TimeUnitHours,
					CreationUnixTimestamp: 10,
					DeletionUnixTimestamp: 100,
				},
				{
					Id:                    "vm-new",
					Kind:                  "VirtualMachine",
					BasePrice:             100,
					TimeUnit:              util.TimeUnitSeconds,
					CreationUnixTimestamp: 10,
					DeletionUnixTimestamp: 100,
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
				TimeUnit:              util.TimeUnitHours,
				CreationUnixTimestamp: 10,
				DeletionUnixTimestamp: 100,
			}},
		},
	}

	input := &costpb.CreateOrUpdateCostRequest{
		CostGroup:             "test-cost-group",
		Id:                    "vm-existing",
		Kind:                  "VirtualMachine",
		BasePrice:             100,
		TimeUnit:              util.TimeUnitSeconds,
		CreationUnixTimestamp: 10,
		DeletionUnixTimestamp: util.Ref(int64(100)),
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
					TimeUnit:              util.TimeUnitSeconds,
					CreationUnixTimestamp: 10,
					DeletionUnixTimestamp: 100,
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
