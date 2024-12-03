package condition

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
)

type Condition struct {
	Status             v1.Status `json:"status"`
	Type               string    `json:"type"`
	Message            string    `json:"message"`
	Reason             string    `json:"reason"`
	LastTransitionTime v1.Time   `json:"lastTransitionTime"`
	LastUpdateTime     v1.Time   `json:"lastUpdateTime"`
}

func get(obj runtime.Object, name string) *Condition {
	ptr := reflect.ValueOf(obj).FieldByName("Status").FieldByName("Conditions").Addr()
	v := ptr.Interface().([]Condition)

	for _, vv := range v {
		if vv.Type == name {
			return &vv
		}
	}
}
