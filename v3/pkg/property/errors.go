package property

import "fmt"

type ValidationError struct {
	message string
}

func (v ValidationError) Error() string {
	return v.message
}

func NewValidationErrorf(msg string, value ...any) ValidationError {
	if msg == "" {
		msg = "invalid value"
	}

	return ValidationError{
		message: fmt.Sprintf(msg, value...),
	}
}

type TypeConversionError struct {
	message string
}

func (t TypeConversionError) Error() string {
	return t.message
}

func NewTypeConversionErrorf(v string, t string) TypeConversionError {
	return TypeConversionError{
		message: fmt.Sprintf("invalid value, cannot convert %s to %s", v, t),
	}
}
