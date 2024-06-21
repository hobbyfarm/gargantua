package setting

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hobbyfarm/gargantua/v3/pkg/property"

	settingpb "github.com/hobbyfarm/gargantua/v3/protos/setting"
)

type SettingName string

const (
	SettingRegistrationDisabled              SettingName = "registration-disabled"
	SettingAdminUIMOTD                       SettingName = "motd-admin-ui"
	SettingUIMOTD                            SettingName = "motd-ui"
	ScheduledEventRetentionTime              SettingName = "scheduledevent-retention-time"
	StrictAccessCodeValidation               SettingName = "strict-accesscode-validation"
	SettingRegistrationPrivacyPolicyRequired SettingName = "registration-privacy-policy-required"
	SettingRegistrationPrivacyPolicyLink     SettingName = "registration-privacy-policy-link"
	SettingRegistrationPrivacyPolicyLinkName SettingName = "registration-privacy-policy-linkname"
	ImprintLink                              SettingName = "imprint-link"
	ImprintLinkName                          SettingName = "imprint-linkname"
	AboutModalButtons                        SettingName = "aboutmodal-buttons"
	UserTokenExpiration                      SettingName = "user-token-expiration"
)

var DataTypeMappingToProto = map[property.DataType]settingpb.DataType{
	property.DataTypeFloat:   settingpb.DataType_DATA_TYPE_FLOAT,
	property.DataTypeInteger: settingpb.DataType_DATA_TYPE_INTEGER,
	property.DataTypeString:  settingpb.DataType_DATA_TYPE_STRING,
	property.DataTypeBoolean: settingpb.DataType_DATA_TYPE_BOOLEAN,
}

var ValueTypeMappingToProto = map[property.ValueType]settingpb.ValueType{
	property.ValueTypeArray:  settingpb.ValueType_VALUE_TYPE_ARRAY,
	property.ValueTypeMap:    settingpb.ValueType_VALUE_TYPE_MAP,
	property.ValueTypeScalar: settingpb.ValueType_VALUE_TYPE_SCALAR,
}

var DataTypeMappingToHfTypes = map[settingpb.DataType]property.DataType{
	settingpb.DataType_DATA_TYPE_FLOAT:   property.DataTypeFloat,
	settingpb.DataType_DATA_TYPE_INTEGER: property.DataTypeInteger,
	settingpb.DataType_DATA_TYPE_STRING:  property.DataTypeString,
	settingpb.DataType_DATA_TYPE_BOOLEAN: property.DataTypeBoolean,
}

var ValueTypeMappingToHfTypes = map[settingpb.ValueType]property.ValueType{
	settingpb.ValueType_VALUE_TYPE_ARRAY:  property.ValueTypeArray,
	settingpb.ValueType_VALUE_TYPE_MAP:    property.ValueTypeMap,
	settingpb.ValueType_VALUE_TYPE_SCALAR: property.ValueTypeScalar,
}

func FromJSON(p *settingpb.Property, value string) (any, error) {
	if p.ValueType == settingpb.ValueType_VALUE_TYPE_SCALAR {
		switch p.DataType {
		case settingpb.DataType_DATA_TYPE_FLOAT:
			return strconv.ParseFloat(value, 64)
		case settingpb.DataType_DATA_TYPE_INTEGER:
			return strconv.Atoi(value)
		case settingpb.DataType_DATA_TYPE_BOOLEAN:
			return strconv.ParseBool(value)
		case settingpb.DataType_DATA_TYPE_STRING:
			fallthrough
		default:
			return value, nil
		}
	}

	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.DisallowUnknownFields()

	if p.ValueType == settingpb.ValueType_VALUE_TYPE_MAP {
		switch p.DataType {
		case settingpb.DataType_DATA_TYPE_FLOAT:
			var out = map[string]float64{}
			return out, decoder.Decode(&out)
		case settingpb.DataType_DATA_TYPE_INTEGER:
			var out = map[string]int{}
			return out, decoder.Decode(&out)
		case settingpb.DataType_DATA_TYPE_STRING:
			var out = map[string]string{}
			return out, decoder.Decode(&out)
		case settingpb.DataType_DATA_TYPE_BOOLEAN:
			var out = map[string]bool{}
			return out, decoder.Decode(&out)
		}
	}

	if p.ValueType == settingpb.ValueType_VALUE_TYPE_ARRAY {
		switch p.DataType {
		case settingpb.DataType_DATA_TYPE_FLOAT:
			var out = []float64{}
			return out, decoder.Decode(&out)
		case settingpb.DataType_DATA_TYPE_INTEGER:
			var out = []int{}
			return out, decoder.Decode(&out)
		case settingpb.DataType_DATA_TYPE_STRING:
			var out = []string{}
			return out, decoder.Decode(&out)
		case settingpb.DataType_DATA_TYPE_BOOLEAN:
			var out = []bool{}
			return out, decoder.Decode(&out)
		}
	}

	return nil, fmt.Errorf("no match for datatype %s and valuetype %s", p.DataType, p.ValueType)
}
