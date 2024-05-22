package util

// type HfClientUpdate[T metav1.Object] interface {
// 	Update(ctx context.Context, id string, opts metav1.UpdateOptions) (T, error)
// }

// func GenericRemoveFinalizer[T metav1.Object, U HfClientUpdate[T]](ctx context.Context, obj T, client U, finalizer string) (err error) {
// 	finalizers := obj.GetFinalizers()
// 	if containsFinalizer(finalizers, finalizer) {
// 		newFinalizers := RemoveFinalizer(finalizers, finalizer)
// 		obj.SetFinalizers(newFinalizers)
// 		_, err = client.Update(ctx, obj.GetName(), metav1.UpdateOptions{})
// 	}
// 	return err
// }

// From ControllerUtil to save dep issues

// RemoveFinalizer accepts a slice of finalizers and removes the provided finalizer if present.
func RemoveFinalizer(finalizers []string, finalizer string) []string {
	for i := 0; i < len(finalizers); i++ {
		if finalizers[i] == finalizer {
			finalizers = append(finalizers[:i], finalizers[i+1:]...)
			i--
		}
	}
	return finalizers
}

// ContainsFinalizer checks an Object that the provided finalizer is present.
func ContainsFinalizer(finalizers []string, finalizer string) bool {
	for _, e := range finalizers {
		if e == finalizer {
			return true
		}
	}
	return false
}
