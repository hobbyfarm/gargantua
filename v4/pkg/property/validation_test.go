package property

import (
	"fmt"
	"testing"
)

var (
	testValues = map[bool]map[ValueType]map[DataType]string{
		true: {
			ValueTypeArray: {
				DataTypeBoolean: "[true, false]",
				DataTypeInteger: "[1,2,3,4]",
				DataTypeFloat:   "[1.234, 1.432]",
				DataTypeString:  "[\"one\", \"two\"]",
			},
			ValueTypeMap: {
				DataTypeBoolean: "{\"one\": true, \"two\": false}",
				DataTypeInteger: "{\"one\": 1, \"two\": 2}",
				DataTypeFloat:   "{\"one\": 1.234, \"two\": 1.432}",
				DataTypeString:  "{\"one\": \"one\", \"two\": \"two\"}",
			},
		},
		false: {
			ValueTypeArray: {
				DataTypeBoolean: "[true, true]",
				DataTypeInteger: "[1, 1, 1]",
				DataTypeFloat:   "[1.234, 1.234]",
				DataTypeString:  "[\"one\", \"one\"]",
			},
			ValueTypeMap: {
				DataTypeBoolean: "{\"one\": true, \"two\": true}",
				DataTypeInteger: "{\"one\": 1, \"two\": 1}",
				DataTypeFloat:   "{\"one\": 1.234, \"two\": 1.234}",
				DataTypeString:  "{\"one\": \"one\", \"two\": \"one\"}",
			},
		},
	}
)

func TestValidate_UniqueSlices(t *testing.T) {
	for unique, valueTypes := range testValues {
		for valueType, dataTypes := range valueTypes {
			for dataType, testString := range dataTypes {
				var name = fmt.Sprintf("%s of %s", valueType, dataType)
				if unique {
					name = "unique " + name + " should pass validation"
				} else {
					name = "non-unique " + name + " should not pass validation"
				}

				t.Run(name, func(t *testing.T) {
					var p = Property{
						DataType:  dataType,
						ValueType: valueType,
						SettingValidation: SettingValidation{
							UniqueItems: true,
						},
					}

					err := p.Validate(testString)

					// unique and no error is good
					// unique and error is bad
					if err != nil && unique {
						t.Error(err)
					}

					// not unique and error is good
					// not unique and no error is bad
					if err == nil && !unique {
						t.Error(err)
					}
				})
			}
		}
	}
}
