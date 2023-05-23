package setting

import (
	"fmt"
	v1 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"testing"
)

func Test_ValidateValue(t *testing.T) {
	var stringVar = "testing"
	var intVar = "123"
	var floatVar = "1.234"
	var boolVar = "false"

	var arrayStringVar = "[\"one\", \"two\", \"three\"]"
	var arrayIntVar = "[1,2,3]"
	var arrayFloatVar = "[1.234, 4.321]"
	var arrayBoolVar = "[false, true, false]"

	var mapStringVar = "{\"one\": \"one\", \"two\": \"two\"}"
	var mapIntVar = "{\"one\": 1, \"two\": 2}"
	var mapFloatVar = "{\"one\": 1.234, \"two\": 2.345}"
	var mapBoolVar = "{\"one\": true, \"two\": false}"

	type test struct {
		input        string
		dataType     v1.DataType
		variableType v1.VariableType
	}

	tests := []test{
		{stringVar, v1.DataTypeString, v1.VariableTypeScalar},
		{intVar, v1.DataTypeInteger, v1.VariableTypeScalar},
		{floatVar, v1.DataTypeFloat, v1.VariableTypeScalar},
		{boolVar, v1.DataTypeBoolean, v1.VariableTypeScalar},
		{arrayStringVar, v1.DataTypeString, v1.VariableTypeArray},
		{arrayIntVar, v1.DataTypeInteger, v1.VariableTypeArray},
		{arrayFloatVar, v1.DataTypeFloat, v1.VariableTypeArray},
		{arrayBoolVar, v1.DataTypeBoolean, v1.VariableTypeArray},
		{mapStringVar, v1.DataTypeString, v1.VariableTypeMap},
		{mapIntVar, v1.DataTypeInteger, v1.VariableTypeMap},
		{mapFloatVar, v1.DataTypeFloat, v1.VariableTypeMap},
		{mapBoolVar, v1.DataTypeBoolean, v1.VariableTypeMap},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("testing %s/%s", tt.variableType, tt.dataType), func(t *testing.T) {
			if err := validateValue(tt.input, tt.dataType, tt.variableType); err != nil {
				t.Error(err)
			}
		})
	}

	badTests := []test{
		{"false", v1.DataTypeInteger, v1.VariableTypeScalar},
		{"1.234", v1.DataTypeInteger, v1.VariableTypeScalar},
		{"[]", v1.DataTypeInteger, v1.VariableTypeScalar},
		{"[123]", v1.DataTypeInteger, v1.VariableTypeScalar},
		{"one", v1.DataTypeInteger, v1.VariableTypeScalar},
		{"{\"one\": 1}", v1.DataTypeInteger, v1.VariableTypeScalar},

		{"zero", v1.DataTypeBoolean, v1.VariableTypeScalar},
		{"123", v1.DataTypeBoolean, v1.VariableTypeScalar},
		{"1.234", v1.DataTypeBoolean, v1.VariableTypeScalar},
		{"blah", v1.DataTypeBoolean, v1.VariableTypeScalar},
		{"[false]", v1.DataTypeBoolean, v1.VariableTypeScalar},
		{"{\"one\": false}", v1.DataTypeBoolean, v1.VariableTypeScalar},

		{"blah", v1.DataTypeFloat, v1.VariableTypeScalar},
		{"[123]", v1.DataTypeFloat, v1.VariableTypeScalar},
		{"[1.234]", v1.DataTypeFloat, v1.VariableTypeScalar},
		{"{\"one\": 1.234}", v1.DataTypeFloat, v1.VariableTypeScalar},

		// these tests should ostensibly continue with VariableTypeArray and VariableTypeMap
		// however when using JSON decoding, valid JSON decodes w/o error
		// even if it is decoded into the wrong type
		// thus we are not currently testing these items
		// TODO: test JSON strings that decode improperly
	}

	for _, tt := range badTests {
		t.Run(fmt.Sprintf("testing %s into %s/%s", tt.input, tt.dataType, tt.variableType), func(t *testing.T) {
			if err := validateValue(tt.input, tt.dataType, tt.variableType); err == nil {
				t.Errorf("%s improperly accepted as %s/%s", tt.input, tt.dataType, tt.variableType)
			}
		})
	}
}
