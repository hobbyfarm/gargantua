package util

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
