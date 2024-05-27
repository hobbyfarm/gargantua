// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.21.12
// source: environment/environment.proto

package environmentpb

import (
	context "context"
	general "github.com/hobbyfarm/gargantua/v3/protos/general"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	EnvironmentSvc_CreateEnvironment_FullMethodName           = "/environment.EnvironmentSvc/CreateEnvironment"
	EnvironmentSvc_GetEnvironment_FullMethodName              = "/environment.EnvironmentSvc/GetEnvironment"
	EnvironmentSvc_UpdateEnvironment_FullMethodName           = "/environment.EnvironmentSvc/UpdateEnvironment"
	EnvironmentSvc_DeleteEnvironment_FullMethodName           = "/environment.EnvironmentSvc/DeleteEnvironment"
	EnvironmentSvc_DeleteCollectionEnvironment_FullMethodName = "/environment.EnvironmentSvc/DeleteCollectionEnvironment"
	EnvironmentSvc_ListEnvironment_FullMethodName             = "/environment.EnvironmentSvc/ListEnvironment"
)

// EnvironmentSvcClient is the client API for EnvironmentSvc service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type EnvironmentSvcClient interface {
	CreateEnvironment(ctx context.Context, in *CreateEnvironmentRequest, opts ...grpc.CallOption) (*general.ResourceId, error)
	GetEnvironment(ctx context.Context, in *general.GetRequest, opts ...grpc.CallOption) (*Environment, error)
	UpdateEnvironment(ctx context.Context, in *UpdateEnvironmentRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	DeleteEnvironment(ctx context.Context, in *general.ResourceId, opts ...grpc.CallOption) (*emptypb.Empty, error)
	DeleteCollectionEnvironment(ctx context.Context, in *general.ListOptions, opts ...grpc.CallOption) (*emptypb.Empty, error)
	ListEnvironment(ctx context.Context, in *general.ListOptions, opts ...grpc.CallOption) (*ListEnvironmentsResponse, error)
}

type environmentSvcClient struct {
	cc grpc.ClientConnInterface
}

func NewEnvironmentSvcClient(cc grpc.ClientConnInterface) EnvironmentSvcClient {
	return &environmentSvcClient{cc}
}

func (c *environmentSvcClient) CreateEnvironment(ctx context.Context, in *CreateEnvironmentRequest, opts ...grpc.CallOption) (*general.ResourceId, error) {
	out := new(general.ResourceId)
	err := c.cc.Invoke(ctx, EnvironmentSvc_CreateEnvironment_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *environmentSvcClient) GetEnvironment(ctx context.Context, in *general.GetRequest, opts ...grpc.CallOption) (*Environment, error) {
	out := new(Environment)
	err := c.cc.Invoke(ctx, EnvironmentSvc_GetEnvironment_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *environmentSvcClient) UpdateEnvironment(ctx context.Context, in *UpdateEnvironmentRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, EnvironmentSvc_UpdateEnvironment_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *environmentSvcClient) DeleteEnvironment(ctx context.Context, in *general.ResourceId, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, EnvironmentSvc_DeleteEnvironment_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *environmentSvcClient) DeleteCollectionEnvironment(ctx context.Context, in *general.ListOptions, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, EnvironmentSvc_DeleteCollectionEnvironment_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *environmentSvcClient) ListEnvironment(ctx context.Context, in *general.ListOptions, opts ...grpc.CallOption) (*ListEnvironmentsResponse, error) {
	out := new(ListEnvironmentsResponse)
	err := c.cc.Invoke(ctx, EnvironmentSvc_ListEnvironment_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EnvironmentSvcServer is the server API for EnvironmentSvc service.
// All implementations must embed UnimplementedEnvironmentSvcServer
// for forward compatibility
type EnvironmentSvcServer interface {
	CreateEnvironment(context.Context, *CreateEnvironmentRequest) (*general.ResourceId, error)
	GetEnvironment(context.Context, *general.GetRequest) (*Environment, error)
	UpdateEnvironment(context.Context, *UpdateEnvironmentRequest) (*emptypb.Empty, error)
	DeleteEnvironment(context.Context, *general.ResourceId) (*emptypb.Empty, error)
	DeleteCollectionEnvironment(context.Context, *general.ListOptions) (*emptypb.Empty, error)
	ListEnvironment(context.Context, *general.ListOptions) (*ListEnvironmentsResponse, error)
	mustEmbedUnimplementedEnvironmentSvcServer()
}

// UnimplementedEnvironmentSvcServer must be embedded to have forward compatible implementations.
type UnimplementedEnvironmentSvcServer struct {
}

func (UnimplementedEnvironmentSvcServer) CreateEnvironment(context.Context, *CreateEnvironmentRequest) (*general.ResourceId, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateEnvironment not implemented")
}
func (UnimplementedEnvironmentSvcServer) GetEnvironment(context.Context, *general.GetRequest) (*Environment, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetEnvironment not implemented")
}
func (UnimplementedEnvironmentSvcServer) UpdateEnvironment(context.Context, *UpdateEnvironmentRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateEnvironment not implemented")
}
func (UnimplementedEnvironmentSvcServer) DeleteEnvironment(context.Context, *general.ResourceId) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteEnvironment not implemented")
}
func (UnimplementedEnvironmentSvcServer) DeleteCollectionEnvironment(context.Context, *general.ListOptions) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteCollectionEnvironment not implemented")
}
func (UnimplementedEnvironmentSvcServer) ListEnvironment(context.Context, *general.ListOptions) (*ListEnvironmentsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListEnvironment not implemented")
}
func (UnimplementedEnvironmentSvcServer) mustEmbedUnimplementedEnvironmentSvcServer() {}

// UnsafeEnvironmentSvcServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to EnvironmentSvcServer will
// result in compilation errors.
type UnsafeEnvironmentSvcServer interface {
	mustEmbedUnimplementedEnvironmentSvcServer()
}

func RegisterEnvironmentSvcServer(s grpc.ServiceRegistrar, srv EnvironmentSvcServer) {
	s.RegisterService(&EnvironmentSvc_ServiceDesc, srv)
}

func _EnvironmentSvc_CreateEnvironment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateEnvironmentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnvironmentSvcServer).CreateEnvironment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnvironmentSvc_CreateEnvironment_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnvironmentSvcServer).CreateEnvironment(ctx, req.(*CreateEnvironmentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnvironmentSvc_GetEnvironment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(general.GetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnvironmentSvcServer).GetEnvironment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnvironmentSvc_GetEnvironment_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnvironmentSvcServer).GetEnvironment(ctx, req.(*general.GetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnvironmentSvc_UpdateEnvironment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateEnvironmentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnvironmentSvcServer).UpdateEnvironment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnvironmentSvc_UpdateEnvironment_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnvironmentSvcServer).UpdateEnvironment(ctx, req.(*UpdateEnvironmentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnvironmentSvc_DeleteEnvironment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(general.ResourceId)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnvironmentSvcServer).DeleteEnvironment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnvironmentSvc_DeleteEnvironment_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnvironmentSvcServer).DeleteEnvironment(ctx, req.(*general.ResourceId))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnvironmentSvc_DeleteCollectionEnvironment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(general.ListOptions)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnvironmentSvcServer).DeleteCollectionEnvironment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnvironmentSvc_DeleteCollectionEnvironment_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnvironmentSvcServer).DeleteCollectionEnvironment(ctx, req.(*general.ListOptions))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnvironmentSvc_ListEnvironment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(general.ListOptions)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnvironmentSvcServer).ListEnvironment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnvironmentSvc_ListEnvironment_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnvironmentSvcServer).ListEnvironment(ctx, req.(*general.ListOptions))
	}
	return interceptor(ctx, in, info, handler)
}

// EnvironmentSvc_ServiceDesc is the grpc.ServiceDesc for EnvironmentSvc service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var EnvironmentSvc_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "environment.EnvironmentSvc",
	HandlerType: (*EnvironmentSvcServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateEnvironment",
			Handler:    _EnvironmentSvc_CreateEnvironment_Handler,
		},
		{
			MethodName: "GetEnvironment",
			Handler:    _EnvironmentSvc_GetEnvironment_Handler,
		},
		{
			MethodName: "UpdateEnvironment",
			Handler:    _EnvironmentSvc_UpdateEnvironment_Handler,
		},
		{
			MethodName: "DeleteEnvironment",
			Handler:    _EnvironmentSvc_DeleteEnvironment_Handler,
		},
		{
			MethodName: "DeleteCollectionEnvironment",
			Handler:    _EnvironmentSvc_DeleteCollectionEnvironment_Handler,
		},
		{
			MethodName: "ListEnvironment",
			Handler:    _EnvironmentSvc_ListEnvironment_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "environment/environment.proto",
}
