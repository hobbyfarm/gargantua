package util

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type HfClientWatch interface {
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

func VerifyDeletion(ctx context.Context, clientWatch HfClientWatch, resourceId string) error {
	watcher, err := clientWatch.Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", resourceId),
	})
	if err != nil {
		return fmt.Errorf("unable to verify deletion of object with id %s: %v", resourceId, err)
	}
	defer watcher.Stop()
	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed unexpectedly")
			}
			switch event.Type {
			case watch.Deleted:
				// object deleted successfully
				return nil
			case watch.Error:
				return fmt.Errorf("error watching object: %v", event.Object)
			}
		case <-ctx.Done():
			// returns context.DeadlineExceeded if the context times out and context.Canceled if the context was actively canceled
			return ctx.Err()
		}
	}
}
