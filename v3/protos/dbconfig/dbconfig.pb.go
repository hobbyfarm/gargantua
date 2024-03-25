// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        v3.21.12
// source: dbconfig/dbconfig.proto

package dbconfig

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

type DynamicBindConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id                  string            `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Environment         string            `protobuf:"bytes,2,opt,name=environment,proto3" json:"environment,omitempty"`
	RestrictedBind      bool              `protobuf:"varint,3,opt,name=restricted_bind,json=restrictedBind,proto3" json:"restricted_bind,omitempty"`
	RestrictedBindValue string            `protobuf:"bytes,4,opt,name=restricted_bind_value,json=restrictedBindValue,proto3" json:"restricted_bind_value,omitempty"`
	BurstCountCapacity  map[string]uint32 `protobuf:"bytes,5,rep,name=burst_count_capacity,json=burstCountCapacity,proto3" json:"burst_count_capacity,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
	Labels              map[string]string `protobuf:"bytes,6,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *DynamicBindConfig) Reset() {
	*x = DynamicBindConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_dbconfig_dbconfig_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DynamicBindConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DynamicBindConfig) ProtoMessage() {}

func (x *DynamicBindConfig) ProtoReflect() protoreflect.Message {
	mi := &file_dbconfig_dbconfig_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DynamicBindConfig.ProtoReflect.Descriptor instead.
func (*DynamicBindConfig) Descriptor() ([]byte, []int) {
	return file_dbconfig_dbconfig_proto_rawDescGZIP(), []int{0}
}

func (x *DynamicBindConfig) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *DynamicBindConfig) GetEnvironment() string {
	if x != nil {
		return x.Environment
	}
	return ""
}

func (x *DynamicBindConfig) GetRestrictedBind() bool {
	if x != nil {
		return x.RestrictedBind
	}
	return false
}

func (x *DynamicBindConfig) GetRestrictedBindValue() string {
	if x != nil {
		return x.RestrictedBindValue
	}
	return ""
}

func (x *DynamicBindConfig) GetBurstCountCapacity() map[string]uint32 {
	if x != nil {
		return x.BurstCountCapacity
	}
	return nil
}

func (x *DynamicBindConfig) GetLabels() map[string]string {
	if x != nil {
		return x.Labels
	}
	return nil
}

type CreateDynamicBindConfigRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	SeName              string            `protobuf:"bytes,1,opt,name=se_name,json=seName,proto3" json:"se_name,omitempty"`
	SeUid               string            `protobuf:"bytes,2,opt,name=se_uid,json=seUid,proto3" json:"se_uid,omitempty"`
	EnvName             string            `protobuf:"bytes,3,opt,name=env_name,json=envName,proto3" json:"env_name,omitempty"`
	BurstCountCapacity  map[string]uint32 `protobuf:"bytes,4,rep,name=burst_count_capacity,json=burstCountCapacity,proto3" json:"burst_count_capacity,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
	RestrictedBind      bool              `protobuf:"varint,5,opt,name=restricted_bind,json=restrictedBind,proto3" json:"restricted_bind,omitempty"`
	RestrictedBindValue string            `protobuf:"bytes,6,opt,name=restricted_bind_value,json=restrictedBindValue,proto3" json:"restricted_bind_value,omitempty"`
}

func (x *CreateDynamicBindConfigRequest) Reset() {
	*x = CreateDynamicBindConfigRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_dbconfig_dbconfig_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateDynamicBindConfigRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateDynamicBindConfigRequest) ProtoMessage() {}

func (x *CreateDynamicBindConfigRequest) ProtoReflect() protoreflect.Message {
	mi := &file_dbconfig_dbconfig_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateDynamicBindConfigRequest.ProtoReflect.Descriptor instead.
func (*CreateDynamicBindConfigRequest) Descriptor() ([]byte, []int) {
	return file_dbconfig_dbconfig_proto_rawDescGZIP(), []int{1}
}

func (x *CreateDynamicBindConfigRequest) GetSeName() string {
	if x != nil {
		return x.SeName
	}
	return ""
}

func (x *CreateDynamicBindConfigRequest) GetSeUid() string {
	if x != nil {
		return x.SeUid
	}
	return ""
}

func (x *CreateDynamicBindConfigRequest) GetEnvName() string {
	if x != nil {
		return x.EnvName
	}
	return ""
}

func (x *CreateDynamicBindConfigRequest) GetBurstCountCapacity() map[string]uint32 {
	if x != nil {
		return x.BurstCountCapacity
	}
	return nil
}

func (x *CreateDynamicBindConfigRequest) GetRestrictedBind() bool {
	if x != nil {
		return x.RestrictedBind
	}
	return false
}

func (x *CreateDynamicBindConfigRequest) GetRestrictedBindValue() string {
	if x != nil {
		return x.RestrictedBindValue
	}
	return ""
}

type UpdateDynamicBindConfigRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id                 string                `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Environment        string                `protobuf:"bytes,2,opt,name=environment,proto3" json:"environment,omitempty"`
	RestrictedBind     *wrapperspb.BoolValue `protobuf:"bytes,3,opt,name=restricted_bind,json=restrictedBind,proto3" json:"restricted_bind,omitempty"`
	BurstCountCapacity map[string]uint32     `protobuf:"bytes,4,rep,name=burst_count_capacity,json=burstCountCapacity,proto3" json:"burst_count_capacity,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
}

func (x *UpdateDynamicBindConfigRequest) Reset() {
	*x = UpdateDynamicBindConfigRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_dbconfig_dbconfig_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UpdateDynamicBindConfigRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateDynamicBindConfigRequest) ProtoMessage() {}

func (x *UpdateDynamicBindConfigRequest) ProtoReflect() protoreflect.Message {
	mi := &file_dbconfig_dbconfig_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateDynamicBindConfigRequest.ProtoReflect.Descriptor instead.
func (*UpdateDynamicBindConfigRequest) Descriptor() ([]byte, []int) {
	return file_dbconfig_dbconfig_proto_rawDescGZIP(), []int{2}
}

func (x *UpdateDynamicBindConfigRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *UpdateDynamicBindConfigRequest) GetEnvironment() string {
	if x != nil {
		return x.Environment
	}
	return ""
}

func (x *UpdateDynamicBindConfigRequest) GetRestrictedBind() *wrapperspb.BoolValue {
	if x != nil {
		return x.RestrictedBind
	}
	return nil
}

func (x *UpdateDynamicBindConfigRequest) GetBurstCountCapacity() map[string]uint32 {
	if x != nil {
		return x.BurstCountCapacity
	}
	return nil
}

type ListDynamicBindConfigsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	DbConfig []*DynamicBindConfig `protobuf:"bytes,1,rep,name=db_config,json=dbConfig,proto3" json:"db_config,omitempty"`
}

func (x *ListDynamicBindConfigsResponse) Reset() {
	*x = ListDynamicBindConfigsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_dbconfig_dbconfig_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListDynamicBindConfigsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListDynamicBindConfigsResponse) ProtoMessage() {}

func (x *ListDynamicBindConfigsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_dbconfig_dbconfig_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListDynamicBindConfigsResponse.ProtoReflect.Descriptor instead.
func (*ListDynamicBindConfigsResponse) Descriptor() ([]byte, []int) {
	return file_dbconfig_dbconfig_proto_rawDescGZIP(), []int{3}
}

func (x *ListDynamicBindConfigsResponse) GetDbConfig() []*DynamicBindConfig {
	if x != nil {
		return x.DbConfig
	}
	return nil
}

var File_dbconfig_dbconfig_proto protoreflect.FileDescriptor

var file_dbconfig_dbconfig_proto_rawDesc = []byte{
	0x0a, 0x17, 0x64, 0x62, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2f, 0x64, 0x62, 0x63, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x64, 0x62, 0x63, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x1a, 0x15, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2f, 0x67, 0x65, 0x6e,
	0x65, 0x72, 0x61, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74,
	0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x77, 0x72, 0x61, 0x70, 0x70, 0x65, 0x72,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xcc, 0x03, 0x0a, 0x11, 0x44, 0x79, 0x6e, 0x61,
	0x6d, 0x69, 0x63, 0x42, 0x69, 0x6e, 0x64, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x0e, 0x0a,
	0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x20, 0x0a,
	0x0b, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x0b, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x12,
	0x27, 0x0a, 0x0f, 0x72, 0x65, 0x73, 0x74, 0x72, 0x69, 0x63, 0x74, 0x65, 0x64, 0x5f, 0x62, 0x69,
	0x6e, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0e, 0x72, 0x65, 0x73, 0x74, 0x72, 0x69,
	0x63, 0x74, 0x65, 0x64, 0x42, 0x69, 0x6e, 0x64, 0x12, 0x32, 0x0a, 0x15, 0x72, 0x65, 0x73, 0x74,
	0x72, 0x69, 0x63, 0x74, 0x65, 0x64, 0x5f, 0x62, 0x69, 0x6e, 0x64, 0x5f, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x13, 0x72, 0x65, 0x73, 0x74, 0x72, 0x69, 0x63,
	0x74, 0x65, 0x64, 0x42, 0x69, 0x6e, 0x64, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x65, 0x0a, 0x14,
	0x62, 0x75, 0x72, 0x73, 0x74, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x5f, 0x63, 0x61, 0x70, 0x61,
	0x63, 0x69, 0x74, 0x79, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x33, 0x2e, 0x64, 0x62, 0x63,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x44, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63, 0x42, 0x69, 0x6e,
	0x64, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x42, 0x75, 0x72, 0x73, 0x74, 0x43, 0x6f, 0x75,
	0x6e, 0x74, 0x43, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74, 0x79, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52,
	0x12, 0x62, 0x75, 0x72, 0x73, 0x74, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x43, 0x61, 0x70, 0x61, 0x63,
	0x69, 0x74, 0x79, 0x12, 0x3f, 0x0a, 0x06, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x18, 0x06, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x64, 0x62, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x44,
	0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63, 0x42, 0x69, 0x6e, 0x64, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x6c, 0x61,
	0x62, 0x65, 0x6c, 0x73, 0x1a, 0x45, 0x0a, 0x17, 0x42, 0x75, 0x72, 0x73, 0x74, 0x43, 0x6f, 0x75,
	0x6e, 0x74, 0x43, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74, 0x79, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12,
	0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65,
	0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d,
	0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x39, 0x0a, 0x0b, 0x4c,
	0x61, 0x62, 0x65, 0x6c, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65,
	0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x83, 0x03, 0x0a, 0x1e, 0x43, 0x72, 0x65, 0x61, 0x74,
	0x65, 0x44, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63, 0x42, 0x69, 0x6e, 0x64, 0x43, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x17, 0x0a, 0x07, 0x73, 0x65, 0x5f,
	0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x65, 0x4e, 0x61,
	0x6d, 0x65, 0x12, 0x15, 0x0a, 0x06, 0x73, 0x65, 0x5f, 0x75, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x05, 0x73, 0x65, 0x55, 0x69, 0x64, 0x12, 0x19, 0x0a, 0x08, 0x65, 0x6e, 0x76,
	0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x65, 0x6e, 0x76,
	0x4e, 0x61, 0x6d, 0x65, 0x12, 0x72, 0x0a, 0x14, 0x62, 0x75, 0x72, 0x73, 0x74, 0x5f, 0x63, 0x6f,
	0x75, 0x6e, 0x74, 0x5f, 0x63, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74, 0x79, 0x18, 0x04, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x40, 0x2e, 0x64, 0x62, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x43, 0x72,
	0x65, 0x61, 0x74, 0x65, 0x44, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63, 0x42, 0x69, 0x6e, 0x64, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x42, 0x75, 0x72,
	0x73, 0x74, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x43, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74, 0x79, 0x45,
	0x6e, 0x74, 0x72, 0x79, 0x52, 0x12, 0x62, 0x75, 0x72, 0x73, 0x74, 0x43, 0x6f, 0x75, 0x6e, 0x74,
	0x43, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74, 0x79, 0x12, 0x27, 0x0a, 0x0f, 0x72, 0x65, 0x73, 0x74,
	0x72, 0x69, 0x63, 0x74, 0x65, 0x64, 0x5f, 0x62, 0x69, 0x6e, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x08, 0x52, 0x0e, 0x72, 0x65, 0x73, 0x74, 0x72, 0x69, 0x63, 0x74, 0x65, 0x64, 0x42, 0x69, 0x6e,
	0x64, 0x12, 0x32, 0x0a, 0x15, 0x72, 0x65, 0x73, 0x74, 0x72, 0x69, 0x63, 0x74, 0x65, 0x64, 0x5f,
	0x62, 0x69, 0x6e, 0x64, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x13, 0x72, 0x65, 0x73, 0x74, 0x72, 0x69, 0x63, 0x74, 0x65, 0x64, 0x42, 0x69, 0x6e, 0x64,
	0x56, 0x61, 0x6c, 0x75, 0x65, 0x1a, 0x45, 0x0a, 0x17, 0x42, 0x75, 0x72, 0x73, 0x74, 0x43, 0x6f,
	0x75, 0x6e, 0x74, 0x43, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74, 0x79, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b,
	0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0d, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xd2, 0x02, 0x0a,
	0x1e, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x44, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63, 0x42, 0x69,
	0x6e, 0x64, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12,
	0x20, 0x0a, 0x0b, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e,
	0x74, 0x12, 0x43, 0x0a, 0x0f, 0x72, 0x65, 0x73, 0x74, 0x72, 0x69, 0x63, 0x74, 0x65, 0x64, 0x5f,
	0x62, 0x69, 0x6e, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x42, 0x6f, 0x6f,
	0x6c, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x0e, 0x72, 0x65, 0x73, 0x74, 0x72, 0x69, 0x63, 0x74,
	0x65, 0x64, 0x42, 0x69, 0x6e, 0x64, 0x12, 0x72, 0x0a, 0x14, 0x62, 0x75, 0x72, 0x73, 0x74, 0x5f,
	0x63, 0x6f, 0x75, 0x6e, 0x74, 0x5f, 0x63, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74, 0x79, 0x18, 0x04,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x40, 0x2e, 0x64, 0x62, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e,
	0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x44, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63, 0x42, 0x69, 0x6e,
	0x64, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x42,
	0x75, 0x72, 0x73, 0x74, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x43, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74,
	0x79, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x12, 0x62, 0x75, 0x72, 0x73, 0x74, 0x43, 0x6f, 0x75,
	0x6e, 0x74, 0x43, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74, 0x79, 0x1a, 0x45, 0x0a, 0x17, 0x42, 0x75,
	0x72, 0x73, 0x74, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x43, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74, 0x79,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38,
	0x01, 0x22, 0x5a, 0x0a, 0x1e, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63,
	0x42, 0x69, 0x6e, 0x64, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x38, 0x0a, 0x09, 0x64, 0x62, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x64, 0x62, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x2e, 0x44, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63, 0x42, 0x69, 0x6e, 0x64, 0x43, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x52, 0x08, 0x64, 0x62, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x32, 0x8e, 0x04,
	0x0a, 0x14, 0x44, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63, 0x42, 0x69, 0x6e, 0x64, 0x43, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x53, 0x76, 0x63, 0x12, 0x5b, 0x0a, 0x17, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x44, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63, 0x42, 0x69, 0x6e, 0x64, 0x43, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x12, 0x28, 0x2e, 0x64, 0x62, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x43, 0x72, 0x65,
	0x61, 0x74, 0x65, 0x44, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63, 0x42, 0x69, 0x6e, 0x64, 0x43, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d,
	0x70, 0x74, 0x79, 0x12, 0x48, 0x0a, 0x14, 0x47, 0x65, 0x74, 0x44, 0x79, 0x6e, 0x61, 0x6d, 0x69,
	0x63, 0x42, 0x69, 0x6e, 0x64, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x13, 0x2e, 0x67, 0x65,
	0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x47, 0x65, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x1a, 0x1b, 0x2e, 0x64, 0x62, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x44, 0x79, 0x6e, 0x61,
	0x6d, 0x69, 0x63, 0x42, 0x69, 0x6e, 0x64, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x5b, 0x0a,
	0x17, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x44, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63, 0x42, 0x69,
	0x6e, 0x64, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x28, 0x2e, 0x64, 0x62, 0x63, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x2e, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x44, 0x79, 0x6e, 0x61, 0x6d, 0x69,
	0x63, 0x42, 0x69, 0x6e, 0x64, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x46, 0x0a, 0x17, 0x44, 0x65,
	0x6c, 0x65, 0x74, 0x65, 0x44, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63, 0x42, 0x69, 0x6e, 0x64, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x13, 0x2e, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e,
	0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x49, 0x64, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70,
	0x74, 0x79, 0x12, 0x51, 0x0a, 0x21, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x43, 0x6f, 0x6c, 0x6c,
	0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x44, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63, 0x42, 0x69, 0x6e,
	0x64, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x14, 0x2e, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61,
	0x6c, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x1a, 0x16, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x57, 0x0a, 0x15, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x79, 0x6e,
	0x61, 0x6d, 0x69, 0x63, 0x42, 0x69, 0x6e, 0x64, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x14,
	0x2e, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x4f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x1a, 0x28, 0x2e, 0x64, 0x62, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e,
	0x4c, 0x69, 0x73, 0x74, 0x44, 0x79, 0x6e, 0x61, 0x6d, 0x69, 0x63, 0x42, 0x69, 0x6e, 0x64, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x33,
	0x5a, 0x31, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x68, 0x6f, 0x62,
	0x62, 0x79, 0x66, 0x61, 0x72, 0x6d, 0x2f, 0x67, 0x61, 0x72, 0x67, 0x61, 0x6e, 0x74, 0x75, 0x61,
	0x2f, 0x76, 0x33, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2f, 0x64, 0x62, 0x63, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_dbconfig_dbconfig_proto_rawDescOnce sync.Once
	file_dbconfig_dbconfig_proto_rawDescData = file_dbconfig_dbconfig_proto_rawDesc
)

func file_dbconfig_dbconfig_proto_rawDescGZIP() []byte {
	file_dbconfig_dbconfig_proto_rawDescOnce.Do(func() {
		file_dbconfig_dbconfig_proto_rawDescData = protoimpl.X.CompressGZIP(file_dbconfig_dbconfig_proto_rawDescData)
	})
	return file_dbconfig_dbconfig_proto_rawDescData
}

var file_dbconfig_dbconfig_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_dbconfig_dbconfig_proto_goTypes = []interface{}{
	(*DynamicBindConfig)(nil),              // 0: dbconfig.DynamicBindConfig
	(*CreateDynamicBindConfigRequest)(nil), // 1: dbconfig.CreateDynamicBindConfigRequest
	(*UpdateDynamicBindConfigRequest)(nil), // 2: dbconfig.UpdateDynamicBindConfigRequest
	(*ListDynamicBindConfigsResponse)(nil), // 3: dbconfig.ListDynamicBindConfigsResponse
	nil,                                    // 4: dbconfig.DynamicBindConfig.BurstCountCapacityEntry
	nil,                                    // 5: dbconfig.DynamicBindConfig.LabelsEntry
	nil,                                    // 6: dbconfig.CreateDynamicBindConfigRequest.BurstCountCapacityEntry
	nil,                                    // 7: dbconfig.UpdateDynamicBindConfigRequest.BurstCountCapacityEntry
	(*wrapperspb.BoolValue)(nil),           // 8: google.protobuf.BoolValue
	(*general.GetRequest)(nil),             // 9: general.GetRequest
	(*general.ResourceId)(nil),             // 10: general.ResourceId
	(*general.ListOptions)(nil),            // 11: general.ListOptions
	(*emptypb.Empty)(nil),                  // 12: google.protobuf.Empty
}
var file_dbconfig_dbconfig_proto_depIdxs = []int32{
	4,  // 0: dbconfig.DynamicBindConfig.burst_count_capacity:type_name -> dbconfig.DynamicBindConfig.BurstCountCapacityEntry
	5,  // 1: dbconfig.DynamicBindConfig.labels:type_name -> dbconfig.DynamicBindConfig.LabelsEntry
	6,  // 2: dbconfig.CreateDynamicBindConfigRequest.burst_count_capacity:type_name -> dbconfig.CreateDynamicBindConfigRequest.BurstCountCapacityEntry
	8,  // 3: dbconfig.UpdateDynamicBindConfigRequest.restricted_bind:type_name -> google.protobuf.BoolValue
	7,  // 4: dbconfig.UpdateDynamicBindConfigRequest.burst_count_capacity:type_name -> dbconfig.UpdateDynamicBindConfigRequest.BurstCountCapacityEntry
	0,  // 5: dbconfig.ListDynamicBindConfigsResponse.db_config:type_name -> dbconfig.DynamicBindConfig
	1,  // 6: dbconfig.DynamicBindConfigSvc.CreateDynamicBindConfig:input_type -> dbconfig.CreateDynamicBindConfigRequest
	9,  // 7: dbconfig.DynamicBindConfigSvc.GetDynamicBindConfig:input_type -> general.GetRequest
	2,  // 8: dbconfig.DynamicBindConfigSvc.UpdateDynamicBindConfig:input_type -> dbconfig.UpdateDynamicBindConfigRequest
	10, // 9: dbconfig.DynamicBindConfigSvc.DeleteDynamicBindConfig:input_type -> general.ResourceId
	11, // 10: dbconfig.DynamicBindConfigSvc.DeleteCollectionDynamicBindConfig:input_type -> general.ListOptions
	11, // 11: dbconfig.DynamicBindConfigSvc.ListDynamicBindConfig:input_type -> general.ListOptions
	12, // 12: dbconfig.DynamicBindConfigSvc.CreateDynamicBindConfig:output_type -> google.protobuf.Empty
	0,  // 13: dbconfig.DynamicBindConfigSvc.GetDynamicBindConfig:output_type -> dbconfig.DynamicBindConfig
	12, // 14: dbconfig.DynamicBindConfigSvc.UpdateDynamicBindConfig:output_type -> google.protobuf.Empty
	12, // 15: dbconfig.DynamicBindConfigSvc.DeleteDynamicBindConfig:output_type -> google.protobuf.Empty
	12, // 16: dbconfig.DynamicBindConfigSvc.DeleteCollectionDynamicBindConfig:output_type -> google.protobuf.Empty
	3,  // 17: dbconfig.DynamicBindConfigSvc.ListDynamicBindConfig:output_type -> dbconfig.ListDynamicBindConfigsResponse
	12, // [12:18] is the sub-list for method output_type
	6,  // [6:12] is the sub-list for method input_type
	6,  // [6:6] is the sub-list for extension type_name
	6,  // [6:6] is the sub-list for extension extendee
	0,  // [0:6] is the sub-list for field type_name
}

func init() { file_dbconfig_dbconfig_proto_init() }
func file_dbconfig_dbconfig_proto_init() {
	if File_dbconfig_dbconfig_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_dbconfig_dbconfig_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DynamicBindConfig); i {
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
		file_dbconfig_dbconfig_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CreateDynamicBindConfigRequest); i {
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
		file_dbconfig_dbconfig_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UpdateDynamicBindConfigRequest); i {
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
		file_dbconfig_dbconfig_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListDynamicBindConfigsResponse); i {
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
			RawDescriptor: file_dbconfig_dbconfig_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_dbconfig_dbconfig_proto_goTypes,
		DependencyIndexes: file_dbconfig_dbconfig_proto_depIdxs,
		MessageInfos:      file_dbconfig_dbconfig_proto_msgTypes,
	}.Build()
	File_dbconfig_dbconfig_proto = out.File
	file_dbconfig_dbconfig_proto_rawDesc = nil
	file_dbconfig_dbconfig_proto_goTypes = nil
	file_dbconfig_dbconfig_proto_depIdxs = nil
}
