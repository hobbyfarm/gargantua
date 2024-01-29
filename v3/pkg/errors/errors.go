package errors

import (
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type HobbyfarmError struct {
	Code        int
	Message     string
	Description string
}

func (h HobbyfarmError) Error() string {
	return h.Message
}

func NewAlreadyExists(msg string) HobbyfarmError {
	return HobbyfarmError{
		Code:        409,
		Message:     msg,
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

// generic function which returns a standard gRPC status error
// @c: The gRPC error code which specifies the kind of our error
// @format: The formatted error message to return
// @details: A proto message
// @a: The arguments for the formatted error message
func GrpcError[T proto.Message](c codes.Code, format string, details T, a ...any) error {
	err := status.Newf(
		c,
		format,
		a...,
	)
	err, wde := err.WithDetails(details)
	if wde != nil {
		return wde
	}
	return err.Err()
}
