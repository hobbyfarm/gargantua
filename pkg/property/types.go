package property

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type DataType string

var (
	DataTypeString  DataType = "string"
	DataTypeInteger DataType = "integer"
	DataTypeFloat   DataType = "float"
	DataTypeBoolean DataType = "boolean"
)

type ValueType string

var (
	ValueTypeScalar ValueType = "scalar"
	ValueTypeArray  ValueType = "array"
	ValueTypeMap    ValueType = "map"
)

// +k8s:deepcopy-gen=true

type Property struct {
	DataType  DataType  `json:"dataType"`
	ValueType ValueType `json:"valueType"`

	SettingValidation
}

// +k8s:deepcopy-gen=true

type SettingValidation struct {
	Required    bool     `json:"required,omitempty"`
	Maximum     *float64 `json:"maximum,omitempty"`
	Minimum     *float64 `json:"minimum,omitempty"`
	MaxLength   *int64   `json:"maxLength,omitempty"`
	MinLength   *int64   `json:"minLength,omitempty"`
	Format      *string  `json:"format,omitempty"`
	Pattern     *string  `json:"pattern,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Default     *string  `json:"default,omitempty"`
	UniqueItems bool     `json:"uniqueItems,omitempty"`
}

// is there a more elegant way to solve this problem using generics?
// probably not, i'll have two problems then.

func (p Property) FromJSON(value string) (any, error) {
	if p.ValueType == ValueTypeScalar {
		switch p.DataType {
		case DataTypeFloat:
			return strconv.ParseFloat(value, 64)
		case DataTypeInteger:
			return strconv.Atoi(value)
		case DataTypeBoolean:
			return strconv.ParseBool(value)
		case DataTypeString:
			fallthrough
		default:
			return value, nil
		}
	}

	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.DisallowUnknownFields()

	if p.ValueType == ValueTypeMap {
		switch p.DataType {
		case DataTypeFloat:
			var out = map[string]float64{}
			return out, decoder.Decode(&out)
		case DataTypeInteger:
			var out = map[string]int{}
			return out, decoder.Decode(&out)
		case DataTypeString:
			var out = map[string]string{}
			return out, decoder.Decode(&out)
		case DataTypeBoolean:
			var out = map[string]bool{}
			return out, decoder.Decode(&out)
		}
	}

	if p.ValueType == ValueTypeArray {
		switch p.DataType {
		case DataTypeFloat:
			var out = []float64{}
			return out, decoder.Decode(&out)
		case DataTypeInteger:
			var out = []int{}
			return out, decoder.Decode(&out)
		case DataTypeString:
			var out = []string{}
			return out, decoder.Decode(&out)
		case DataTypeBoolean:
			var out = []bool{}
			return out, decoder.Decode(&out)
		}
	}

	return nil, fmt.Errorf("no match for datatype %s and valuetype %s", p.DataType, p.ValueType)
}

func (p Property) ToJSON(value string) ([]byte, error) {
	switch p.ValueType {
	case ValueTypeScalar:
		switch p.DataType {
		case DataTypeString:
			return []byte("\"" + value + "\""), nil
		default:
			return []byte(value), nil
		}
	default:
		return json.Marshal(value)
	}
}
