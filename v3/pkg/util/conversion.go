package util

import (
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	scheduledeventpb "github.com/hobbyfarm/gargantua/v3/protos/scheduledevent"
)

type IntegerType interface {
	uint32 | int
}

// A function that converts an integer type T to an integer type U
func ConvertIntMap[T, U IntegerType](input map[string]T) map[string]U {
	output := make(map[string]U)
	for key, value := range input {
		output[key] = U(value)
	}
	return output
}

// A helper function that converts nested maps containing a struct with a map field into raw nested maps.
// E. g., it can convert map[string]*generalpb.StringMap to map[string]map[string]string.
// To make this work generic, we additionally need to pass a function which retrieves the raw map from our struct value.
// E. g., GetRawStringMap retrieves the raw map[string]string from *generalpb.StringMap.
func ConvertMapStruct[K comparable, V any, T any](input map[string]T, getMapFunc func(T) map[K]V) map[string]map[K]V {
	output := make(map[string]map[K]V, len(input))
	for key, val := range input {
		output[key] = getMapFunc(val)
	}
	return output
}

func GetRawStringMap(val *generalpb.StringMap) map[string]string {
	return val.GetValue()
}

func GetRawVMTemplateCountMap(val *scheduledeventpb.VMTemplateCountMap) map[string]uint32 {
	return val.GetVmTemplateCounts()
}

// A function that converts []*generalpb.StringMap to []map[string]string
func ConvertToStringMapSlice(stringMapSlice []*generalpb.StringMap) []map[string]string {
	output := make([]map[string]string, 0, len(stringMapSlice))
	for _, vm := range stringMapSlice {
		output = append(output, vm.GetValue())
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

// This function converts a string or underlying string type to a protobuf enum
func ConvertToPBEnum[T ~string, PB ~int32](val T, pbmap map[string]int32, dftVal PB) PB {
	v, ok := pbmap[string(val)]
	if !ok {
		return dftVal
	}
	return PB(v)
}

func ConvertToStringEnum[PB ~int32, T ~string](pbval PB, pbmap map[int32]string, dftStr T) T {
	v, ok := pbmap[int32(pbval)]
	if !ok {
		return dftStr
	}
	return T(v)
}
