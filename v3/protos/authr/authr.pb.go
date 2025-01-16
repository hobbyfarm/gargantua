// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.1
// 	protoc        v3.21.12
// source: authr/authr.proto

package authrpb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type AuthRRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UserName      string                 `protobuf:"bytes,1,opt,name=userName,proto3" json:"userName,omitempty"`
	Request       *RbacRequest           `protobuf:"bytes,2,opt,name=request,proto3" json:"request,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AuthRRequest) Reset() {
	*x = AuthRRequest{}
	mi := &file_authr_authr_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AuthRRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AuthRRequest) ProtoMessage() {}

func (x *AuthRRequest) ProtoReflect() protoreflect.Message {
	mi := &file_authr_authr_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AuthRRequest.ProtoReflect.Descriptor instead.
func (*AuthRRequest) Descriptor() ([]byte, []int) {
	return file_authr_authr_proto_rawDescGZIP(), []int{0}
}

func (x *AuthRRequest) GetUserName() string {
	if x != nil {
		return x.UserName
	}
	return ""
}

func (x *AuthRRequest) GetRequest() *RbacRequest {
	if x != nil {
		return x.Request
	}
	return nil
}

type AuthRResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AuthRResponse) Reset() {
	*x = AuthRResponse{}
	mi := &file_authr_authr_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AuthRResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AuthRResponse) ProtoMessage() {}

func (x *AuthRResponse) ProtoReflect() protoreflect.Message {
	mi := &file_authr_authr_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AuthRResponse.ProtoReflect.Descriptor instead.
func (*AuthRResponse) Descriptor() ([]byte, []int) {
	return file_authr_authr_proto_rawDescGZIP(), []int{1}
}

func (x *AuthRResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

type RbacRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Operator      string                 `protobuf:"bytes,1,opt,name=operator,proto3" json:"operator,omitempty"`
	Permissions   []*Permission          `protobuf:"bytes,2,rep,name=permissions,proto3" json:"permissions,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RbacRequest) Reset() {
	*x = RbacRequest{}
	mi := &file_authr_authr_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RbacRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RbacRequest) ProtoMessage() {}

func (x *RbacRequest) ProtoReflect() protoreflect.Message {
	mi := &file_authr_authr_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RbacRequest.ProtoReflect.Descriptor instead.
func (*RbacRequest) Descriptor() ([]byte, []int) {
	return file_authr_authr_proto_rawDescGZIP(), []int{2}
}

func (x *RbacRequest) GetOperator() string {
	if x != nil {
		return x.Operator
	}
	return ""
}

func (x *RbacRequest) GetPermissions() []*Permission {
	if x != nil {
		return x.Permissions
	}
	return nil
}

type Permission struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ApiGroup      string                 `protobuf:"bytes,1,opt,name=apiGroup,proto3" json:"apiGroup,omitempty"`
	Resource      string                 `protobuf:"bytes,2,opt,name=resource,proto3" json:"resource,omitempty"`
	Verb          string                 `protobuf:"bytes,3,opt,name=verb,proto3" json:"verb,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Permission) Reset() {
	*x = Permission{}
	mi := &file_authr_authr_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Permission) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Permission) ProtoMessage() {}

func (x *Permission) ProtoReflect() protoreflect.Message {
	mi := &file_authr_authr_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Permission.ProtoReflect.Descriptor instead.
func (*Permission) Descriptor() ([]byte, []int) {
	return file_authr_authr_proto_rawDescGZIP(), []int{3}
}

func (x *Permission) GetApiGroup() string {
	if x != nil {
		return x.ApiGroup
	}
	return ""
}

func (x *Permission) GetResource() string {
	if x != nil {
		return x.Resource
	}
	return ""
}

func (x *Permission) GetVerb() string {
	if x != nil {
		return x.Verb
	}
	return ""
}

var File_authr_authr_proto protoreflect.FileDescriptor

var file_authr_authr_proto_rawDesc = []byte{
	0x0a, 0x11, 0x61, 0x75, 0x74, 0x68, 0x72, 0x2f, 0x61, 0x75, 0x74, 0x68, 0x72, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x05, 0x61, 0x75, 0x74, 0x68, 0x72, 0x22, 0x58, 0x0a, 0x0c, 0x41, 0x75,
	0x74, 0x68, 0x52, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x75, 0x73,
	0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x75, 0x73,
	0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x2c, 0x0a, 0x07, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x61, 0x75, 0x74, 0x68, 0x72, 0x2e,
	0x52, 0x62, 0x61, 0x63, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x52, 0x07, 0x72, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x22, 0x29, 0x0a, 0x0d, 0x41, 0x75, 0x74, 0x68, 0x52, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x22,
	0x5e, 0x0a, 0x0b, 0x52, 0x62, 0x61, 0x63, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1a,
	0x0a, 0x08, 0x6f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x6f, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x08, 0x6f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x6f, 0x72, 0x12, 0x33, 0x0a, 0x0b, 0x70, 0x65,
	0x72, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x11, 0x2e, 0x61, 0x75, 0x74, 0x68, 0x72, 0x2e, 0x50, 0x65, 0x72, 0x6d, 0x69, 0x73, 0x73, 0x69,
	0x6f, 0x6e, 0x52, 0x0b, 0x70, 0x65, 0x72, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x22,
	0x58, 0x0a, 0x0a, 0x50, 0x65, 0x72, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x1a, 0x0a,
	0x08, 0x61, 0x70, 0x69, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x08, 0x61, 0x70, 0x69, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x12, 0x1a, 0x0a, 0x08, 0x72, 0x65, 0x73,
	0x6f, 0x75, 0x72, 0x63, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x72, 0x65, 0x73,
	0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x76, 0x65, 0x72, 0x62, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x76, 0x65, 0x72, 0x62, 0x32, 0x3b, 0x0a, 0x05, 0x41, 0x75, 0x74,
	0x68, 0x52, 0x12, 0x32, 0x0a, 0x05, 0x41, 0x75, 0x74, 0x68, 0x52, 0x12, 0x13, 0x2e, 0x61, 0x75,
	0x74, 0x68, 0x72, 0x2e, 0x41, 0x75, 0x74, 0x68, 0x52, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x1a, 0x14, 0x2e, 0x61, 0x75, 0x74, 0x68, 0x72, 0x2e, 0x41, 0x75, 0x74, 0x68, 0x52, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x38, 0x5a, 0x36, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x68, 0x6f, 0x62, 0x62, 0x79, 0x66, 0x61, 0x72, 0x6d, 0x2f, 0x67,
	0x61, 0x72, 0x67, 0x61, 0x6e, 0x74, 0x75, 0x61, 0x2f, 0x76, 0x33, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x73, 0x2f, 0x61, 0x75, 0x74, 0x68, 0x72, 0x3b, 0x61, 0x75, 0x74, 0x68, 0x72, 0x70, 0x62,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_authr_authr_proto_rawDescOnce sync.Once
	file_authr_authr_proto_rawDescData = file_authr_authr_proto_rawDesc
)

func file_authr_authr_proto_rawDescGZIP() []byte {
	file_authr_authr_proto_rawDescOnce.Do(func() {
		file_authr_authr_proto_rawDescData = protoimpl.X.CompressGZIP(file_authr_authr_proto_rawDescData)
	})
	return file_authr_authr_proto_rawDescData
}

var file_authr_authr_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_authr_authr_proto_goTypes = []any{
	(*AuthRRequest)(nil),  // 0: authr.AuthRRequest
	(*AuthRResponse)(nil), // 1: authr.AuthRResponse
	(*RbacRequest)(nil),   // 2: authr.RbacRequest
	(*Permission)(nil),    // 3: authr.Permission
}
var file_authr_authr_proto_depIdxs = []int32{
	2, // 0: authr.AuthRRequest.request:type_name -> authr.RbacRequest
	3, // 1: authr.RbacRequest.permissions:type_name -> authr.Permission
	0, // 2: authr.AuthR.AuthR:input_type -> authr.AuthRRequest
	1, // 3: authr.AuthR.AuthR:output_type -> authr.AuthRResponse
	3, // [3:4] is the sub-list for method output_type
	2, // [2:3] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_authr_authr_proto_init() }
func file_authr_authr_proto_init() {
	if File_authr_authr_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_authr_authr_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_authr_authr_proto_goTypes,
		DependencyIndexes: file_authr_authr_proto_depIdxs,
		MessageInfos:      file_authr_authr_proto_msgTypes,
	}.Build()
	File_authr_authr_proto = out.File
	file_authr_authr_proto_rawDesc = nil
	file_authr_authr_proto_goTypes = nil
	file_authr_authr_proto_depIdxs = nil
}
