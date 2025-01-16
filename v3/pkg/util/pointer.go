package util

import "reflect"

func IsDefaultValue[T comparable](v T) bool {
	var defaultValue T
	return reflect.DeepEqual(v, defaultValue)
}

// RefOrNil returns a pointer to the value if it's not the default value, otherwise returns nil
func RefOrNil[T comparable](v T) *T {
	if IsDefaultValue(v) {
		return nil
	}
	return &v
}

// Ref returns the pointer to the value
func Ref[T any](v T) *T {
	return &v
}

// Deref returns the value of the pointer
func Deref[T any](ref *T) T {
	return *ref
}
