package net

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// UidList represents a list of uid.
type UidList struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// The port that this range starts from.
	Uid           []uint32 `protobuf:"varint,1,rep,packed,name=uid,proto3" json:"uid,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UidList) Reset() {
	*x = UidList{}
	mi := &file_common_net_uid_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UidList) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UidList) ProtoMessage() {}

func (x *UidList) ProtoReflect() protoreflect.Message {
	mi := &file_common_net_uid_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UidList.ProtoReflect.Descriptor instead.
func (*UidList) Descriptor() ([]byte, []int) {
	return file_common_net_uid_proto_rawDescGZIP(), []int{0}
}

func (x *UidList) GetUid() []uint32 {
	if x != nil {
		return x.Uid
	}
	return nil
}

var File_common_net_uid_proto protoreflect.FileDescriptor

const file_common_net_uid_proto_rawDesc = "" +
	"\n" +
	"\x14common/net/uid.proto\x12\x15v2ray.core.common.net\"\x1b\n" +
	"\aUidList\x12\x10\n" +
	"\x03uid\x18\x01 \x03(\rR\x03uidB`\n" +
	"\x19com.v2ray.core.common.netP\x01Z)github.com/v2fly/v2ray-core/v5/common/net\xaa\x02\x15V2Ray.Core.Common.Netb\x06proto3"

var (
	file_common_net_uid_proto_rawDescOnce sync.Once
	file_common_net_uid_proto_rawDescData []byte
)

func file_common_net_uid_proto_rawDescGZIP() []byte {
	file_common_net_uid_proto_rawDescOnce.Do(func() {
		file_common_net_uid_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_common_net_uid_proto_rawDesc), len(file_common_net_uid_proto_rawDesc)))
	})
	return file_common_net_uid_proto_rawDescData
}

var file_common_net_uid_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_common_net_uid_proto_goTypes = []any{
	(*UidList)(nil), // 0: v2ray.core.common.net.UidList
}
var file_common_net_uid_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_common_net_uid_proto_init() }
func file_common_net_uid_proto_init() {
	if File_common_net_uid_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_common_net_uid_proto_rawDesc), len(file_common_net_uid_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_common_net_uid_proto_goTypes,
		DependencyIndexes: file_common_net_uid_proto_depIdxs,
		MessageInfos:      file_common_net_uid_proto_msgTypes,
	}.Build()
	File_common_net_uid_proto = out.File
	file_common_net_uid_proto_goTypes = nil
	file_common_net_uid_proto_depIdxs = nil
}
