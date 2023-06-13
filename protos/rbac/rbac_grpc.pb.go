// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.19.1
// source: rbac/rbac.proto

package rbac

import (
	context "context"
	authr "github.com/hobbyfarm/gargantua/protos/authr"
	user "github.com/hobbyfarm/gargantua/protos/user"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// RbacSvcClient is the client API for RbacSvc service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RbacSvcClient interface {
	Grants(ctx context.Context, in *GrantRequest, opts ...grpc.CallOption) (*authr.AuthRResponse, error)
	GetAccessSet(ctx context.Context, in *user.UserId, opts ...grpc.CallOption) (*AccessSet, error)
	GetHobbyfarmRoleBindings(ctx context.Context, in *user.UserId, opts ...grpc.CallOption) (*RoleBindings, error)
}

type rbacSvcClient struct {
	cc grpc.ClientConnInterface
}

func NewRbacSvcClient(cc grpc.ClientConnInterface) RbacSvcClient {
	return &rbacSvcClient{cc}
}

func (c *rbacSvcClient) Grants(ctx context.Context, in *GrantRequest, opts ...grpc.CallOption) (*authr.AuthRResponse, error) {
	out := new(authr.AuthRResponse)
	err := c.cc.Invoke(ctx, "/rbac.RbacSvc/Grants", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rbacSvcClient) GetAccessSet(ctx context.Context, in *user.UserId, opts ...grpc.CallOption) (*AccessSet, error) {
	out := new(AccessSet)
	err := c.cc.Invoke(ctx, "/rbac.RbacSvc/GetAccessSet", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rbacSvcClient) GetHobbyfarmRoleBindings(ctx context.Context, in *user.UserId, opts ...grpc.CallOption) (*RoleBindings, error) {
	out := new(RoleBindings)
	err := c.cc.Invoke(ctx, "/rbac.RbacSvc/GetHobbyfarmRoleBindings", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RbacSvcServer is the server API for RbacSvc service.
// All implementations should embed UnimplementedRbacSvcServer
// for forward compatibility
type RbacSvcServer interface {
	Grants(context.Context, *GrantRequest) (*authr.AuthRResponse, error)
	GetAccessSet(context.Context, *user.UserId) (*AccessSet, error)
	GetHobbyfarmRoleBindings(context.Context, *user.UserId) (*RoleBindings, error)
}

// UnimplementedRbacSvcServer should be embedded to have forward compatible implementations.
type UnimplementedRbacSvcServer struct {
}

func (UnimplementedRbacSvcServer) Grants(context.Context, *GrantRequest) (*authr.AuthRResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Grants not implemented")
}
func (UnimplementedRbacSvcServer) GetAccessSet(context.Context, *user.UserId) (*AccessSet, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAccessSet not implemented")
}
func (UnimplementedRbacSvcServer) GetHobbyfarmRoleBindings(context.Context, *user.UserId) (*RoleBindings, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetHobbyfarmRoleBindings not implemented")
}

// UnsafeRbacSvcServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RbacSvcServer will
// result in compilation errors.
type UnsafeRbacSvcServer interface {
	mustEmbedUnimplementedRbacSvcServer()
}

func RegisterRbacSvcServer(s grpc.ServiceRegistrar, srv RbacSvcServer) {
	s.RegisterService(&RbacSvc_ServiceDesc, srv)
}

func _RbacSvc_Grants_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GrantRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RbacSvcServer).Grants(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rbac.RbacSvc/Grants",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RbacSvcServer).Grants(ctx, req.(*GrantRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RbacSvc_GetAccessSet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(user.UserId)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RbacSvcServer).GetAccessSet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rbac.RbacSvc/GetAccessSet",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RbacSvcServer).GetAccessSet(ctx, req.(*user.UserId))
	}
	return interceptor(ctx, in, info, handler)
}

func _RbacSvc_GetHobbyfarmRoleBindings_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(user.UserId)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RbacSvcServer).GetHobbyfarmRoleBindings(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rbac.RbacSvc/GetHobbyfarmRoleBindings",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RbacSvcServer).GetHobbyfarmRoleBindings(ctx, req.(*user.UserId))
	}
	return interceptor(ctx, in, info, handler)
}

// RbacSvc_ServiceDesc is the grpc.ServiceDesc for RbacSvc service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RbacSvc_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "rbac.RbacSvc",
	HandlerType: (*RbacSvcServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Grants",
			Handler:    _RbacSvc_Grants_Handler,
		},
		{
			MethodName: "GetAccessSet",
			Handler:    _RbacSvc_GetAccessSet_Handler,
		},
		{
			MethodName: "GetHobbyfarmRoleBindings",
			Handler:    _RbacSvc_GetHobbyfarmRoleBindings_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "rbac/rbac.proto",
}
