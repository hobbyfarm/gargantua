package helpers

import (
	"context"
	"github.com/rancher/lasso/pkg/client"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"log/slog"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Obj interface {
	runtime.Object
	GetName() string
	GetNamespace() string
}

func UpdateStatus(ctx context.Context, client *client.Client, obj Obj) {
	if err := client.UpdateStatus(ctx, obj.GetNamespace(), obj, obj, v1.UpdateOptions{}); err != nil {
		slog.Error("updating status for object", "gvk", obj.GetObjectKind().GroupVersionKind().String(),
			"objName", obj.GetName(), "error", err.Error())
	}
}

func ReconcileFunc(f reconcile.Func) reconcile.Reconciler {
	return struct {
		reconcile.Func
	}{
		Func: f,
	}
}
