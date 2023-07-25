// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v3.21.12
// source: accesscode/accesscode.proto

package accesscode

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ResourceId struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *ResourceId) Reset() {
	*x = ResourceId{}
	if protoimpl.UnsafeEnabled {
		mi := &file_accesscode_accesscode_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ResourceId) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ResourceId) ProtoMessage() {}

func (x *ResourceId) ProtoReflect() protoreflect.Message {
	mi := &file_accesscode_accesscode_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ResourceId.ProtoReflect.Descriptor instead.
func (*ResourceId) Descriptor() ([]byte, []int) {
	return file_accesscode_accesscode_proto_rawDescGZIP(), []int{0}
}

func (x *ResourceId) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type OneTimeAccessCode struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id                string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	User              string `protobuf:"bytes,2,opt,name=user,proto3" json:"user,omitempty"`
	RedeemedTimestamp string `protobuf:"bytes,3,opt,name=redeemed_timestamp,json=redeemedTimestamp,proto3" json:"redeemed_timestamp,omitempty"`
}

func (x *OneTimeAccessCode) Reset() {
	*x = OneTimeAccessCode{}
	if protoimpl.UnsafeEnabled {
		mi := &file_accesscode_accesscode_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *OneTimeAccessCode) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OneTimeAccessCode) ProtoMessage() {}

func (x *OneTimeAccessCode) ProtoReflect() protoreflect.Message {
	mi := &file_accesscode_accesscode_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OneTimeAccessCode.ProtoReflect.Descriptor instead.
func (*OneTimeAccessCode) Descriptor() ([]byte, []int) {
	return file_accesscode_accesscode_proto_rawDescGZIP(), []int{1}
}

func (x *OneTimeAccessCode) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *OneTimeAccessCode) GetUser() string {
	if x != nil {
		return x.User
	}
	return ""
}

func (x *OneTimeAccessCode) GetRedeemedTimestamp() string {
	if x != nil {
		return x.RedeemedTimestamp
	}
	return ""
}

var File_accesscode_accesscode_proto protoreflect.FileDescriptor

var file_accesscode_accesscode_proto_rawDesc = []byte{
	0x0a, 0x1b, 0x61, 0x63, 0x63, 0x65, 0x73, 0x73, 0x63, 0x6f, 0x64, 0x65, 0x2f, 0x61, 0x63, 0x63,
	0x65, 0x73, 0x73, 0x63, 0x6f, 0x64, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0b, 0x61,
	0x63, 0x63, 0x65, 0x73, 0x73, 0x5f, 0x63, 0x6f, 0x64, 0x65, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74,
	0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x1c, 0x0a, 0x0a, 0x52, 0x65, 0x73, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x49, 0x64, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x02, 0x69, 0x64, 0x22, 0x66, 0x0a, 0x11, 0x4f, 0x6e, 0x65, 0x54, 0x69, 0x6d, 0x65,
	0x41, 0x63, 0x63, 0x65, 0x73, 0x73, 0x43, 0x6f, 0x64, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x75, 0x73,
	0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x75, 0x73, 0x65, 0x72, 0x12, 0x2d,
	0x0a, 0x12, 0x72, 0x65, 0x64, 0x65, 0x65, 0x6d, 0x65, 0x64, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x11, 0x72, 0x65, 0x64, 0x65,
	0x65, 0x6d, 0x65, 0x64, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x32, 0x99, 0x01,
	0x0a, 0x0d, 0x41, 0x63, 0x63, 0x65, 0x73, 0x73, 0x43, 0x6f, 0x64, 0x65, 0x53, 0x76, 0x63, 0x12,
	0x42, 0x0a, 0x07, 0x47, 0x65, 0x74, 0x4f, 0x74, 0x61, 0x63, 0x12, 0x17, 0x2e, 0x61, 0x63, 0x63,
	0x65, 0x73, 0x73, 0x5f, 0x63, 0x6f, 0x64, 0x65, 0x2e, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x49, 0x64, 0x1a, 0x1e, 0x2e, 0x61, 0x63, 0x63, 0x65, 0x73, 0x73, 0x5f, 0x63, 0x6f, 0x64,
	0x65, 0x2e, 0x4f, 0x6e, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x41, 0x63, 0x63, 0x65, 0x73, 0x73, 0x43,
	0x6f, 0x64, 0x65, 0x12, 0x44, 0x0a, 0x0a, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x4f, 0x74, 0x61,
	0x63, 0x12, 0x1e, 0x2e, 0x61, 0x63, 0x63, 0x65, 0x73, 0x73, 0x5f, 0x63, 0x6f, 0x64, 0x65, 0x2e,
	0x4f, 0x6e, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x41, 0x63, 0x63, 0x65, 0x73, 0x73, 0x43, 0x6f, 0x64,
	0x65, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x42, 0x32, 0x5a, 0x30, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x68, 0x6f, 0x62, 0x62, 0x79, 0x66, 0x61, 0x72,
	0x6d, 0x2f, 0x67, 0x61, 0x72, 0x67, 0x61, 0x6e, 0x74, 0x75, 0x61, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x73, 0x2f, 0x61, 0x63, 0x63, 0x65, 0x73, 0x73, 0x63, 0x6f, 0x64, 0x65, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_accesscode_accesscode_proto_rawDescOnce sync.Once
	file_accesscode_accesscode_proto_rawDescData = file_accesscode_accesscode_proto_rawDesc
)

func file_accesscode_accesscode_proto_rawDescGZIP() []byte {
	file_accesscode_accesscode_proto_rawDescOnce.Do(func() {
		file_accesscode_accesscode_proto_rawDescData = protoimpl.X.CompressGZIP(file_accesscode_accesscode_proto_rawDescData)
	})
	return file_accesscode_accesscode_proto_rawDescData
}

var file_accesscode_accesscode_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_accesscode_accesscode_proto_goTypes = []interface{}{
	(*ResourceId)(nil),        // 0: access_code.ResourceId
	(*OneTimeAccessCode)(nil), // 1: access_code.OneTimeAccessCode
	(*emptypb.Empty)(nil),     // 2: google.protobuf.Empty
}
var file_accesscode_accesscode_proto_depIdxs = []int32{
	0, // 0: access_code.AccessCodeSvc.GetOtac:input_type -> access_code.ResourceId
	1, // 1: access_code.AccessCodeSvc.UpdateOtac:input_type -> access_code.OneTimeAccessCode
	1, // 2: access_code.AccessCodeSvc.GetOtac:output_type -> access_code.OneTimeAccessCode
	2, // 3: access_code.AccessCodeSvc.UpdateOtac:output_type -> google.protobuf.Empty
	2, // [2:4] is the sub-list for method output_type
	0, // [0:2] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_accesscode_accesscode_proto_init() }
func file_accesscode_accesscode_proto_init() {
	if File_accesscode_accesscode_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_accesscode_accesscode_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ResourceId); i {
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
		file_accesscode_accesscode_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*OneTimeAccessCode); i {
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
			RawDescriptor: file_accesscode_accesscode_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_accesscode_accesscode_proto_goTypes,
		DependencyIndexes: file_accesscode_accesscode_proto_depIdxs,
		MessageInfos:      file_accesscode_accesscode_proto_msgTypes,
	}.Build()
	File_accesscode_accesscode_proto = out.File
	file_accesscode_accesscode_proto_rawDesc = nil
	file_accesscode_accesscode_proto_goTypes = nil
	file_accesscode_accesscode_proto_depIdxs = nil
}
