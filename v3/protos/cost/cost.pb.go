// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.2
// 	protoc        v3.21.12
// source: cost/cost.proto

package costpb

import (
	general "github.com/hobbyfarm/gargantua/v3/protos/general"
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

type Cost struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	CostGroup     string                 `protobuf:"bytes,1,opt,name=cost_group,json=costGroup,proto3" json:"cost_group,omitempty"` // name of the cost group
	Total         float64                `protobuf:"fixed64,2,opt,name=total,proto3" json:"total,omitempty"`                        // total cost for all sources
	Source        []*CostSource          `protobuf:"bytes,3,rep,name=source,proto3" json:"source,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Cost) Reset() {
	*x = Cost{}
	mi := &file_cost_cost_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Cost) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Cost) ProtoMessage() {}

func (x *Cost) ProtoReflect() protoreflect.Message {
	mi := &file_cost_cost_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Cost.ProtoReflect.Descriptor instead.
func (*Cost) Descriptor() ([]byte, []int) {
	return file_cost_cost_proto_rawDescGZIP(), []int{0}
}

func (x *Cost) GetCostGroup() string {
	if x != nil {
		return x.CostGroup
	}
	return ""
}

func (x *Cost) GetTotal() float64 {
	if x != nil {
		return x.Total
	}
	return 0
}

func (x *Cost) GetSource() []*CostSource {
	if x != nil {
		return x.Source
	}
	return nil
}

type CostSource struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Kind          string                 `protobuf:"bytes,1,opt,name=kind,proto3" json:"kind,omitempty"`   // resource kind like VirtualMachine
	Cost          float64                `protobuf:"fixed64,2,opt,name=cost,proto3" json:"cost,omitempty"` // total cost for this kind
	Count         uint64                 `protobuf:"varint,3,opt,name=count,proto3" json:"count,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CostSource) Reset() {
	*x = CostSource{}
	mi := &file_cost_cost_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CostSource) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CostSource) ProtoMessage() {}

func (x *CostSource) ProtoReflect() protoreflect.Message {
	mi := &file_cost_cost_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CostSource.ProtoReflect.Descriptor instead.
func (*CostSource) Descriptor() ([]byte, []int) {
	return file_cost_cost_proto_rawDescGZIP(), []int{1}
}

func (x *CostSource) GetKind() string {
	if x != nil {
		return x.Kind
	}
	return ""
}

func (x *CostSource) GetCost() float64 {
	if x != nil {
		return x.Cost
	}
	return 0
}

func (x *CostSource) GetCount() uint64 {
	if x != nil {
		return x.Count
	}
	return 0
}

type CreateOrUpdateCostRequest struct {
	state                 protoimpl.MessageState `protogen:"open.v1"`
	CostGroup             string                 `protobuf:"bytes,1,opt,name=cost_group,json=costGroup,proto3" json:"cost_group,omitempty"`
	Kind                  string                 `protobuf:"bytes,2,opt,name=kind,proto3" json:"kind,omitempty"` // like VirtualMachine
	BasePrice             float64                `protobuf:"fixed64,3,opt,name=base_price,json=basePrice,proto3" json:"base_price,omitempty"`
	TimeUnit              string                 `protobuf:"bytes,4,opt,name=time_unit,json=timeUnit,proto3" json:"time_unit,omitempty"`
	Id                    string                 `protobuf:"bytes,5,opt,name=id,proto3" json:"id,omitempty"`                                                                             // resource id
	CreationUnixTimestamp int64                  `protobuf:"varint,6,opt,name=creation_unix_timestamp,json=creationUnixTimestamp,proto3" json:"creation_unix_timestamp,omitempty"`       // unix timestamp in seconds
	DeletionUnixTimestamp *int64                 `protobuf:"varint,7,opt,name=deletion_unix_timestamp,json=deletionUnixTimestamp,proto3,oneof" json:"deletion_unix_timestamp,omitempty"` // unix timestamp in seconds
	unknownFields         protoimpl.UnknownFields
	sizeCache             protoimpl.SizeCache
}

func (x *CreateOrUpdateCostRequest) Reset() {
	*x = CreateOrUpdateCostRequest{}
	mi := &file_cost_cost_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateOrUpdateCostRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateOrUpdateCostRequest) ProtoMessage() {}

func (x *CreateOrUpdateCostRequest) ProtoReflect() protoreflect.Message {
	mi := &file_cost_cost_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateOrUpdateCostRequest.ProtoReflect.Descriptor instead.
func (*CreateOrUpdateCostRequest) Descriptor() ([]byte, []int) {
	return file_cost_cost_proto_rawDescGZIP(), []int{2}
}

func (x *CreateOrUpdateCostRequest) GetCostGroup() string {
	if x != nil {
		return x.CostGroup
	}
	return ""
}

func (x *CreateOrUpdateCostRequest) GetKind() string {
	if x != nil {
		return x.Kind
	}
	return ""
}

func (x *CreateOrUpdateCostRequest) GetBasePrice() float64 {
	if x != nil {
		return x.BasePrice
	}
	return 0
}

func (x *CreateOrUpdateCostRequest) GetTimeUnit() string {
	if x != nil {
		return x.TimeUnit
	}
	return ""
}

func (x *CreateOrUpdateCostRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *CreateOrUpdateCostRequest) GetCreationUnixTimestamp() int64 {
	if x != nil {
		return x.CreationUnixTimestamp
	}
	return 0
}

func (x *CreateOrUpdateCostRequest) GetDeletionUnixTimestamp() int64 {
	if x != nil && x.DeletionUnixTimestamp != nil {
		return *x.DeletionUnixTimestamp
	}
	return 0
}

type ListCostsResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Costs         []*Cost                `protobuf:"bytes,1,rep,name=costs,proto3" json:"costs,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListCostsResponse) Reset() {
	*x = ListCostsResponse{}
	mi := &file_cost_cost_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListCostsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListCostsResponse) ProtoMessage() {}

func (x *ListCostsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_cost_cost_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListCostsResponse.ProtoReflect.Descriptor instead.
func (*ListCostsResponse) Descriptor() ([]byte, []int) {
	return file_cost_cost_proto_rawDescGZIP(), []int{3}
}

func (x *ListCostsResponse) GetCosts() []*Cost {
	if x != nil {
		return x.Costs
	}
	return nil
}

var File_cost_cost_proto protoreflect.FileDescriptor

var file_cost_cost_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x63, 0x6f, 0x73, 0x74, 0x2f, 0x63, 0x6f, 0x73, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x04, 0x63, 0x6f, 0x73, 0x74, 0x1a, 0x15, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c,
	0x2f, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f,
	0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x65, 0x0a, 0x04, 0x43,
	0x6f, 0x73, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x63, 0x6f, 0x73, 0x74, 0x5f, 0x67, 0x72, 0x6f, 0x75,
	0x70, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x63, 0x6f, 0x73, 0x74, 0x47, 0x72, 0x6f,
	0x75, 0x70, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x01, 0x52, 0x05, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x12, 0x28, 0x0a, 0x06, 0x73, 0x6f, 0x75, 0x72,
	0x63, 0x65, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x63, 0x6f, 0x73, 0x74, 0x2e,
	0x43, 0x6f, 0x73, 0x74, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x52, 0x06, 0x73, 0x6f, 0x75, 0x72,
	0x63, 0x65, 0x22, 0x4a, 0x0a, 0x0a, 0x43, 0x6f, 0x73, 0x74, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x12, 0x12, 0x0a, 0x04, 0x6b, 0x69, 0x6e, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04,
	0x6b, 0x69, 0x6e, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x63, 0x6f, 0x73, 0x74, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x01, 0x52, 0x04, 0x63, 0x6f, 0x73, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x63, 0x6f, 0x75, 0x6e,
	0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x22, 0xab,
	0x02, 0x0a, 0x19, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x4f, 0x72, 0x55, 0x70, 0x64, 0x61, 0x74,
	0x65, 0x43, 0x6f, 0x73, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1d, 0x0a, 0x0a,
	0x63, 0x6f, 0x73, 0x74, 0x5f, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x09, 0x63, 0x6f, 0x73, 0x74, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x12, 0x12, 0x0a, 0x04, 0x6b,
	0x69, 0x6e, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6b, 0x69, 0x6e, 0x64, 0x12,
	0x1d, 0x0a, 0x0a, 0x62, 0x61, 0x73, 0x65, 0x5f, 0x70, 0x72, 0x69, 0x63, 0x65, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x01, 0x52, 0x09, 0x62, 0x61, 0x73, 0x65, 0x50, 0x72, 0x69, 0x63, 0x65, 0x12, 0x1b,
	0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x5f, 0x75, 0x6e, 0x69, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x08, 0x74, 0x69, 0x6d, 0x65, 0x55, 0x6e, 0x69, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69,
	0x64, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x36, 0x0a, 0x17, 0x63,
	0x72, 0x65, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x75, 0x6e, 0x69, 0x78, 0x5f, 0x74, 0x69, 0x6d,
	0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x06, 0x20, 0x01, 0x28, 0x03, 0x52, 0x15, 0x63, 0x72,
	0x65, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x55, 0x6e, 0x69, 0x78, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x12, 0x3b, 0x0a, 0x17, 0x64, 0x65, 0x6c, 0x65, 0x74, 0x69, 0x6f, 0x6e, 0x5f,
	0x75, 0x6e, 0x69, 0x78, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x07,
	0x20, 0x01, 0x28, 0x03, 0x48, 0x00, 0x52, 0x15, 0x64, 0x65, 0x6c, 0x65, 0x74, 0x69, 0x6f, 0x6e,
	0x55, 0x6e, 0x69, 0x78, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x88, 0x01, 0x01,
	0x42, 0x1a, 0x0a, 0x18, 0x5f, 0x64, 0x65, 0x6c, 0x65, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x75, 0x6e,
	0x69, 0x78, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x22, 0x35, 0x0a, 0x11,
	0x4c, 0x69, 0x73, 0x74, 0x43, 0x6f, 0x73, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x20, 0x0a, 0x05, 0x63, 0x6f, 0x73, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x0a, 0x2e, 0x63, 0x6f, 0x73, 0x74, 0x2e, 0x43, 0x6f, 0x73, 0x74, 0x52, 0x05, 0x63, 0x6f,
	0x73, 0x74, 0x73, 0x32, 0xdd, 0x02, 0x0a, 0x07, 0x43, 0x6f, 0x73, 0x74, 0x53, 0x76, 0x63, 0x12,
	0x4a, 0x0a, 0x12, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x4f, 0x72, 0x55, 0x70, 0x64, 0x61, 0x74,
	0x65, 0x43, 0x6f, 0x73, 0x74, 0x12, 0x1f, 0x2e, 0x63, 0x6f, 0x73, 0x74, 0x2e, 0x43, 0x72, 0x65,
	0x61, 0x74, 0x65, 0x4f, 0x72, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x43, 0x6f, 0x73, 0x74, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x13, 0x2e, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c,
	0x2e, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x49, 0x64, 0x12, 0x31, 0x0a, 0x0e, 0x47,
	0x65, 0x74, 0x43, 0x6f, 0x73, 0x74, 0x48, 0x69, 0x73, 0x74, 0x6f, 0x72, 0x79, 0x12, 0x13, 0x2e,
	0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x47, 0x65, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x0a, 0x2e, 0x63, 0x6f, 0x73, 0x74, 0x2e, 0x43, 0x6f, 0x73, 0x74, 0x12, 0x31,
	0x0a, 0x0e, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x73, 0x74, 0x50, 0x72, 0x65, 0x73, 0x65, 0x6e, 0x74,
	0x12, 0x13, 0x2e, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x47, 0x65, 0x74, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x0a, 0x2e, 0x63, 0x6f, 0x73, 0x74, 0x2e, 0x43, 0x6f, 0x73,
	0x74, 0x12, 0x2a, 0x0a, 0x07, 0x47, 0x65, 0x74, 0x43, 0x6f, 0x73, 0x74, 0x12, 0x13, 0x2e, 0x67,
	0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x47, 0x65, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x0a, 0x2e, 0x63, 0x6f, 0x73, 0x74, 0x2e, 0x43, 0x6f, 0x73, 0x74, 0x12, 0x39, 0x0a,
	0x0a, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x43, 0x6f, 0x73, 0x74, 0x12, 0x13, 0x2e, 0x67, 0x65,
	0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x49, 0x64,
	0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x39, 0x0a, 0x08, 0x4c, 0x69, 0x73, 0x74,
	0x43, 0x6f, 0x73, 0x74, 0x12, 0x14, 0x2e, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c, 0x2e, 0x4c,
	0x69, 0x73, 0x74, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x1a, 0x17, 0x2e, 0x63, 0x6f, 0x73,
	0x74, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x43, 0x6f, 0x73, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x42, 0x36, 0x5a, 0x34, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x68, 0x6f, 0x62, 0x62, 0x79, 0x66, 0x61, 0x72, 0x6d, 0x2f, 0x67, 0x61, 0x72, 0x67,
	0x61, 0x6e, 0x74, 0x75, 0x61, 0x2f, 0x76, 0x33, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2f,
	0x63, 0x6f, 0x73, 0x74, 0x3b, 0x63, 0x6f, 0x73, 0x74, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_cost_cost_proto_rawDescOnce sync.Once
	file_cost_cost_proto_rawDescData = file_cost_cost_proto_rawDesc
)

func file_cost_cost_proto_rawDescGZIP() []byte {
	file_cost_cost_proto_rawDescOnce.Do(func() {
		file_cost_cost_proto_rawDescData = protoimpl.X.CompressGZIP(file_cost_cost_proto_rawDescData)
	})
	return file_cost_cost_proto_rawDescData
}

var file_cost_cost_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_cost_cost_proto_goTypes = []any{
	(*Cost)(nil),                      // 0: cost.Cost
	(*CostSource)(nil),                // 1: cost.CostSource
	(*CreateOrUpdateCostRequest)(nil), // 2: cost.CreateOrUpdateCostRequest
	(*ListCostsResponse)(nil),         // 3: cost.ListCostsResponse
	(*general.GetRequest)(nil),        // 4: general.GetRequest
	(*general.ResourceId)(nil),        // 5: general.ResourceId
	(*general.ListOptions)(nil),       // 6: general.ListOptions
	(*emptypb.Empty)(nil),             // 7: google.protobuf.Empty
}
var file_cost_cost_proto_depIdxs = []int32{
	1, // 0: cost.Cost.source:type_name -> cost.CostSource
	0, // 1: cost.ListCostsResponse.costs:type_name -> cost.Cost
	2, // 2: cost.CostSvc.CreateOrUpdateCost:input_type -> cost.CreateOrUpdateCostRequest
	4, // 3: cost.CostSvc.GetCostHistory:input_type -> general.GetRequest
	4, // 4: cost.CostSvc.GetCostPresent:input_type -> general.GetRequest
	4, // 5: cost.CostSvc.GetCost:input_type -> general.GetRequest
	5, // 6: cost.CostSvc.DeleteCost:input_type -> general.ResourceId
	6, // 7: cost.CostSvc.ListCost:input_type -> general.ListOptions
	5, // 8: cost.CostSvc.CreateOrUpdateCost:output_type -> general.ResourceId
	0, // 9: cost.CostSvc.GetCostHistory:output_type -> cost.Cost
	0, // 10: cost.CostSvc.GetCostPresent:output_type -> cost.Cost
	0, // 11: cost.CostSvc.GetCost:output_type -> cost.Cost
	7, // 12: cost.CostSvc.DeleteCost:output_type -> google.protobuf.Empty
	3, // 13: cost.CostSvc.ListCost:output_type -> cost.ListCostsResponse
	8, // [8:14] is the sub-list for method output_type
	2, // [2:8] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_cost_cost_proto_init() }
func file_cost_cost_proto_init() {
	if File_cost_cost_proto != nil {
		return
	}
	file_cost_cost_proto_msgTypes[2].OneofWrappers = []any{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_cost_cost_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_cost_cost_proto_goTypes,
		DependencyIndexes: file_cost_cost_proto_depIdxs,
		MessageInfos:      file_cost_cost_proto_msgTypes,
	}.Build()
	File_cost_cost_proto = out.File
	file_cost_cost_proto_rawDesc = nil
	file_cost_cost_proto_goTypes = nil
	file_cost_cost_proto_depIdxs = nil
}
