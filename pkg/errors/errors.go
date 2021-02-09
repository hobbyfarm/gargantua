package errors

type HobbyfarmError struct {
	Code int
	Message string
	Description string
}

func (h HobbyfarmError) Error() string {
	return h.Message
}

func NewAlreadyExists(msg string) HobbyfarmError {
	return HobbyfarmError{
		Code: 409,
		Message: msg,
		Description: "resource already exists",
	}
}

func IsAlreadyExists(err error) bool {
	he, ok := err.(HobbyfarmError)
	if !ok {
		return false
	}
	
	return he.Code == 409
}