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
