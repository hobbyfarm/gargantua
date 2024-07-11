package util

import (
	"context"
	"errors"
	"time"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	hfv1 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v1"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	"google.golang.org/grpc/codes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Verify[T metav1.Object, C GenericCacheRetriever[T]](getter C, object T, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan bool, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var fromCache T
				fromCache, err := getter.Get(object.GetName())
				if err != nil {
					if apierrors.IsNotFound(err) {
						done <- true
						return
					}
					glog.Error(err)
					done <- false
					return
				}
				if ResourceVersionAtLeast(fromCache.GetResourceVersion(), object.GetResourceVersion()) {
					glog.V(5).Infof("Resource version matched for %s", object.GetName())
					done <- true
					return
				}
				time.Sleep(100 * time.Millisecond) // Wait before retrying
			}
		}
	}()

	select {
	case <-ctx.Done():
		glog.Errorf("Timeout occurred while verifying resource version for %s", object.GetName())
		return ctx.Err()
	case success := <-done:
		if success {
			return nil
		}
		return errors.New("an error occurred during verification")
	}
}

func VerifyTaskContent(vm_tasks []hfv1.VirtualMachineTasks, request proto.Message) error {
	//Verify that name, description and command are not empty
	for _, vm_task := range vm_tasks {
		if vm_task.VMName == "" {
			glog.Errorf("error vm_name (of vm_tasks) is not specified")
			return hferrors.GrpcError(codes.InvalidArgument, "vm_name for vm_tasks property is not specified", request)
		}
		for _, task := range vm_task.Tasks {
			if task.Name == "" {
				glog.Errorf("error name of task in vm_tasks is not specified")
				return hferrors.GrpcError(codes.InvalidArgument, "name of task in vm_tasks is not specified", request)
			}
			if task.Description == "" {
				glog.Errorf("error description of task in vm_tasks is not specified")
				return hferrors.GrpcError(codes.InvalidArgument, "description of task in vm_tasks is not specified", request)
			}
			if task.Command == "" || task.Command == "[]" {
				glog.Errorf("error command of task in vm_tasks is not specified")
				return hferrors.GrpcError(codes.InvalidArgument, "command of task in vm_tasks is not specified", request)
			}
		}
	}
	return nil
}
