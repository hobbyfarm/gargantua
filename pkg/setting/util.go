package setting

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hobbyfarm/gargantua/v3/pkg/property"
	settingProto "github.com/hobbyfarm/gargantua/v3/protos/setting"
)

type SettingName string

const (
	SettingRegistrationDisabled              SettingName = "registration-disabled"
	SettingAdminUIMOTD                       SettingName = "motd-admin-ui"
	SettingUIMOTD                            SettingName = "motd-ui"
	ScheduledEventRetentionTime              SettingName = "scheduledevent-retention-time"
	SettingRegistrationPrivacyPolicyRequired SettingName = "registration-privacy-policy-required"
	SettingRegistrationPrivacyPolicyLink     SettingName = "registration-privacy-policy-link"
	SettingRegistrationPrivacyPolicyLinkName SettingName = "registration-privacy-policy-linkname"
	ImprintLink                              SettingName = "imprint-link"
	ImprintLinkName                          SettingName = "imprint-linkname"
	AboutModalButtons                        SettingName = "aboutmodal-buttons"
)

var DataTypeMappingToProto = map[property.DataType]settingProto.DataType{
	property.DataTypeFloat:   settingProto.DataType_DATA_TYPE_FLOAT,
	property.DataTypeInteger: settingProto.DataType_DATA_TYPE_INTEGER,
	property.DataTypeString:  settingProto.DataType_DATA_TYPE_STRING,
	property.DataTypeBoolean: settingProto.DataType_DATA_TYPE_BOOLEAN,
}

var ValueTypeMappingToProto = map[property.ValueType]settingProto.ValueType{
	property.ValueTypeArray:  settingProto.ValueType_VALUE_TYPE_ARRAY,
	property.ValueTypeMap:    settingProto.ValueType_VALUE_TYPE_MAP,
	property.ValueTypeScalar: settingProto.ValueType_VALUE_TYPE_SCALAR,
}

var DataTypeMappingToHfTypes = map[settingProto.DataType]property.DataType{
	settingProto.DataType_DATA_TYPE_FLOAT:   property.DataTypeFloat,
	settingProto.DataType_DATA_TYPE_INTEGER: property.DataTypeInteger,
	settingProto.DataType_DATA_TYPE_STRING:  property.DataTypeString,
	settingProto.DataType_DATA_TYPE_BOOLEAN: property.DataTypeBoolean,
}

var ValueTypeMappingToHfTypes = map[settingProto.ValueType]property.ValueType{
	settingProto.ValueType_VALUE_TYPE_ARRAY:  property.ValueTypeArray,
	settingProto.ValueType_VALUE_TYPE_MAP:    property.ValueTypeMap,
	settingProto.ValueType_VALUE_TYPE_SCALAR: property.ValueTypeScalar,
}

func FromJSON(p *settingProto.Property, value string) (any, error) {
	if p.ValueType == settingProto.ValueType_VALUE_TYPE_SCALAR {
		switch p.DataType {
		case settingProto.DataType_DATA_TYPE_FLOAT:
			return strconv.ParseFloat(value, 64)
		case settingProto.DataType_DATA_TYPE_INTEGER:
			return strconv.Atoi(value)
		case settingProto.DataType_DATA_TYPE_BOOLEAN:
			return strconv.ParseBool(value)
		case settingProto.DataType_DATA_TYPE_STRING:
			fallthrough
		default:
			return value, nil
		}
	}

	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.DisallowUnknownFields()

	if p.ValueType == settingProto.ValueType_VALUE_TYPE_MAP {
		switch p.DataType {
		case settingProto.DataType_DATA_TYPE_FLOAT:
			var out = map[string]float64{}
			return out, decoder.Decode(&out)
		case settingProto.DataType_DATA_TYPE_INTEGER:
			var out = map[string]int{}
			return out, decoder.Decode(&out)
		case settingProto.DataType_DATA_TYPE_STRING:
			var out = map[string]string{}
			return out, decoder.Decode(&out)
		case settingProto.DataType_DATA_TYPE_BOOLEAN:
			var out = map[string]bool{}
			return out, decoder.Decode(&out)
		}
	}

	if p.ValueType == settingProto.ValueType_VALUE_TYPE_ARRAY {
		switch p.DataType {
		case settingProto.DataType_DATA_TYPE_FLOAT:
			var out = []float64{}
			return out, decoder.Decode(&out)
		case settingProto.DataType_DATA_TYPE_INTEGER:
			var out = []int{}
			return out, decoder.Decode(&out)
		case settingProto.DataType_DATA_TYPE_STRING:
			var out = []string{}
			return out, decoder.Decode(&out)
		case settingProto.DataType_DATA_TYPE_BOOLEAN:
			var out = []bool{}
			return out, decoder.Decode(&out)
		}
	}

	return nil, fmt.Errorf("no match for datatype %s and valuetype %s", p.DataType, p.ValueType)
}
