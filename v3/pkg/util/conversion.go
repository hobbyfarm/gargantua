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
func ConvertToStringMapSlice(stringMapSlice []*generalpb.StringMap) []map[string]string {
	output := make([]map[string]string, 0, len(stringMapSlice))
	for _, vm := range stringMapSlice {
		output = append(output, vm.GetValue())
	}
	return output
}

// A function that converts map[string]*generalpb.StringMap to map[string]map[string]string
func ConvertToStringMapMap(stringMapMap map[string]*generalpb.StringMap) map[string]map[string]string {
	output := make(map[string]map[string]string, len(stringMapMap))
	for key, val := range stringMapMap {
		output[key] = val.GetValue()
	}
	return output
}

// A function that converts map[string]map[string]string to map[string]*generalpb.StringMap
func ConvertToStringMapStructMap(in map[string]map[string]string) map[string]*generalpb.StringMap {
	output := make(map[string]*generalpb.StringMap, len(in))
	for key, val := range in {
		output[key] = &generalpb.StringMap{Value: val}
	}
	return output
}
