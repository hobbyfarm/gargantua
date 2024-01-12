package property

import (
	"github.com/peterhellberg/duration"
	"github.com/xeipuuv/gojsonschema"
	"math"
	"reflect"
	"regexp"
	"strconv"
)

type validator func(value any) error

type Iterable interface {
	Items() []any
}

func (p Property) getValidators() []validator {
	return []validator{
		p.validateRequired,
		p.validateMin,
		p.validateMinLength,
		p.validateMax,
		p.validateMaxLength,
		p.validatePattern,
		p.validateFormat,
		p.validateEnum,
		p.validateUniqueItems,
	}
}

// Validate executes validation of value given rules and definitions in Property
func (p Property) Validate(value string) error {
	// return error value if validation fails
	// else return nil
	var val any
	val, err := p.FromJSON(value)
	if err != nil {
		return err
	}

	for _, f := range p.getValidators() {
		if err := f(val); err != nil {
			return err
		}
	}

	return nil
}

func (p Property) validateRequired(value any) error {
	if value == nil {
		return NewValidationErrorf("value required, received %v", value)
	}

	return nil
}

func (p Property) validateMin(value any) error {
	var min float64

	if p.Minimum == nil {
		return nil
	}

	min = *p.Minimum
	switch v := value.(type) {
	case float64:
		if v < min {
			return NewValidationErrorf("value %f lower than minimum %f", v, min)
		}
	case float32:
		if v < float32(min) {
			return NewValidationErrorf("value %f lower than minimum %f", v, float32(min))
		}
	case int:
		if v < int(min) {
			return NewValidationErrorf("value %d lower than minimum %d", v, int(min))
		}
	case int64:
		if v < int64(min) {
			return NewValidationErrorf("value %d lower than minimum %d", v, int64(min))
		}
	case int32:
		if v < int32(min) {
			return NewValidationErrorf("value %d lower than minimum %d", v, int32(min))
		}
	default:
		return nil
	}

	return nil
}

func (p Property) validateMax(value any) error {
	var max float64
	if p.Maximum == nil {
		max = math.MaxFloat64
	} else {
		max = *p.Maximum
	}

	switch v := value.(type) {
	case float64:
		if v > max {
			return NewValidationErrorf("value %f higher than maximum %f", v, max)
		}
	case float32:
		if v > float32(max) {
			return NewValidationErrorf("value %f higher than maximum %f", v, float32(max))
		}
	case int64:
		if v > int64(max) {
			return NewValidationErrorf("value %d higher than maximum %d", v, int64(max))
		}
	case int32:
		if v > int32(max) {
			return NewValidationErrorf("value %d higher than maximum %d", v, int32(max))
		}
	default:
		return nil
	}

	return nil
}

func (p Property) validateMinLength(value any) error {
	var min int
	if p.MinLength == nil {
		return nil
	}

	min = int(*p.MinLength)

	k := reflect.TypeOf(value).Kind()
	if k != reflect.String && k != reflect.Slice && k != reflect.Array && k != reflect.Map {
		return NewValidationErrorf("minimum length validation not supported for kind %s", k)
	}

	vLen := reflect.ValueOf(value).Len()
	if vLen < min {
		return NewValidationErrorf("value length %d lower than minimum length %d", vLen, min)
	}

	return nil
}

func (p Property) validateMaxLength(value any) error {
	var max int
	if p.MaxLength == nil {
		return nil
	}

	max = int(*p.MaxLength)

	k := reflect.TypeOf(value).Kind()
	if k != reflect.String && k != reflect.Slice && k != reflect.Array && k != reflect.Map {
		return NewValidationErrorf("minimum length validation not supported for kind %s", k)
	}

	vLen := reflect.ValueOf(value).Len()
	if vLen > max {
		return NewValidationErrorf("value length %d higher than maximum length %d", vLen, max)
	}

	return nil
}

func (p Property) validatePattern(value any) error {
	if p.Pattern == nil {
		return nil
	}

	r, err := regexp.Compile(*p.Pattern)
	if err != nil {
		return err
	}

	switch v := value.(type) {
	case string:
		if !r.Match([]byte(v)) {
			return NewValidationErrorf("value %s does not match pattern %s", value, *p.Pattern)
		}
	case []string:
		for index, sliceVal := range v {
			if !r.Match([]byte(sliceVal)) {
				return NewValidationErrorf("value %s at index %d does not match pattern %s", sliceVal, index, *p.Pattern)
			}
		}
	case map[string]string:
		for key, mapVal := range v {
			if !r.Match([]byte(mapVal)) {
				return NewValidationErrorf("value %s at key %s does not match pattern %s", mapVal, key, *p.Pattern)
			}
		}
	default:
		return NewValidationErrorf("pattern validation not supported for kind %s", reflect.TypeOf(value).Kind())
	}

	return nil
}

type durationChecker struct{}

func (checker durationChecker) IsFormat(value any) bool {
	v, ok := value.(string)
	if !ok {
		return true
	}

	_, err := duration.Parse(v)
	if err != nil {
		return false
	}

	return true
}

func (p Property) validateFormat(value any) error {
	if p.Format == nil {
		return nil
	}

	switch v := value.(type) {
	case string:
		return validateFormat(*p.Format, v)
	case []string:
		for _, vv := range v {
			if err := validateFormat(*p.Format, vv); err != nil {
				return err
			}
		}
	case map[string]string:
		for _, vv := range v {
			if err := validateFormat(*p.Format, vv); err != nil {
				return err
			}
		}
	default:
		return NewValidationErrorf("format validation not supported for kind %s", reflect.TypeOf(value).Kind())
	}

	return nil
}

func validateFormat(format string, value string) error {
	fc := gojsonschema.FormatCheckers

	fc.Add("duration", durationChecker{})

	if !gojsonschema.FormatCheckers.IsFormat(format, value) {
		return NewValidationErrorf("value %s does not match format %s", value, format)
	}

	return nil
}

func (p Property) validateEnum(value any) error {
	if len(p.Enum) == 0 {
		return nil
	}

	switch v := value.(type) {
	case string:
		return validateStringEnum(v, p.Enum)
	case []string:
		for _, vv := range v {
			if err := validateStringEnum(vv, p.Enum); err != nil {
				return err
			}
		}
	case map[string]string:
		for _, vv := range v {
			if err := validateStringEnum(vv, p.Enum); err != nil {
				return err
			}
		}
	case int:
		return validateIntEnum(v, p.Enum)
	case []int:
		for _, vv := range v {
			if err := validateIntEnum(vv, p.Enum); err != nil {
				return err
			}
		}
	case map[string]int:
		for _, vv := range v {
			if err := validateIntEnum(vv, p.Enum); err != nil {
				return err
			}
		}
	case float64:
		return validateFloatEnum(v, p.Enum)
	case []float64:
		for _, vv := range v {
			if err := validateFloatEnum(vv, p.Enum); err != nil {
				return err
			}
		}
	case map[string]float64:
		for _, vv := range v {
			if err := validateFloatEnum(vv, p.Enum); err != nil {
				return err
			}
		}
	default:
		return NewValidationErrorf("enum validation not defined for kind %s", reflect.TypeOf(value).Kind())
	}

	return nil
}

func validateStringEnum(value string, enum []string) error {
	for _, e := range enum {
		if value == e {
			return nil
		}
	}

	return NewValidationErrorf("value %s not found in enum", value)
}

func validateIntEnum(value int, enum []string) error {
	var ok = false
	for _, e := range enum {
		i, err := strconv.Atoi(e)
		if err != nil {
			return err
		}
		if value == i {
			ok = true
		}
	}

	if !ok {
		return NewValidationErrorf("value %d not found in enum", value)
	}

	return nil
}

func validateFloatEnum(value float64, enum []string) error {
	var ok = false
	for _, e := range enum {
		f, err := strconv.ParseFloat(e, 64)
		if err != nil {
			return err
		}

		if value == f {
			ok = true
		}
	}

	if !ok {
		return NewValidationErrorf("value %f not found in enum", value)
	}

	return nil
}

func (p Property) validateUniqueItems(value any) error {
	if p.UniqueItems {
		switch v := value.(type) {
		case []string:
			return validateUniqueItemsSlice(v)
		case []int:
			return validateUniqueItemsSlice(v)
		case []float64:
			return validateUniqueItemsSlice(v)
		case map[string]string:
			return validateUniqueItemsMap(v)
		case map[string]int:
			return validateUniqueItemsMap(v)
		case map[string]float64:
			return validateUniqueItemsMap(v)
		}
	}

	return nil
}

func validateUniqueItemsSlice[GenericType any](items []GenericType) error {
	var unique = map[any]bool{}

	for _, i := range items {
		if unique[i] {
			return NewValidationErrorf("value %v not unique", i)
		}

		unique[i] = true
	}

	return nil
}

func validateUniqueItemsMap[GenericType any](items map[string]GenericType) error {
	var unique = map[any]bool{}

	for _, i := range items {
		if unique[i] {
			return NewValidationErrorf("value %v not unique", i)
		}

		unique[i] = true
	}

	return nil
}
