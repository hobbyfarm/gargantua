package util

import (
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
)

type IntegerType interface {
	uint32 | int
}

// A function that converts an integer type T to an integer type U
func ConvertMap[T, U IntegerType](input map[string]T) map[string]U {
	output := make(map[string]U)
	for key, value := range input {
		output[key] = U(value)
	}
	return output
}

// A function that converts []*generalpb.StringMap to []map[string]string
func ConvertStringMapSlice(vms []*generalpb.StringMap) []map[string]string {
	output := make([]map[string]string, 0, len(vms))
	for _, vm := range vms {
		output = append(output, vm.GetValue())
	}
	return output
}
