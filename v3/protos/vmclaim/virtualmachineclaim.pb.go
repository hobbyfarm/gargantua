// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        v3.21.12
// source: vmclaim/virtualmachineclaim.proto

package vmclaim

import (
	general "github.com/hobbyfarm/gargantua/v3/protos/general"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type VMClaim struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id                  string                `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	UserId              string                `protobuf:"bytes,2,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	RestrictedBind      bool                  `protobuf:"varint,3,opt,name=restricted_bind,json=restrictedBind,proto3" json:"restricted_bind,omitempty"`
	RestrictedBindValue string                `protobuf:"bytes,4,opt,name=restricted_bind_value,json=restrictedBindValue,proto3" json:"restricted_bind_value,omitempty"`
	Vms                 map[string]*VMClaimVM `protobuf:"bytes,5,rep,name=vms,proto3" json:"vms,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	DynamicCapable      bool                  `protobuf:"varint,6,opt,name=dynamic_capable,json=dynamicCapable,proto3" json:"dynamic_capable,omitempty"`
	BaseName            string                `protobuf:"bytes,7,opt,name=base_name,json=baseName,proto3" json:"base_name,omitempty"`
	Labels              map[string]string     `protobuf:"bytes,8,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Status              *VMClaimStatus        `protobuf:"bytes,9,opt,name=status,proto3" json:"status,omitempty"`
}

func (x *VMClaim) Reset() {
	*x = VMClaim{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vmclaim_virtualmachineclaim_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VMClaim) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VMClaim) ProtoMessage() {}

func (x *VMClaim) ProtoReflect() protoreflect.Message {
	mi := &file_vmclaim_virtualmachineclaim_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VMClaim.ProtoReflect.Descriptor instead.
func (*VMClaim) Descriptor() ([]byte, []int) {
	return file_vmclaim_virtualmachineclaim_proto_rawDescGZIP(), []int{0}
}

func (x *VMClaim) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *VMClaim) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *VMClaim) GetRestrictedBind() bool {
	if x != nil {
		return x.RestrictedBind
	}
	return false
}

func (x *VMClaim) GetRestrictedBindValue() string {
	if x != nil {
		return x.RestrictedBindValue
	}
	return ""
}

func (x *VMClaim) GetVms() map[string]*VMClaimVM {
	if x != nil {
		return x.Vms
	}
	return nil
}

func (x *VMClaim) GetDynamicCapable() bool {
	if x != nil {
		return x.DynamicCapable
	}
	return false
}

func (x *VMClaim) GetBaseName() string {
	if x != nil {
		return x.BaseName
	}
	return ""
}

func (x *VMClaim) GetLabels() map[string]string {
	if x != nil {
		return x.Labels
	}
	return nil
}

func (x *VMClaim) GetStatus() *VMClaimStatus {
	if x != nil {
		return x.Status
	}
	return nil
}

type CreateVMClaimRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id                  string            `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	UserName            string            `protobuf:"bytes,2,opt,name=user_name,json=userName,proto3" json:"user_name,omitempty"`
	Vmset               map[string]string `protobuf:"bytes,3,rep,name=vmset,proto3" json:"vmset,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	RestrictedBind      bool              `protobuf:"varint,4,opt,name=restricted_bind,json=restrictedBind,proto3" json:"restricted_bind,omitempty"`
	RestrictedBindValue string            `protobuf:"bytes,5,opt,name=restricted_bind_value,json=restrictedBindValue,proto3" json:"restricted_bind_value,omitempty"`
	DynamicCapable      bool              `protobuf:"varint,6,opt,name=dynamic_capable,json=dynamicCapable,proto3" json:"dynamic_capable,omitempty"`
	Labels              map[string]string `protobuf:"bytes,7,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *CreateVMClaimRequest) Reset() {
	*x = CreateVMClaimRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vmclaim_virtualmachineclaim_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateVMClaimRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateVMClaimRequest) ProtoMessage() {}

func (x *CreateVMClaimRequest) ProtoReflect() protoreflect.Message {
	mi := &file_vmclaim_virtualmachineclaim_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateVMClaimRequest.ProtoReflect.Descriptor instead.
func (*CreateVMClaimRequest) Descriptor() ([]byte, []int) {
	return file_vmclaim_virtualmachineclaim_proto_rawDescGZIP(), []int{1}
}

func (x *CreateVMClaimRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *CreateVMClaimRequest) GetUserName() string {
	if x != nil {
		return x.UserName
	}
	return ""
}

func (x *CreateVMClaimRequest) GetVmset() map[string]string {
	if x != nil {
		return x.Vmset
	}
	return nil
}

func (x *CreateVMClaimRequest) GetRestrictedBind() bool {
	if x != nil {
		return x.RestrictedBind
	}
	return false
}

func (x *CreateVMClaimRequest) GetRestrictedBindValue() string {
	if x != nil {
		return x.RestrictedBindValue
	}
	return ""
}

func (x *CreateVMClaimRequest) GetDynamicCapable() bool {
	if x != nil {
		return x.DynamicCapable
	}
	return false
}

func (x *CreateVMClaimRequest) GetLabels() map[string]string {
	if x != nil {
		return x.Labels
	}
	return nil
}

type UpdateVMClaimRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id             string                `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Vmset          map[string]*VMClaimVM `protobuf:"bytes,2,rep,name=vmset,proto3" json:"vmset,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	RestrictedBind *wrapperspb.BoolValue `protobuf:"bytes,3,opt,name=restricted_bind,json=restrictedBind,proto3" json:"restricted_bind,omitempty"`
}

func (x *UpdateVMClaimRequest) Reset() {
	*x = UpdateVMClaimRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vmclaim_virtualmachineclaim_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UpdateVMClaimRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateVMClaimRequest) ProtoMessage() {}

func (x *UpdateVMClaimRequest) ProtoReflect() protoreflect.Message {
	mi := &file_vmclaim_virtualmachineclaim_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateVMClaimRequest.ProtoReflect.Descriptor instead.
func (*UpdateVMClaimRequest) Descriptor() ([]byte, []int) {
	return file_vmclaim_virtualmachineclaim_proto_rawDescGZIP(), []int{2}
}

func (x *UpdateVMClaimRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *UpdateVMClaimRequest) GetVmset() map[string]*VMClaimVM {
	if x != nil {
		return x.Vmset
	}
	return nil
}

func (x *UpdateVMClaimRequest) GetRestrictedBind() *wrapperspb.BoolValue {
	if x != nil {
		return x.RestrictedBind
	}
	return nil
}

type UpdateVMClaimStatusRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id                 string                  `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Bindmode           string                  `protobuf:"bytes,2,opt,name=bindmode,proto3" json:"bindmode,omitempty"`
	StaticBindAttempts *wrapperspb.UInt32Value `protobuf:"bytes,3,opt,name=static_bind_attempts,json=staticBindAttempts,proto3" json:"static_bind_attempts,omitempty"`
	Bound              *wrapperspb.BoolValue   `protobuf:"bytes,4,opt,name=bound,proto3" json:"bound,omitempty"`
	Ready              *wrapperspb.BoolValue   `protobuf:"bytes,5,opt,name=ready,proto3" json:"ready,omitempty"`
	Tainted            *wrapperspb.BoolValue   `protobuf:"bytes,6,opt,name=tainted,proto3" json:"tainted,omitempty"`
}

func (x *UpdateVMClaimStatusRequest) Reset() {
	*x = UpdateVMClaimStatusRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vmclaim_virtualmachineclaim_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UpdateVMClaimStatusRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateVMClaimStatusRequest) ProtoMessage() {}

func (x *UpdateVMClaimStatusRequest) ProtoReflect() protoreflect.Message {
	mi := &file_vmclaim_virtualmachineclaim_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateVMClaimStatusRequest.ProtoReflect.Descriptor instead.
func (*UpdateVMClaimStatusRequest) Descriptor() ([]byte, []int) {
	return file_vmclaim_virtualmachineclaim_proto_rawDescGZIP(), []int{3}
}

func (x *UpdateVMClaimStatusRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *UpdateVMClaimStatusRequest) GetBindmode() string {
	if x != nil {
		return x.Bindmode
	}
	return ""
}

func (x *UpdateVMClaimStatusRequest) GetStaticBindAttempts() *wrapperspb.UInt32Value {
	if x != nil {
		return x.StaticBindAttempts
	}
	return nil
}

func (x *UpdateVMClaimStatusRequest) GetBound() *wrapperspb.BoolValue {
	if x != nil {
		return x.Bound
	}
	return nil
}

func (x *UpdateVMClaimStatusRequest) GetReady() *wrapperspb.BoolValue {
	if x != nil {
		return x.Ready
	}
	return nil
}

func (x *UpdateVMClaimStatusRequest) GetTainted() *wrapperspb.BoolValue {
	if x != nil {
		return x.Tainted
	}
	return nil
}

type VMClaimStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Bindmode           string `protobuf:"bytes,1,opt,name=bindmode,proto3" json:"bindmode,omitempty"`
	StaticBindAttempts uint32 `protobuf:"varint,2,opt,name=static_bind_attempts,json=staticBindAttempts,proto3" json:"static_bind_attempts,omitempty"`
	Bound              bool   `protobuf:"varint,3,opt,name=bound,proto3" json:"bound,omitempty"`
	Ready              bool   `protobuf:"varint,4,opt,name=ready,proto3" json:"ready,omitempty"`
	Tainted            bool   `protobuf:"varint,5,opt,name=tainted,proto3" json:"tainted,omitempty"`
}

func (x *VMClaimStatus) Reset() {
	*x = VMClaimStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vmclaim_virtualmachineclaim_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VMClaimStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VMClaimStatus) ProtoMessage() {}

func (x *VMClaimStatus) ProtoReflect() protoreflect.Message {
	mi := &file_vmclaim_virtualmachineclaim_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VMClaimStatus.ProtoReflect.Descriptor instead.
func (*VMClaimStatus) Descriptor() ([]byte, []int) {
	return file_vmclaim_virtualmachineclaim_proto_rawDescGZIP(), []int{4}
}

func (x *VMClaimStatus) GetBindmode() string {
	if x != nil {
		return x.Bindmode
	}
	return ""
}

func (x *VMClaimStatus) GetStaticBindAttempts() uint32 {
	if x != nil {
		return x.StaticBindAttempts
	}
	return 0
}

func (x *VMClaimStatus) GetBound() bool {
	if x != nil {
		return x.Bound
	}
	return false
}

func (x *VMClaimStatus) GetReady() bool {
	if x != nil {
		return x.Ready
	}
	return false
}

func (x *VMClaimStatus) GetTainted() bool {
	if x != nil {
		return x.Tainted
	}
	return false
}

type VMClaimVM struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Template         string `protobuf:"bytes,1,opt,name=template,proto3" json:"template,omitempty"`
	VirtualMachineId string `protobuf:"bytes,2,opt,name=virtual_machine_id,json=virtualMachineId,proto3" json:"virtual_machine_id,omitempty"`
}

func (x *VMClaimVM) Reset() {
	*x = VMClaimVM{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vmclaim_virtualmachineclaim_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VMClaimVM) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VMClaimVM) ProtoMessage() {}

func (x *VMClaimVM) ProtoReflect() protoreflect.Message {
	mi := &file_vmclaim_virtualmachineclaim_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VMClaimVM.ProtoReflect.Descriptor instead.
func (*VMClaimVM) Descriptor() ([]byte, []int) {
	return file_vmclaim_virtualmachineclaim_proto_rawDescGZIP(), []int{5}
}

func (x *VMClaimVM) GetTemplate() string {
	if x != nil {
		return x.Template
	}
	return ""
}

func (x *VMClaimVM) GetVirtualMachineId() string {
	if x != nil {
		return x.VirtualMachineId
	}
	return ""
}

type ListVMClaimsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Vmclaims []*VMClaim `protobuf:"bytes,1,rep,name=vmclaims,proto3" json:"vmclaims,omitempty"`
}

func (x *ListVMClaimsResponse) Reset() {
	*x = ListVMClaimsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vmclaim_virtualmachineclaim_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListVMClaimsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListVMClaimsResponse) ProtoMessage() {}

func (x *ListVMClaimsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_vmclaim_virtualmachineclaim_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListVMClaimsResponse.ProtoReflect.Descriptor instead.
func (*ListVMClaimsResponse) Descriptor() ([]byte, []int) {
	return file_vmclaim_virtualmachineclaim_proto_rawDescGZIP(), []int{6}
}

func (x *ListVMClaimsResponse) GetVmclaims() []*VMClaim {
	if x != nil {
		return x.Vmclaims
	}
	return nil
}

var File_vmclaim_virtualmachineclaim_proto protoreflect.FileDescriptor

var file_vmclaim_virtualmachineclaim_proto_rawDesc = []byte{
	0x0a, 0x21, 0x76, 0x6d, 0x63, 0x6c, 0x61, 0x69, 0x6d, 0x2f, 0x76, 0x69, 0x72, 0x74, 0x75, 0x61,
	0x6c, 0x6d, 0x61, 0x63, 0x68, 0x69, 0x6e, 0x65, 0x63, 0x6c, 0x61, 0x69, 0x6d, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x07, 0x76, 0x6d, 0x63, 0x6c, 0x61, 0x69, 0x6d, 0x1a, 0x15, 0x67, 0x65,
	0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2f, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2f, 0x77, 0x72, 0x61, 0x70, 0x70, 0x65, 0x72, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x22, 0xef, 0x03, 0x0a, 0x07, 0x56, 0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x12, 0x0e, 0x0a, 0x02,
	0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x17, 0x0a, 0x07,
	0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x75,
	0x73, 0x65, 0x72, 0x49, 0x64, 0x12, 0x27, 0x0a, 0x0f, 0x72, 0x65, 0x73, 0x74, 0x72, 0x69, 0x63,
	0x74, 0x65, 0x64, 0x5f, 0x62, 0x69, 0x6e, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0e,
	0x72, 0x65, 0x73, 0x74, 0x72, 0x69, 0x63, 0x74, 0x65, 0x64, 0x42, 0x69, 0x6e, 0x64, 0x12, 0x32,
	0x0a, 0x15, 0x72, 0x65, 0x73, 0x74, 0x72, 0x69, 0x63, 0x74, 0x65, 0x64, 0x5f, 0x62, 0x69, 0x6e,
	0x64, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x13, 0x72,
	0x65, 0x73, 0x74, 0x72, 0x69, 0x63, 0x74, 0x65, 0x64, 0x42, 0x69, 0x6e, 0x64, 0x56, 0x61, 0x6c,
	0x75, 0x65, 0x12, 0x2b, 0x0a, 0x03, 0x76, 0x6d, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x19, 0x2e, 0x76, 0x6d, 0x63, 0x6c, 0x61, 0x69, 0x6d, 0x2e, 0x56, 0x4d, 0x43, 0x6c, 0x61, 0x69,
	0x6d, 0x2e, 0x56, 0x6d, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x03, 0x76, 0x6d, 0x73, 0x12,
	0x27, 0x0a, 0x0f, 0x64, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63, 0x5f, 0x63, 0x61, 0x70, 0x61, 0x62,
	0x6c, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0e, 0x64, 0x79, 0x6e, 0x61, 0x6d, 0x69,
	0x63, 0x43, 0x61, 0x70, 0x61, 0x62, 0x6c, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x62, 0x61, 0x73, 0x65,
	0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x62, 0x61, 0x73,
	0x65, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x34, 0x0a, 0x06, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x18,
	0x08, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x76, 0x6d, 0x63, 0x6c, 0x61, 0x69, 0x6d, 0x2e,
	0x56, 0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x45, 0x6e,
	0x74, 0x72, 0x79, 0x52, 0x06, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x12, 0x2e, 0x0a, 0x06, 0x73,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x09, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x76, 0x6d,
	0x63, 0x6c, 0x61, 0x69, 0x6d, 0x2e, 0x56, 0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x53, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x1a, 0x4a, 0x0a, 0x08, 0x56,
	0x6d, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x28, 0x0a, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x76, 0x6d, 0x63, 0x6c, 0x61,
	0x69, 0x6d, 0x2e, 0x56, 0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x56, 0x4d, 0x52, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x39, 0x0a, 0x0b, 0x4c, 0x61, 0x62, 0x65, 0x6c,
	0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02,
	0x38, 0x01, 0x22, 0xc1, 0x03, 0x0a, 0x14, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x56, 0x4d, 0x43,
	0x6c, 0x61, 0x69, 0x6d, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x1b, 0x0a, 0x09, 0x75,
	0x73, 0x65, 0x72, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08,
	0x75, 0x73, 0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x3e, 0x0a, 0x05, 0x76, 0x6d, 0x73, 0x65,
	0x74, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x76, 0x6d, 0x63, 0x6c, 0x61, 0x69,
	0x6d, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x56, 0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x56, 0x6d, 0x73, 0x65, 0x74, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x52, 0x05, 0x76, 0x6d, 0x73, 0x65, 0x74, 0x12, 0x27, 0x0a, 0x0f, 0x72, 0x65, 0x73, 0x74,
	0x72, 0x69, 0x63, 0x74, 0x65, 0x64, 0x5f, 0x62, 0x69, 0x6e, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x08, 0x52, 0x0e, 0x72, 0x65, 0x73, 0x74, 0x72, 0x69, 0x63, 0x74, 0x65, 0x64, 0x42, 0x69, 0x6e,
	0x64, 0x12, 0x32, 0x0a, 0x15, 0x72, 0x65, 0x73, 0x74, 0x72, 0x69, 0x63, 0x74, 0x65, 0x64, 0x5f,
	0x62, 0x69, 0x6e, 0x64, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x13, 0x72, 0x65, 0x73, 0x74, 0x72, 0x69, 0x63, 0x74, 0x65, 0x64, 0x42, 0x69, 0x6e, 0x64,
	0x56, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x27, 0x0a, 0x0f, 0x64, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63,
	0x5f, 0x63, 0x61, 0x70, 0x61, 0x62, 0x6c, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0e,
	0x64, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63, 0x43, 0x61, 0x70, 0x61, 0x62, 0x6c, 0x65, 0x12, 0x41,
	0x0a, 0x06, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x18, 0x07, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x29,
	0x2e, 0x76, 0x6d, 0x63, 0x6c, 0x61, 0x69, 0x6d, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x56,
	0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x4c, 0x61,
	0x62, 0x65, 0x6c, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x6c, 0x61, 0x62, 0x65, 0x6c,
	0x73, 0x1a, 0x38, 0x0a, 0x0a, 0x56, 0x6d, 0x73, 0x65, 0x74, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12,
	0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65,
	0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x39, 0x0a, 0x0b, 0x4c,
	0x61, 0x62, 0x65, 0x6c, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65,
	0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xf9, 0x01, 0x0a, 0x14, 0x55, 0x70, 0x64, 0x61, 0x74,
	0x65, 0x56, 0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12,
	0x3e, 0x0a, 0x05, 0x76, 0x6d, 0x73, 0x65, 0x74, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x28,
	0x2e, 0x76, 0x6d, 0x63, 0x6c, 0x61, 0x69, 0x6d, 0x2e, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x56,
	0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x56, 0x6d,
	0x73, 0x65, 0x74, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x05, 0x76, 0x6d, 0x73, 0x65, 0x74, 0x12,
	0x43, 0x0a, 0x0f, 0x72, 0x65, 0x73, 0x74, 0x72, 0x69, 0x63, 0x74, 0x65, 0x64, 0x5f, 0x62, 0x69,
	0x6e, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x42, 0x6f, 0x6f, 0x6c, 0x56,
	0x61, 0x6c, 0x75, 0x65, 0x52, 0x0e, 0x72, 0x65, 0x73, 0x74, 0x72, 0x69, 0x63, 0x74, 0x65, 0x64,
	0x42, 0x69, 0x6e, 0x64, 0x1a, 0x4c, 0x0a, 0x0a, 0x56, 0x6d, 0x73, 0x65, 0x74, 0x45, 0x6e, 0x74,
	0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x03, 0x6b, 0x65, 0x79, 0x12, 0x28, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x76, 0x6d, 0x63, 0x6c, 0x61, 0x69, 0x6d, 0x2e, 0x56, 0x4d,
	0x43, 0x6c, 0x61, 0x69, 0x6d, 0x56, 0x4d, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02,
	0x38, 0x01, 0x22, 0xb2, 0x02, 0x0a, 0x1a, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x56, 0x4d, 0x43,
	0x6c, 0x61, 0x69, 0x6d, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69,
	0x64, 0x12, 0x1a, 0x0a, 0x08, 0x62, 0x69, 0x6e, 0x64, 0x6d, 0x6f, 0x64, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x08, 0x62, 0x69, 0x6e, 0x64, 0x6d, 0x6f, 0x64, 0x65, 0x12, 0x4e, 0x0a,
	0x14, 0x73, 0x74, 0x61, 0x74, 0x69, 0x63, 0x5f, 0x62, 0x69, 0x6e, 0x64, 0x5f, 0x61, 0x74, 0x74,
	0x65, 0x6d, 0x70, 0x74, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x55, 0x49,
	0x6e, 0x74, 0x33, 0x32, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x12, 0x73, 0x74, 0x61, 0x74, 0x69,
	0x63, 0x42, 0x69, 0x6e, 0x64, 0x41, 0x74, 0x74, 0x65, 0x6d, 0x70, 0x74, 0x73, 0x12, 0x30, 0x0a,
	0x05, 0x62, 0x6f, 0x75, 0x6e, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x42,
	0x6f, 0x6f, 0x6c, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x05, 0x62, 0x6f, 0x75, 0x6e, 0x64, 0x12,
	0x30, 0x0a, 0x05, 0x72, 0x65, 0x61, 0x64, 0x79, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x42, 0x6f, 0x6f, 0x6c, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x05, 0x72, 0x65, 0x61, 0x64,
	0x79, 0x12, 0x34, 0x0a, 0x07, 0x74, 0x61, 0x69, 0x6e, 0x74, 0x65, 0x64, 0x18, 0x06, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x42, 0x6f, 0x6f, 0x6c, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x07,
	0x74, 0x61, 0x69, 0x6e, 0x74, 0x65, 0x64, 0x22, 0xa3, 0x01, 0x0a, 0x0d, 0x56, 0x4d, 0x43, 0x6c,
	0x61, 0x69, 0x6d, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x1a, 0x0a, 0x08, 0x62, 0x69, 0x6e,
	0x64, 0x6d, 0x6f, 0x64, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x62, 0x69, 0x6e,
	0x64, 0x6d, 0x6f, 0x64, 0x65, 0x12, 0x30, 0x0a, 0x14, 0x73, 0x74, 0x61, 0x74, 0x69, 0x63, 0x5f,
	0x62, 0x69, 0x6e, 0x64, 0x5f, 0x61, 0x74, 0x74, 0x65, 0x6d, 0x70, 0x74, 0x73, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0d, 0x52, 0x12, 0x73, 0x74, 0x61, 0x74, 0x69, 0x63, 0x42, 0x69, 0x6e, 0x64, 0x41,
	0x74, 0x74, 0x65, 0x6d, 0x70, 0x74, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x62, 0x6f, 0x75, 0x6e, 0x64,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x05, 0x62, 0x6f, 0x75, 0x6e, 0x64, 0x12, 0x14, 0x0a,
	0x05, 0x72, 0x65, 0x61, 0x64, 0x79, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x05, 0x72, 0x65,
	0x61, 0x64, 0x79, 0x12, 0x18, 0x0a, 0x07, 0x74, 0x61, 0x69, 0x6e, 0x74, 0x65, 0x64, 0x18, 0x05,
	0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x74, 0x61, 0x69, 0x6e, 0x74, 0x65, 0x64, 0x22, 0x55, 0x0a,
	0x09, 0x56, 0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x56, 0x4d, 0x12, 0x1a, 0x0a, 0x08, 0x74, 0x65,
	0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x74, 0x65,
	0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x12, 0x2c, 0x0a, 0x12, 0x76, 0x69, 0x72, 0x74, 0x75, 0x61,
	0x6c, 0x5f, 0x6d, 0x61, 0x63, 0x68, 0x69, 0x6e, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x10, 0x76, 0x69, 0x72, 0x74, 0x75, 0x61, 0x6c, 0x4d, 0x61, 0x63, 0x68, 0x69,
	0x6e, 0x65, 0x49, 0x64, 0x22, 0x44, 0x0a, 0x14, 0x4c, 0x69, 0x73, 0x74, 0x56, 0x4d, 0x43, 0x6c,
	0x61, 0x69, 0x6d, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x2c, 0x0a, 0x08,
	0x76, 0x6d, 0x63, 0x6c, 0x61, 0x69, 0x6d, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x10,
	0x2e, 0x76, 0x6d, 0x63, 0x6c, 0x61, 0x69, 0x6d, 0x2e, 0x56, 0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d,
	0x52, 0x08, 0x76, 0x6d, 0x63, 0x6c, 0x61, 0x69, 0x6d, 0x73, 0x32, 0xf0, 0x03, 0x0a, 0x0a, 0x56,
	0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x53, 0x76, 0x63, 0x12, 0x46, 0x0a, 0x0d, 0x43, 0x72, 0x65,
	0x61, 0x74, 0x65, 0x56, 0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x12, 0x1d, 0x2e, 0x76, 0x6d, 0x63,
	0x6c, 0x61, 0x69, 0x6d, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x56, 0x4d, 0x43, 0x6c, 0x61,
	0x69, 0x6d, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74,
	0x79, 0x12, 0x33, 0x0a, 0x0a, 0x47, 0x65, 0x74, 0x56, 0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x12,
	0x13, 0x2e, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72,
	0x63, 0x65, 0x49, 0x64, 0x1a, 0x10, 0x2e, 0x76, 0x6d, 0x63, 0x6c, 0x61, 0x69, 0x6d, 0x2e, 0x56,
	0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x12, 0x46, 0x0a, 0x0d, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65,
	0x56, 0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x12, 0x1d, 0x2e, 0x76, 0x6d, 0x63, 0x6c, 0x61, 0x69,
	0x6d, 0x2e, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x56, 0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x52,
	0x0a, 0x13, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x56, 0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x53,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x23, 0x2e, 0x76, 0x6d, 0x63, 0x6c, 0x61, 0x69, 0x6d, 0x2e,
	0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x56, 0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x53, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70,
	0x74, 0x79, 0x12, 0x3c, 0x0a, 0x0d, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x56, 0x4d, 0x43, 0x6c,
	0x61, 0x69, 0x6d, 0x12, 0x13, 0x2e, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x52, 0x65,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x49, 0x64, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79,
	0x12, 0x47, 0x0a, 0x17, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x43, 0x6f, 0x6c, 0x6c, 0x65, 0x63,
	0x74, 0x69, 0x6f, 0x6e, 0x56, 0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x12, 0x14, 0x2e, 0x67, 0x65,
	0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e,
	0x73, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x42, 0x0a, 0x0b, 0x4c, 0x69, 0x73,
	0x74, 0x56, 0x4d, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x12, 0x14, 0x2e, 0x67, 0x65, 0x6e, 0x65, 0x72,
	0x61, 0x6c, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x1a, 0x1d,
	0x2e, 0x76, 0x6d, 0x63, 0x6c, 0x61, 0x69, 0x6d, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x56, 0x4d, 0x43,
	0x6c, 0x61, 0x69, 0x6d, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x32, 0x5a,
	0x30, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x68, 0x6f, 0x62, 0x62,
	0x79, 0x66, 0x61, 0x72, 0x6d, 0x2f, 0x67, 0x61, 0x72, 0x67, 0x61, 0x6e, 0x74, 0x75, 0x61, 0x2f,
	0x76, 0x33, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2f, 0x76, 0x6d, 0x63, 0x6c, 0x61, 0x69,
	0x6d, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_vmclaim_virtualmachineclaim_proto_rawDescOnce sync.Once
	file_vmclaim_virtualmachineclaim_proto_rawDescData = file_vmclaim_virtualmachineclaim_proto_rawDesc
)

func file_vmclaim_virtualmachineclaim_proto_rawDescGZIP() []byte {
	file_vmclaim_virtualmachineclaim_proto_rawDescOnce.Do(func() {
		file_vmclaim_virtualmachineclaim_proto_rawDescData = protoimpl.X.CompressGZIP(file_vmclaim_virtualmachineclaim_proto_rawDescData)
	})
	return file_vmclaim_virtualmachineclaim_proto_rawDescData
}

var file_vmclaim_virtualmachineclaim_proto_msgTypes = make([]protoimpl.MessageInfo, 12)
var file_vmclaim_virtualmachineclaim_proto_goTypes = []interface{}{
	(*VMClaim)(nil),                    // 0: vmclaim.VMClaim
	(*CreateVMClaimRequest)(nil),       // 1: vmclaim.CreateVMClaimRequest
	(*UpdateVMClaimRequest)(nil),       // 2: vmclaim.UpdateVMClaimRequest
	(*UpdateVMClaimStatusRequest)(nil), // 3: vmclaim.UpdateVMClaimStatusRequest
	(*VMClaimStatus)(nil),              // 4: vmclaim.VMClaimStatus
	(*VMClaimVM)(nil),                  // 5: vmclaim.VMClaimVM
	(*ListVMClaimsResponse)(nil),       // 6: vmclaim.ListVMClaimsResponse
	nil,                                // 7: vmclaim.VMClaim.VmsEntry
	nil,                                // 8: vmclaim.VMClaim.LabelsEntry
	nil,                                // 9: vmclaim.CreateVMClaimRequest.VmsetEntry
	nil,                                // 10: vmclaim.CreateVMClaimRequest.LabelsEntry
	nil,                                // 11: vmclaim.UpdateVMClaimRequest.VmsetEntry
	(*wrapperspb.BoolValue)(nil),       // 12: google.protobuf.BoolValue
	(*wrapperspb.UInt32Value)(nil),     // 13: google.protobuf.UInt32Value
	(*general.ResourceId)(nil),         // 14: general.ResourceId
	(*general.ListOptions)(nil),        // 15: general.ListOptions
	(*emptypb.Empty)(nil),              // 16: google.protobuf.Empty
}
var file_vmclaim_virtualmachineclaim_proto_depIdxs = []int32{
	7,  // 0: vmclaim.VMClaim.vms:type_name -> vmclaim.VMClaim.VmsEntry
	8,  // 1: vmclaim.VMClaim.labels:type_name -> vmclaim.VMClaim.LabelsEntry
	4,  // 2: vmclaim.VMClaim.status:type_name -> vmclaim.VMClaimStatus
	9,  // 3: vmclaim.CreateVMClaimRequest.vmset:type_name -> vmclaim.CreateVMClaimRequest.VmsetEntry
	10, // 4: vmclaim.CreateVMClaimRequest.labels:type_name -> vmclaim.CreateVMClaimRequest.LabelsEntry
	11, // 5: vmclaim.UpdateVMClaimRequest.vmset:type_name -> vmclaim.UpdateVMClaimRequest.VmsetEntry
	12, // 6: vmclaim.UpdateVMClaimRequest.restricted_bind:type_name -> google.protobuf.BoolValue
	13, // 7: vmclaim.UpdateVMClaimStatusRequest.static_bind_attempts:type_name -> google.protobuf.UInt32Value
	12, // 8: vmclaim.UpdateVMClaimStatusRequest.bound:type_name -> google.protobuf.BoolValue
	12, // 9: vmclaim.UpdateVMClaimStatusRequest.ready:type_name -> google.protobuf.BoolValue
	12, // 10: vmclaim.UpdateVMClaimStatusRequest.tainted:type_name -> google.protobuf.BoolValue
	0,  // 11: vmclaim.ListVMClaimsResponse.vmclaims:type_name -> vmclaim.VMClaim
	5,  // 12: vmclaim.VMClaim.VmsEntry.value:type_name -> vmclaim.VMClaimVM
	5,  // 13: vmclaim.UpdateVMClaimRequest.VmsetEntry.value:type_name -> vmclaim.VMClaimVM
	1,  // 14: vmclaim.VMClaimSvc.CreateVMClaim:input_type -> vmclaim.CreateVMClaimRequest
	14, // 15: vmclaim.VMClaimSvc.GetVMClaim:input_type -> general.ResourceId
	2,  // 16: vmclaim.VMClaimSvc.UpdateVMClaim:input_type -> vmclaim.UpdateVMClaimRequest
	3,  // 17: vmclaim.VMClaimSvc.UpdateVMClaimStatus:input_type -> vmclaim.UpdateVMClaimStatusRequest
	14, // 18: vmclaim.VMClaimSvc.DeleteVMClaim:input_type -> general.ResourceId
	15, // 19: vmclaim.VMClaimSvc.DeleteCollectionVMClaim:input_type -> general.ListOptions
	15, // 20: vmclaim.VMClaimSvc.ListVMClaim:input_type -> general.ListOptions
	16, // 21: vmclaim.VMClaimSvc.CreateVMClaim:output_type -> google.protobuf.Empty
	0,  // 22: vmclaim.VMClaimSvc.GetVMClaim:output_type -> vmclaim.VMClaim
	16, // 23: vmclaim.VMClaimSvc.UpdateVMClaim:output_type -> google.protobuf.Empty
	16, // 24: vmclaim.VMClaimSvc.UpdateVMClaimStatus:output_type -> google.protobuf.Empty
	16, // 25: vmclaim.VMClaimSvc.DeleteVMClaim:output_type -> google.protobuf.Empty
	16, // 26: vmclaim.VMClaimSvc.DeleteCollectionVMClaim:output_type -> google.protobuf.Empty
	6,  // 27: vmclaim.VMClaimSvc.ListVMClaim:output_type -> vmclaim.ListVMClaimsResponse
	21, // [21:28] is the sub-list for method output_type
	14, // [14:21] is the sub-list for method input_type
	14, // [14:14] is the sub-list for extension type_name
	14, // [14:14] is the sub-list for extension extendee
	0,  // [0:14] is the sub-list for field type_name
}

func init() { file_vmclaim_virtualmachineclaim_proto_init() }
func file_vmclaim_virtualmachineclaim_proto_init() {
	if File_vmclaim_virtualmachineclaim_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_vmclaim_virtualmachineclaim_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VMClaim); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_vmclaim_virtualmachineclaim_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CreateVMClaimRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_vmclaim_virtualmachineclaim_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UpdateVMClaimRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_vmclaim_virtualmachineclaim_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UpdateVMClaimStatusRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_vmclaim_virtualmachineclaim_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VMClaimStatus); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_vmclaim_virtualmachineclaim_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VMClaimVM); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_vmclaim_virtualmachineclaim_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListVMClaimsResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_vmclaim_virtualmachineclaim_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   12,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_vmclaim_virtualmachineclaim_proto_goTypes,
		DependencyIndexes: file_vmclaim_virtualmachineclaim_proto_depIdxs,
		MessageInfos:      file_vmclaim_virtualmachineclaim_proto_msgTypes,
	}.Build()
	File_vmclaim_virtualmachineclaim_proto = out.File
	file_vmclaim_virtualmachineclaim_proto_rawDesc = nil
	file_vmclaim_virtualmachineclaim_proto_goTypes = nil
	file_vmclaim_virtualmachineclaim_proto_depIdxs = nil
}
