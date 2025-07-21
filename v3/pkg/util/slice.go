package util

import (
	"reflect"
	"sort"
)

// SliceSortEqual checks if both slices are equal.
func SliceSortEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aSorted := make([]string, len(a))
	bSorted := make([]string, len(b))

	copy(aSorted, a)
	copy(bSorted, b)

	sort.Strings(aSorted)
	sort.Strings(bSorted)

	return reflect.DeepEqual(aSorted, bSorted)
}
