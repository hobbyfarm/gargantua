package setting

import (
	"context"
	"encoding/json"
	"fmt"
	v12 "github.com/hobbyfarm/gargantua/pkg/apis/hobbyfarm.io/v1"
	"github.com/hobbyfarm/gargantua/pkg/webhook/validation"
	"github.com/pkg/errors"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"
)

var typeMap map[v12.VariableType]map[v12.DataType]any

func init() {
	typeMap = map[v12.VariableType]map[v12.DataType]any{
		v12.VariableTypeMap: {
			v12.DataTypeString:  map[string]string{},
			v12.DataTypeBoolean: map[string]bool{},
			v12.DataTypeFloat:   map[string]float32{},
			v12.DataTypeInteger: map[string]int{},
		},
		v12.VariableTypeArray: {
			v12.DataTypeInteger: []int{},
			v12.DataTypeFloat:   []float32{},
			v12.DataTypeBoolean: []bool{},
			v12.DataTypeString:  []string{},
		},
	}

	setting := &v12.Setting{}
	gvk := setting.GroupVersionKind()

	validation.RegisterAdmitters(gvk, V1Review, V1beta1Review)
}

func V1beta1Review(ctx context.Context, ar *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
	resp := V1Review(ctx, validation.ConvertAdmissionRequestToV1(ar))
	return validation.ConvertAdmissionResponseToV1beta1(resp)
}

func V1Review(_ context.Context, ar *v1.AdmissionRequest) *v1.AdmissionResponse {
	resp := &v1.AdmissionResponse{}

	oldObj, ok := ar.OldObject.Object.(*v12.Setting)
	if !ok {
		return respDenied("could not cast old object into hobbyfarm.io/setting")
	}

	newObj, ok := ar.Object.Object.(*v12.Setting)
	if !ok {
		return respDenied("could not cast new object into hobbyfarm.io/setting")
	}

	if oldObj.DataType != newObj.DataType {
		return respDenied("datatype field immutable")
	}

	if oldObj.VariableType != newObj.VariableType {
		return respDenied(" variabletype field immutable")
	}

	// make sure we get what we want
	if err := validateValue(newObj.Value, newObj.DataType, newObj.VariableType); err != nil {
		return respDenied("cannot convert %s into %s/%s: %s", newObj.Value, newObj.VariableType, newObj.DataType, err.Error())
	}

	// if its an enum, check that all enum values are of the data type
	if err := validateEnum(newObj.Enum, newObj.DataType); err != nil {
		return respDenied("enum contains value that does not convert into %s: %s", newObj.DataType, err.Error())
	}

	if !validateValueEnum(newObj.Value, newObj.Enum) {
		return respDenied("value is not contained within enum list")
	}

	resp.Allowed = true
	return resp
}

func respDenied(msg string, fields ...interface{}) *v1.AdmissionResponse {
	return &v1.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Message: fmt.Sprintf(msg, fields...),
		},
	}
}

func validateValue(in string, dataType v12.DataType, variableType v12.VariableType) error {
	if variableType == v12.VariableTypeEnum {
		variableType = v12.VariableTypeScalar
	}

	if variableType == v12.VariableTypeScalar {
		var err error
		switch dataType {
		case v12.DataTypeBoolean:
			_, err = strconv.ParseBool(in)
		case v12.DataTypeInteger:
			_, err = strconv.Atoi(in)
		case v12.DataTypeFloat:
			_, err = strconv.ParseFloat(in, 32)
		}

		return err
	}

	out := typeMap[variableType][dataType]

	decoder := json.NewDecoder(strings.NewReader(in))
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&out)

	if err != nil {
		return errors.Wrapf(err, "error validating %s (%s/%s)", in, variableType, dataType)
	}

	return nil
}

func validateEnum(enums []string, dataType v12.DataType) error {
	for _, v := range enums {
		switch dataType {
		case v12.DataTypeBoolean:
			if _, err := strconv.ParseBool(v); err != nil {
				return err
			}
		case v12.DataTypeFloat:
			if _, err := strconv.ParseFloat(v, 32); err != nil {
				return err
			}
		case v12.DataTypeInteger:
			if _, err := strconv.ParseInt(v, 10, 32); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateValueEnum(val string, enum []string) bool {
	var ok = false
	for _, v := range enum {
		if val == v {
			ok = true
		}
	}

	return ok
}
