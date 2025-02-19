// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: vmclaim/virtualmachineclaim.proto

package vmclaimpb

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
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	VMClaimSvc_CreateVMClaim_FullMethodName           = "/vmclaim.VMClaimSvc/CreateVMClaim"
	VMClaimSvc_GetVMClaim_FullMethodName              = "/vmclaim.VMClaimSvc/GetVMClaim"
	VMClaimSvc_UpdateVMClaim_FullMethodName           = "/vmclaim.VMClaimSvc/UpdateVMClaim"
	VMClaimSvc_UpdateVMClaimStatus_FullMethodName     = "/vmclaim.VMClaimSvc/UpdateVMClaimStatus"
	VMClaimSvc_DeleteVMClaim_FullMethodName           = "/vmclaim.VMClaimSvc/DeleteVMClaim"
	VMClaimSvc_DeleteCollectionVMClaim_FullMethodName = "/vmclaim.VMClaimSvc/DeleteCollectionVMClaim"
	VMClaimSvc_ListVMClaim_FullMethodName             = "/vmclaim.VMClaimSvc/ListVMClaim"
	VMClaimSvc_AddToWorkqueue_FullMethodName          = "/vmclaim.VMClaimSvc/AddToWorkqueue"
)

// VMClaimSvcClient is the client API for VMClaimSvc service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type VMClaimSvcClient interface {
	CreateVMClaim(ctx context.Context, in *CreateVMClaimRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	GetVMClaim(ctx context.Context, in *general.GetRequest, opts ...grpc.CallOption) (*VMClaim, error)
	UpdateVMClaim(ctx context.Context, in *UpdateVMClaimRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	UpdateVMClaimStatus(ctx context.Context, in *UpdateVMClaimStatusRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	DeleteVMClaim(ctx context.Context, in *general.ResourceId, opts ...grpc.CallOption) (*emptypb.Empty, error)
	DeleteCollectionVMClaim(ctx context.Context, in *general.ListOptions, opts ...grpc.CallOption) (*emptypb.Empty, error)
	ListVMClaim(ctx context.Context, in *general.ListOptions, opts ...grpc.CallOption) (*ListVMClaimsResponse, error)
	AddToWorkqueue(ctx context.Context, in *general.ResourceId, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type vMClaimSvcClient struct {
	cc grpc.ClientConnInterface
}

func NewVMClaimSvcClient(cc grpc.ClientConnInterface) VMClaimSvcClient {
	return &vMClaimSvcClient{cc}
}

func (c *vMClaimSvcClient) CreateVMClaim(ctx context.Context, in *CreateVMClaimRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, VMClaimSvc_CreateVMClaim_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vMClaimSvcClient) GetVMClaim(ctx context.Context, in *general.GetRequest, opts ...grpc.CallOption) (*VMClaim, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(VMClaim)
	err := c.cc.Invoke(ctx, VMClaimSvc_GetVMClaim_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vMClaimSvcClient) UpdateVMClaim(ctx context.Context, in *UpdateVMClaimRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, VMClaimSvc_UpdateVMClaim_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vMClaimSvcClient) UpdateVMClaimStatus(ctx context.Context, in *UpdateVMClaimStatusRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, VMClaimSvc_UpdateVMClaimStatus_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vMClaimSvcClient) DeleteVMClaim(ctx context.Context, in *general.ResourceId, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, VMClaimSvc_DeleteVMClaim_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vMClaimSvcClient) DeleteCollectionVMClaim(ctx context.Context, in *general.ListOptions, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, VMClaimSvc_DeleteCollectionVMClaim_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vMClaimSvcClient) ListVMClaim(ctx context.Context, in *general.ListOptions, opts ...grpc.CallOption) (*ListVMClaimsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ListVMClaimsResponse)
	err := c.cc.Invoke(ctx, VMClaimSvc_ListVMClaim_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vMClaimSvcClient) AddToWorkqueue(ctx context.Context, in *general.ResourceId, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, VMClaimSvc_AddToWorkqueue_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// VMClaimSvcServer is the server API for VMClaimSvc service.
// All implementations must embed UnimplementedVMClaimSvcServer
// for forward compatibility.
type VMClaimSvcServer interface {
	CreateVMClaim(context.Context, *CreateVMClaimRequest) (*emptypb.Empty, error)
	GetVMClaim(context.Context, *general.GetRequest) (*VMClaim, error)
	UpdateVMClaim(context.Context, *UpdateVMClaimRequest) (*emptypb.Empty, error)
	UpdateVMClaimStatus(context.Context, *UpdateVMClaimStatusRequest) (*emptypb.Empty, error)
	DeleteVMClaim(context.Context, *general.ResourceId) (*emptypb.Empty, error)
	DeleteCollectionVMClaim(context.Context, *general.ListOptions) (*emptypb.Empty, error)
	ListVMClaim(context.Context, *general.ListOptions) (*ListVMClaimsResponse, error)
	AddToWorkqueue(context.Context, *general.ResourceId) (*emptypb.Empty, error)
	mustEmbedUnimplementedVMClaimSvcServer()
}

// UnimplementedVMClaimSvcServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedVMClaimSvcServer struct{}

func (UnimplementedVMClaimSvcServer) CreateVMClaim(context.Context, *CreateVMClaimRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateVMClaim not implemented")
}
func (UnimplementedVMClaimSvcServer) GetVMClaim(context.Context, *general.GetRequest) (*VMClaim, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetVMClaim not implemented")
}
func (UnimplementedVMClaimSvcServer) UpdateVMClaim(context.Context, *UpdateVMClaimRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateVMClaim not implemented")
}
func (UnimplementedVMClaimSvcServer) UpdateVMClaimStatus(context.Context, *UpdateVMClaimStatusRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateVMClaimStatus not implemented")
}
func (UnimplementedVMClaimSvcServer) DeleteVMClaim(context.Context, *general.ResourceId) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteVMClaim not implemented")
}
func (UnimplementedVMClaimSvcServer) DeleteCollectionVMClaim(context.Context, *general.ListOptions) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteCollectionVMClaim not implemented")
}
func (UnimplementedVMClaimSvcServer) ListVMClaim(context.Context, *general.ListOptions) (*ListVMClaimsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListVMClaim not implemented")
}
func (UnimplementedVMClaimSvcServer) AddToWorkqueue(context.Context, *general.ResourceId) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddToWorkqueue not implemented")
}
func (UnimplementedVMClaimSvcServer) mustEmbedUnimplementedVMClaimSvcServer() {}
func (UnimplementedVMClaimSvcServer) testEmbeddedByValue()                    {}

// UnsafeVMClaimSvcServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to VMClaimSvcServer will
// result in compilation errors.
type UnsafeVMClaimSvcServer interface {
	mustEmbedUnimplementedVMClaimSvcServer()
}

func RegisterVMClaimSvcServer(s grpc.ServiceRegistrar, srv VMClaimSvcServer) {
	// If the following call pancis, it indicates UnimplementedVMClaimSvcServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&VMClaimSvc_ServiceDesc, srv)
}

func _VMClaimSvc_CreateVMClaim_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateVMClaimRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VMClaimSvcServer).CreateVMClaim(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: VMClaimSvc_CreateVMClaim_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VMClaimSvcServer).CreateVMClaim(ctx, req.(*CreateVMClaimRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VMClaimSvc_GetVMClaim_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(general.GetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VMClaimSvcServer).GetVMClaim(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: VMClaimSvc_GetVMClaim_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VMClaimSvcServer).GetVMClaim(ctx, req.(*general.GetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VMClaimSvc_UpdateVMClaim_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateVMClaimRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VMClaimSvcServer).UpdateVMClaim(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: VMClaimSvc_UpdateVMClaim_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VMClaimSvcServer).UpdateVMClaim(ctx, req.(*UpdateVMClaimRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VMClaimSvc_UpdateVMClaimStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateVMClaimStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VMClaimSvcServer).UpdateVMClaimStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: VMClaimSvc_UpdateVMClaimStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VMClaimSvcServer).UpdateVMClaimStatus(ctx, req.(*UpdateVMClaimStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VMClaimSvc_DeleteVMClaim_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(general.ResourceId)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VMClaimSvcServer).DeleteVMClaim(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: VMClaimSvc_DeleteVMClaim_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VMClaimSvcServer).DeleteVMClaim(ctx, req.(*general.ResourceId))
	}
	return interceptor(ctx, in, info, handler)
}

func _VMClaimSvc_DeleteCollectionVMClaim_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(general.ListOptions)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VMClaimSvcServer).DeleteCollectionVMClaim(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: VMClaimSvc_DeleteCollectionVMClaim_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VMClaimSvcServer).DeleteCollectionVMClaim(ctx, req.(*general.ListOptions))
	}
	return interceptor(ctx, in, info, handler)
}

func _VMClaimSvc_ListVMClaim_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(general.ListOptions)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VMClaimSvcServer).ListVMClaim(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: VMClaimSvc_ListVMClaim_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VMClaimSvcServer).ListVMClaim(ctx, req.(*general.ListOptions))
	}
	return interceptor(ctx, in, info, handler)
}

func _VMClaimSvc_AddToWorkqueue_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(general.ResourceId)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VMClaimSvcServer).AddToWorkqueue(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: VMClaimSvc_AddToWorkqueue_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VMClaimSvcServer).AddToWorkqueue(ctx, req.(*general.ResourceId))
	}
	return interceptor(ctx, in, info, handler)
}

// VMClaimSvc_ServiceDesc is the grpc.ServiceDesc for VMClaimSvc service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var VMClaimSvc_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "vmclaim.VMClaimSvc",
	HandlerType: (*VMClaimSvcServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateVMClaim",
			Handler:    _VMClaimSvc_CreateVMClaim_Handler,
		},
		{
			MethodName: "GetVMClaim",
			Handler:    _VMClaimSvc_GetVMClaim_Handler,
		},
		{
			MethodName: "UpdateVMClaim",
			Handler:    _VMClaimSvc_UpdateVMClaim_Handler,
		},
		{
			MethodName: "UpdateVMClaimStatus",
			Handler:    _VMClaimSvc_UpdateVMClaimStatus_Handler,
		},
		{
			MethodName: "DeleteVMClaim",
			Handler:    _VMClaimSvc_DeleteVMClaim_Handler,
		},
		{
			MethodName: "DeleteCollectionVMClaim",
			Handler:    _VMClaimSvc_DeleteCollectionVMClaim_Handler,
		},
		{
			MethodName: "ListVMClaim",
			Handler:    _VMClaimSvc_ListVMClaim_Handler,
		},
		{
			MethodName: "AddToWorkqueue",
			Handler:    _VMClaimSvc_AddToWorkqueue_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "vmclaim/virtualmachineclaim.proto",
}
