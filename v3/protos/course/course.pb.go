// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.3
// 	protoc        v3.21.12
// source: course/course.proto

package coursepb

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

type Course struct {
	state             protoimpl.MessageState `protogen:"open.v1"`
	Id                string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Uid               string                 `protobuf:"bytes,2,opt,name=uid,proto3" json:"uid,omitempty"`
	Name              string                 `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	Description       string                 `protobuf:"bytes,4,opt,name=description,proto3" json:"description,omitempty"`
	Scenarios         []string               `protobuf:"bytes,5,rep,name=scenarios,proto3" json:"scenarios,omitempty"`
	Categories        []string               `protobuf:"bytes,6,rep,name=categories,proto3" json:"categories,omitempty"`
	Vms               []*general.StringMap   `protobuf:"bytes,7,rep,name=vms,proto3" json:"vms,omitempty"`
	KeepaliveDuration string                 `protobuf:"bytes,8,opt,name=keepalive_duration,json=keepaliveDuration,proto3" json:"keepalive_duration,omitempty"`
	PauseDuration     string                 `protobuf:"bytes,9,opt,name=pause_duration,json=pauseDuration,proto3" json:"pause_duration,omitempty"`
	Pausable          bool                   `protobuf:"varint,10,opt,name=pausable,proto3" json:"pausable,omitempty"`
	KeepVm            bool                   `protobuf:"varint,11,opt,name=keep_vm,json=keepVm,proto3" json:"keep_vm,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *Course) Reset() {
	*x = Course{}
	mi := &file_course_course_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Course) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Course) ProtoMessage() {}

func (x *Course) ProtoReflect() protoreflect.Message {
	mi := &file_course_course_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Course.ProtoReflect.Descriptor instead.
func (*Course) Descriptor() ([]byte, []int) {
	return file_course_course_proto_rawDescGZIP(), []int{0}
}

func (x *Course) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Course) GetUid() string {
	if x != nil {
		return x.Uid
	}
	return ""
}

func (x *Course) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Course) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *Course) GetScenarios() []string {
	if x != nil {
		return x.Scenarios
	}
	return nil
}

func (x *Course) GetCategories() []string {
	if x != nil {
		return x.Categories
	}
	return nil
}

func (x *Course) GetVms() []*general.StringMap {
	if x != nil {
		return x.Vms
	}
	return nil
}

func (x *Course) GetKeepaliveDuration() string {
	if x != nil {
		return x.KeepaliveDuration
	}
	return ""
}

func (x *Course) GetPauseDuration() string {
	if x != nil {
		return x.PauseDuration
	}
	return ""
}

func (x *Course) GetPausable() bool {
	if x != nil {
		return x.Pausable
	}
	return false
}

func (x *Course) GetKeepVm() bool {
	if x != nil {
		return x.KeepVm
	}
	return false
}

type CreateCourseRequest struct {
	state             protoimpl.MessageState `protogen:"open.v1"`
	Name              string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Description       string                 `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
	RawScenarios      string                 `protobuf:"bytes,3,opt,name=raw_scenarios,json=rawScenarios,proto3" json:"raw_scenarios,omitempty"`
	RawCategories     string                 `protobuf:"bytes,4,opt,name=raw_categories,json=rawCategories,proto3" json:"raw_categories,omitempty"`
	RawVms            string                 `protobuf:"bytes,5,opt,name=raw_vms,json=rawVms,proto3" json:"raw_vms,omitempty"`
	KeepaliveDuration string                 `protobuf:"bytes,6,opt,name=keepalive_duration,json=keepaliveDuration,proto3" json:"keepalive_duration,omitempty"`
	PauseDuration     string                 `protobuf:"bytes,7,opt,name=pause_duration,json=pauseDuration,proto3" json:"pause_duration,omitempty"`
	Pausable          bool                   `protobuf:"varint,8,opt,name=pausable,proto3" json:"pausable,omitempty"`
	KeepVm            bool                   `protobuf:"varint,9,opt,name=keep_vm,json=keepVm,proto3" json:"keep_vm,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *CreateCourseRequest) Reset() {
	*x = CreateCourseRequest{}
	mi := &file_course_course_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateCourseRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateCourseRequest) ProtoMessage() {}

func (x *CreateCourseRequest) ProtoReflect() protoreflect.Message {
	mi := &file_course_course_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateCourseRequest.ProtoReflect.Descriptor instead.
func (*CreateCourseRequest) Descriptor() ([]byte, []int) {
	return file_course_course_proto_rawDescGZIP(), []int{1}
}

func (x *CreateCourseRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *CreateCourseRequest) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *CreateCourseRequest) GetRawScenarios() string {
	if x != nil {
		return x.RawScenarios
	}
	return ""
}

func (x *CreateCourseRequest) GetRawCategories() string {
	if x != nil {
		return x.RawCategories
	}
	return ""
}

func (x *CreateCourseRequest) GetRawVms() string {
	if x != nil {
		return x.RawVms
	}
	return ""
}

func (x *CreateCourseRequest) GetKeepaliveDuration() string {
	if x != nil {
		return x.KeepaliveDuration
	}
	return ""
}

func (x *CreateCourseRequest) GetPauseDuration() string {
	if x != nil {
		return x.PauseDuration
	}
	return ""
}

func (x *CreateCourseRequest) GetPausable() bool {
	if x != nil {
		return x.Pausable
	}
	return false
}

func (x *CreateCourseRequest) GetKeepVm() bool {
	if x != nil {
		return x.KeepVm
	}
	return false
}

type UpdateCourseRequest struct {
	state             protoimpl.MessageState  `protogen:"open.v1"`
	Id                string                  `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Name              string                  `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Description       string                  `protobuf:"bytes,3,opt,name=description,proto3" json:"description,omitempty"`
	RawScenarios      string                  `protobuf:"bytes,4,opt,name=raw_scenarios,json=rawScenarios,proto3" json:"raw_scenarios,omitempty"`
	RawCategories     string                  `protobuf:"bytes,5,opt,name=raw_categories,json=rawCategories,proto3" json:"raw_categories,omitempty"`
	RawVms            string                  `protobuf:"bytes,6,opt,name=raw_vms,json=rawVms,proto3" json:"raw_vms,omitempty"`
	KeepaliveDuration *wrapperspb.StringValue `protobuf:"bytes,7,opt,name=keepalive_duration,json=keepaliveDuration,proto3" json:"keepalive_duration,omitempty"`
	PauseDuration     *wrapperspb.StringValue `protobuf:"bytes,8,opt,name=pause_duration,json=pauseDuration,proto3" json:"pause_duration,omitempty"`
	Pausable          *wrapperspb.BoolValue   `protobuf:"bytes,9,opt,name=pausable,proto3" json:"pausable,omitempty"`
	KeepVm            *wrapperspb.BoolValue   `protobuf:"bytes,10,opt,name=keep_vm,json=keepVm,proto3" json:"keep_vm,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *UpdateCourseRequest) Reset() {
	*x = UpdateCourseRequest{}
	mi := &file_course_course_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UpdateCourseRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateCourseRequest) ProtoMessage() {}

func (x *UpdateCourseRequest) ProtoReflect() protoreflect.Message {
	mi := &file_course_course_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateCourseRequest.ProtoReflect.Descriptor instead.
func (*UpdateCourseRequest) Descriptor() ([]byte, []int) {
	return file_course_course_proto_rawDescGZIP(), []int{2}
}

func (x *UpdateCourseRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *UpdateCourseRequest) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *UpdateCourseRequest) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *UpdateCourseRequest) GetRawScenarios() string {
	if x != nil {
		return x.RawScenarios
	}
	return ""
}

func (x *UpdateCourseRequest) GetRawCategories() string {
	if x != nil {
		return x.RawCategories
	}
	return ""
}

func (x *UpdateCourseRequest) GetRawVms() string {
	if x != nil {
		return x.RawVms
	}
	return ""
}

func (x *UpdateCourseRequest) GetKeepaliveDuration() *wrapperspb.StringValue {
	if x != nil {
		return x.KeepaliveDuration
	}
	return nil
}

func (x *UpdateCourseRequest) GetPauseDuration() *wrapperspb.StringValue {
	if x != nil {
		return x.PauseDuration
	}
	return nil
}

func (x *UpdateCourseRequest) GetPausable() *wrapperspb.BoolValue {
	if x != nil {
		return x.Pausable
	}
	return nil
}

func (x *UpdateCourseRequest) GetKeepVm() *wrapperspb.BoolValue {
	if x != nil {
		return x.KeepVm
	}
	return nil
}

type ListCoursesResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Courses       []*Course              `protobuf:"bytes,1,rep,name=courses,proto3" json:"courses,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListCoursesResponse) Reset() {
	*x = ListCoursesResponse{}
	mi := &file_course_course_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListCoursesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListCoursesResponse) ProtoMessage() {}

func (x *ListCoursesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_course_course_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListCoursesResponse.ProtoReflect.Descriptor instead.
func (*ListCoursesResponse) Descriptor() ([]byte, []int) {
	return file_course_course_proto_rawDescGZIP(), []int{3}
}

func (x *ListCoursesResponse) GetCourses() []*Course {
	if x != nil {
		return x.Courses
	}
	return nil
}

var File_course_course_proto protoreflect.FileDescriptor

var file_course_course_proto_rawDesc = []byte{
	0x0a, 0x13, 0x63, 0x6f, 0x75, 0x72, 0x73, 0x65, 0x2f, 0x63, 0x6f, 0x75, 0x72, 0x73, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x06, 0x63, 0x6f, 0x75, 0x72, 0x73, 0x65, 0x1a, 0x15, 0x67,
	0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2f, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2f, 0x77, 0x72, 0x61, 0x70, 0x70, 0x65, 0x72, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0xcf, 0x02, 0x0a, 0x06, 0x43, 0x6f, 0x75, 0x72, 0x73, 0x65, 0x12, 0x0e, 0x0a, 0x02,
	0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x10, 0x0a, 0x03,
	0x75, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x75, 0x69, 0x64, 0x12, 0x12,
	0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x12, 0x20, 0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x63, 0x65, 0x6e, 0x61, 0x72, 0x69, 0x6f,
	0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x09, 0x52, 0x09, 0x73, 0x63, 0x65, 0x6e, 0x61, 0x72, 0x69,
	0x6f, 0x73, 0x12, 0x1e, 0x0a, 0x0a, 0x63, 0x61, 0x74, 0x65, 0x67, 0x6f, 0x72, 0x69, 0x65, 0x73,
	0x18, 0x06, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0a, 0x63, 0x61, 0x74, 0x65, 0x67, 0x6f, 0x72, 0x69,
	0x65, 0x73, 0x12, 0x24, 0x0a, 0x03, 0x76, 0x6d, 0x73, 0x18, 0x07, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x12, 0x2e, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67,
	0x4d, 0x61, 0x70, 0x52, 0x03, 0x76, 0x6d, 0x73, 0x12, 0x2d, 0x0a, 0x12, 0x6b, 0x65, 0x65, 0x70,
	0x61, 0x6c, 0x69, 0x76, 0x65, 0x5f, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x08,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x11, 0x6b, 0x65, 0x65, 0x70, 0x61, 0x6c, 0x69, 0x76, 0x65, 0x44,
	0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x25, 0x0a, 0x0e, 0x70, 0x61, 0x75, 0x73, 0x65,
	0x5f, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x09, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0d, 0x70, 0x61, 0x75, 0x73, 0x65, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1a,
	0x0a, 0x08, 0x70, 0x61, 0x75, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x08,
	0x52, 0x08, 0x70, 0x61, 0x75, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x12, 0x17, 0x0a, 0x07, 0x6b, 0x65,
	0x65, 0x70, 0x5f, 0x76, 0x6d, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x08, 0x52, 0x06, 0x6b, 0x65, 0x65,
	0x70, 0x56, 0x6d, 0x22, 0xbb, 0x02, 0x0a, 0x13, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x43, 0x6f,
	0x75, 0x72, 0x73, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12,
	0x20, 0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x12, 0x23, 0x0a, 0x0d, 0x72, 0x61, 0x77, 0x5f, 0x73, 0x63, 0x65, 0x6e, 0x61, 0x72, 0x69,
	0x6f, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x72, 0x61, 0x77, 0x53, 0x63, 0x65,
	0x6e, 0x61, 0x72, 0x69, 0x6f, 0x73, 0x12, 0x25, 0x0a, 0x0e, 0x72, 0x61, 0x77, 0x5f, 0x63, 0x61,
	0x74, 0x65, 0x67, 0x6f, 0x72, 0x69, 0x65, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d,
	0x72, 0x61, 0x77, 0x43, 0x61, 0x74, 0x65, 0x67, 0x6f, 0x72, 0x69, 0x65, 0x73, 0x12, 0x17, 0x0a,
	0x07, 0x72, 0x61, 0x77, 0x5f, 0x76, 0x6d, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06,
	0x72, 0x61, 0x77, 0x56, 0x6d, 0x73, 0x12, 0x2d, 0x0a, 0x12, 0x6b, 0x65, 0x65, 0x70, 0x61, 0x6c,
	0x69, 0x76, 0x65, 0x5f, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x06, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x11, 0x6b, 0x65, 0x65, 0x70, 0x61, 0x6c, 0x69, 0x76, 0x65, 0x44, 0x75, 0x72,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x25, 0x0a, 0x0e, 0x70, 0x61, 0x75, 0x73, 0x65, 0x5f, 0x64,
	0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x70,
	0x61, 0x75, 0x73, 0x65, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1a, 0x0a, 0x08,
	0x70, 0x61, 0x75, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x18, 0x08, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08,
	0x70, 0x61, 0x75, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x12, 0x17, 0x0a, 0x07, 0x6b, 0x65, 0x65, 0x70,
	0x5f, 0x76, 0x6d, 0x18, 0x09, 0x20, 0x01, 0x28, 0x08, 0x52, 0x06, 0x6b, 0x65, 0x65, 0x70, 0x56,
	0x6d, 0x22, 0xbf, 0x03, 0x0a, 0x13, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x43, 0x6f, 0x75, 0x72,
	0x73, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x20, 0x0a,
	0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12,
	0x23, 0x0a, 0x0d, 0x72, 0x61, 0x77, 0x5f, 0x73, 0x63, 0x65, 0x6e, 0x61, 0x72, 0x69, 0x6f, 0x73,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x72, 0x61, 0x77, 0x53, 0x63, 0x65, 0x6e, 0x61,
	0x72, 0x69, 0x6f, 0x73, 0x12, 0x25, 0x0a, 0x0e, 0x72, 0x61, 0x77, 0x5f, 0x63, 0x61, 0x74, 0x65,
	0x67, 0x6f, 0x72, 0x69, 0x65, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x72, 0x61,
	0x77, 0x43, 0x61, 0x74, 0x65, 0x67, 0x6f, 0x72, 0x69, 0x65, 0x73, 0x12, 0x17, 0x0a, 0x07, 0x72,
	0x61, 0x77, 0x5f, 0x76, 0x6d, 0x73, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72, 0x61,
	0x77, 0x56, 0x6d, 0x73, 0x12, 0x4b, 0x0a, 0x12, 0x6b, 0x65, 0x65, 0x70, 0x61, 0x6c, 0x69, 0x76,
	0x65, 0x5f, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x11,
	0x6b, 0x65, 0x65, 0x70, 0x61, 0x6c, 0x69, 0x76, 0x65, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x12, 0x43, 0x0a, 0x0e, 0x70, 0x61, 0x75, 0x73, 0x65, 0x5f, 0x64, 0x75, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69,
	0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x0d, 0x70, 0x61, 0x75, 0x73, 0x65, 0x44, 0x75,
	0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x36, 0x0a, 0x08, 0x70, 0x61, 0x75, 0x73, 0x61, 0x62,
	0x6c, 0x65, 0x18, 0x09, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x42, 0x6f, 0x6f, 0x6c, 0x56,
	0x61, 0x6c, 0x75, 0x65, 0x52, 0x08, 0x70, 0x61, 0x75, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x12, 0x33,
	0x0a, 0x07, 0x6b, 0x65, 0x65, 0x70, 0x5f, 0x76, 0x6d, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x42, 0x6f, 0x6f, 0x6c, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x06, 0x6b, 0x65, 0x65,
	0x70, 0x56, 0x6d, 0x22, 0x3f, 0x0a, 0x13, 0x4c, 0x69, 0x73, 0x74, 0x43, 0x6f, 0x75, 0x72, 0x73,
	0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x28, 0x0a, 0x07, 0x63, 0x6f,
	0x75, 0x72, 0x73, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x63, 0x6f,
	0x75, 0x72, 0x73, 0x65, 0x2e, 0x43, 0x6f, 0x75, 0x72, 0x73, 0x65, 0x52, 0x07, 0x63, 0x6f, 0x75,
	0x72, 0x73, 0x65, 0x73, 0x32, 0x8a, 0x03, 0x0a, 0x09, 0x43, 0x6f, 0x75, 0x72, 0x73, 0x65, 0x53,
	0x76, 0x63, 0x12, 0x40, 0x0a, 0x0c, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x43, 0x6f, 0x75, 0x72,
	0x73, 0x65, 0x12, 0x1b, 0x2e, 0x63, 0x6f, 0x75, 0x72, 0x73, 0x65, 0x2e, 0x43, 0x72, 0x65, 0x61,
	0x74, 0x65, 0x43, 0x6f, 0x75, 0x72, 0x73, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a,
	0x13, 0x2e, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72,
	0x63, 0x65, 0x49, 0x64, 0x12, 0x30, 0x0a, 0x09, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x75, 0x72, 0x73,
	0x65, 0x12, 0x13, 0x2e, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x47, 0x65, 0x74, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x0e, 0x2e, 0x63, 0x6f, 0x75, 0x72, 0x73, 0x65, 0x2e,
	0x43, 0x6f, 0x75, 0x72, 0x73, 0x65, 0x12, 0x43, 0x0a, 0x0c, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65,
	0x43, 0x6f, 0x75, 0x72, 0x73, 0x65, 0x12, 0x1b, 0x2e, 0x63, 0x6f, 0x75, 0x72, 0x73, 0x65, 0x2e,
	0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x43, 0x6f, 0x75, 0x72, 0x73, 0x65, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x3b, 0x0a, 0x0c, 0x44,
	0x65, 0x6c, 0x65, 0x74, 0x65, 0x43, 0x6f, 0x75, 0x72, 0x73, 0x65, 0x12, 0x13, 0x2e, 0x67, 0x65,
	0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x49, 0x64,
	0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x46, 0x0a, 0x16, 0x44, 0x65, 0x6c, 0x65,
	0x74, 0x65, 0x43, 0x6f, 0x6c, 0x6c, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x75, 0x72,
	0x73, 0x65, 0x12, 0x14, 0x2e, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x4c, 0x69, 0x73,
	0x74, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79,
	0x12, 0x3f, 0x0a, 0x0a, 0x4c, 0x69, 0x73, 0x74, 0x43, 0x6f, 0x75, 0x72, 0x73, 0x65, 0x12, 0x14,
	0x2e, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x4f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x1a, 0x1b, 0x2e, 0x63, 0x6f, 0x75, 0x72, 0x73, 0x65, 0x2e, 0x4c, 0x69,
	0x73, 0x74, 0x43, 0x6f, 0x75, 0x72, 0x73, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x42, 0x3a, 0x5a, 0x38, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x68, 0x6f, 0x62, 0x62, 0x79, 0x66, 0x61, 0x72, 0x6d, 0x2f, 0x67, 0x61, 0x72, 0x67, 0x61, 0x6e,
	0x74, 0x75, 0x61, 0x2f, 0x76, 0x33, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2f, 0x63, 0x6f,
	0x75, 0x72, 0x73, 0x65, 0x3b, 0x63, 0x6f, 0x75, 0x72, 0x73, 0x65, 0x70, 0x62, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_course_course_proto_rawDescOnce sync.Once
	file_course_course_proto_rawDescData = file_course_course_proto_rawDesc
)

func file_course_course_proto_rawDescGZIP() []byte {
	file_course_course_proto_rawDescOnce.Do(func() {
		file_course_course_proto_rawDescData = protoimpl.X.CompressGZIP(file_course_course_proto_rawDescData)
	})
	return file_course_course_proto_rawDescData
}

var file_course_course_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_course_course_proto_goTypes = []any{
	(*Course)(nil),                 // 0: course.Course
	(*CreateCourseRequest)(nil),    // 1: course.CreateCourseRequest
	(*UpdateCourseRequest)(nil),    // 2: course.UpdateCourseRequest
	(*ListCoursesResponse)(nil),    // 3: course.ListCoursesResponse
	(*general.StringMap)(nil),      // 4: general.StringMap
	(*wrapperspb.StringValue)(nil), // 5: google.protobuf.StringValue
	(*wrapperspb.BoolValue)(nil),   // 6: google.protobuf.BoolValue
	(*general.GetRequest)(nil),     // 7: general.GetRequest
	(*general.ResourceId)(nil),     // 8: general.ResourceId
	(*general.ListOptions)(nil),    // 9: general.ListOptions
	(*emptypb.Empty)(nil),          // 10: google.protobuf.Empty
}
var file_course_course_proto_depIdxs = []int32{
	4,  // 0: course.Course.vms:type_name -> general.StringMap
	5,  // 1: course.UpdateCourseRequest.keepalive_duration:type_name -> google.protobuf.StringValue
	5,  // 2: course.UpdateCourseRequest.pause_duration:type_name -> google.protobuf.StringValue
	6,  // 3: course.UpdateCourseRequest.pausable:type_name -> google.protobuf.BoolValue
	6,  // 4: course.UpdateCourseRequest.keep_vm:type_name -> google.protobuf.BoolValue
	0,  // 5: course.ListCoursesResponse.courses:type_name -> course.Course
	1,  // 6: course.CourseSvc.CreateCourse:input_type -> course.CreateCourseRequest
	7,  // 7: course.CourseSvc.GetCourse:input_type -> general.GetRequest
	2,  // 8: course.CourseSvc.UpdateCourse:input_type -> course.UpdateCourseRequest
	8,  // 9: course.CourseSvc.DeleteCourse:input_type -> general.ResourceId
	9,  // 10: course.CourseSvc.DeleteCollectionCourse:input_type -> general.ListOptions
	9,  // 11: course.CourseSvc.ListCourse:input_type -> general.ListOptions
	8,  // 12: course.CourseSvc.CreateCourse:output_type -> general.ResourceId
	0,  // 13: course.CourseSvc.GetCourse:output_type -> course.Course
	10, // 14: course.CourseSvc.UpdateCourse:output_type -> google.protobuf.Empty
	10, // 15: course.CourseSvc.DeleteCourse:output_type -> google.protobuf.Empty
	10, // 16: course.CourseSvc.DeleteCollectionCourse:output_type -> google.protobuf.Empty
	3,  // 17: course.CourseSvc.ListCourse:output_type -> course.ListCoursesResponse
	12, // [12:18] is the sub-list for method output_type
	6,  // [6:12] is the sub-list for method input_type
	6,  // [6:6] is the sub-list for extension type_name
	6,  // [6:6] is the sub-list for extension extendee
	0,  // [0:6] is the sub-list for field type_name
}

func init() { file_course_course_proto_init() }
func file_course_course_proto_init() {
	if File_course_course_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_course_course_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_course_course_proto_goTypes,
		DependencyIndexes: file_course_course_proto_depIdxs,
		MessageInfos:      file_course_course_proto_msgTypes,
	}.Build()
	File_course_course_proto = out.File
	file_course_course_proto_rawDesc = nil
	file_course_course_proto_goTypes = nil
	file_course_course_proto_depIdxs = nil
}
