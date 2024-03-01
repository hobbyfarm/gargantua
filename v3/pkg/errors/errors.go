package errors

import (
	"github.com/golang/protobuf/proto"
	"github.com/hobbyfarm/gargantua/v3/protos/general"
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

func GrpcNotSpecifiedError[T proto.Message](protoMessage T, propName string) error {
	return GrpcError[proto.Message](codes.InvalidArgument, "missing %s", protoMessage, propName)
}

func GrpcIdNotSpecifiedError[T proto.Message](protoMessage T) error {
	return GrpcError[proto.Message](codes.InvalidArgument, "no id specified", protoMessage)
}

func GrpcNotFoundError(resourceId *general.GetRequest, resourceName string) error {
	id := resourceId.GetId()
	return GrpcError[*general.GetRequest](codes.NotFound, "could not find %s for id %s", resourceId, resourceName, id)
}

func IsGrpcNotFound(err error) bool {
	return status.Code(err) == codes.NotFound
}

func GrpcGetError(req *general.GetRequest, resourceName string, err error) error {
	return GrpcError[*general.GetRequest](
		codes.Internal,
		"error while retreiving %s by id %s with error: %v",
		req,
		resourceName,
		req.GetId(),
		err,
	)
}

func GrpcListError(listOptions *general.ListOptions, resourceName string) error {
	return GrpcError[*general.ListOptions](codes.Internal, "error retreiving %s", listOptions, resourceName)
}

func GrpcCacheError[T proto.Message](protoMessage T, resourceName string) error {
	return GrpcError[proto.Message](codes.Unavailable, "error while retreiving %s: cache is not properly synced yet", protoMessage, resourceName)
}

func GrpcParsingError[T proto.Message](protoMessage T, propName string) error {
	return GrpcError[proto.Message](codes.Internal, "error parsing %s", protoMessage, propName)
}

func GetErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	st, ok := status.FromError(err)
	if !ok {
		// not a gRPC error
		return err.Error()
	}
	return st.Message()
}
