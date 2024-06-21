package errors

import (
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IdGetterProtoMessage is an interface that combines the GetId() function and the proto.Message interface.
// general.GetRequest and general.ResourceId do implement this interface
type IdGetterProtoMessage interface {
	proto.Message
	GetId() string
}

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

func GrpcBadRequestError[T proto.Message](protoMessage T, propName string, propValue string) error {
	return GrpcError[proto.Message](codes.InvalidArgument, "invalid value \"%s\" for property %s", protoMessage, propValue, propName)
}

func GrpcNotSpecifiedError[T proto.Message](protoMessage T, propName string) error {
	return GrpcError[proto.Message](codes.InvalidArgument, "missing %s", protoMessage, propName)
}

func GrpcIdNotSpecifiedError[T proto.Message](protoMessage T) error {
	return GrpcError[proto.Message](codes.InvalidArgument, "no id specified", protoMessage)
}

func GrpcNotFoundError[T IdGetterProtoMessage](resourceId T, resourceName string) error {
	id := resourceId.GetId()
	return GrpcError[T](codes.NotFound, "could not find %s for id %s", resourceId, resourceName, id)
}

func IsGrpcNotFound(err error) bool {
	return status.Code(err) == codes.NotFound
}
func IsGrpcParsingError(err error) bool {
	statusErr := status.Convert(err)
	return statusErr.Code() == codes.Internal && strings.HasPrefix(statusErr.Message(), "error parsing")
}

func GrpcGetError(req *generalpb.GetRequest, resourceName string, err error) error {
	return GrpcError[*generalpb.GetRequest](
		codes.Internal,
		"error while retreiving %s by id %s with error: %v",
		req,
		resourceName,
		req.GetId(),
		err,
	)
}

func GrpcListError(listOptions *generalpb.ListOptions, resourceName string) error {
	return GrpcError[*generalpb.ListOptions](codes.Internal, "error retreiving %s", listOptions, resourceName)
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

// Generic function to handle different types of details
func ExtractDetail[T proto.Message](s *status.Status) (T, error) {
	var zeroValue T

	if len(s.Details()) > 0 {
		for _, detail := range s.Details() {
			if details, ok := detail.(T); ok {
				return details, nil
			}
		}
	}
	return zeroValue, fmt.Errorf("no details of the expected type found in the error status")
}
