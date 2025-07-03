package mtproto

import (
	protocol "github.com/v2fly/v2ray-core/v4/common/protocol"
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

type Account struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Secret        []byte                 `protobuf:"bytes,1,opt,name=secret,proto3" json:"secret,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Account) Reset() {
	*x = Account{}
	mi := &file_proxy_mtproto_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Account) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Account) ProtoMessage() {}

func (x *Account) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_mtproto_config_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Account.ProtoReflect.Descriptor instead.
func (*Account) Descriptor() ([]byte, []int) {
	return file_proxy_mtproto_config_proto_rawDescGZIP(), []int{0}
}

func (x *Account) GetSecret() []byte {
	if x != nil {
		return x.Secret
	}
	return nil
}

type ServerConfig struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// User is a list of users that allowed to connect to this inbound.
	// Although this is a repeated field, only the first user is effective for
	// now.
	User          []*protocol.User `protobuf:"bytes,1,rep,name=user,proto3" json:"user,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ServerConfig) Reset() {
	*x = ServerConfig{}
	mi := &file_proxy_mtproto_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ServerConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServerConfig) ProtoMessage() {}

func (x *ServerConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_mtproto_config_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ServerConfig.ProtoReflect.Descriptor instead.
func (*ServerConfig) Descriptor() ([]byte, []int) {
	return file_proxy_mtproto_config_proto_rawDescGZIP(), []int{1}
}

func (x *ServerConfig) GetUser() []*protocol.User {
	if x != nil {
		return x.User
	}
	return nil
}

type ClientConfig struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ClientConfig) Reset() {
	*x = ClientConfig{}
	mi := &file_proxy_mtproto_config_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ClientConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClientConfig) ProtoMessage() {}

func (x *ClientConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_mtproto_config_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ClientConfig.ProtoReflect.Descriptor instead.
func (*ClientConfig) Descriptor() ([]byte, []int) {
	return file_proxy_mtproto_config_proto_rawDescGZIP(), []int{2}
}

var File_proxy_mtproto_config_proto protoreflect.FileDescriptor

const file_proxy_mtproto_config_proto_rawDesc = "" +
	"\n" +
	"\x1aproxy/mtproto/config.proto\x12\x18v2ray.core.proxy.mtproto\x1a\x1acommon/protocol/user.proto\"!\n" +
	"\aAccount\x12\x16\n" +
	"\x06secret\x18\x01 \x01(\fR\x06secret\"D\n" +
	"\fServerConfig\x124\n" +
	"\x04user\x18\x01 \x03(\v2 .v2ray.core.common.protocol.UserR\x04user\"\x0e\n" +
	"\fClientConfigBi\n" +
	"\x1ccom.v2ray.core.proxy.mtprotoP\x01Z,github.com/v2fly/v2ray-core/v4/proxy/mtproto\xaa\x02\x18V2Ray.Core.Proxy.Mtprotob\x06proto3"

var (
	file_proxy_mtproto_config_proto_rawDescOnce sync.Once
	file_proxy_mtproto_config_proto_rawDescData []byte
)

func file_proxy_mtproto_config_proto_rawDescGZIP() []byte {
	file_proxy_mtproto_config_proto_rawDescOnce.Do(func() {
		file_proxy_mtproto_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proxy_mtproto_config_proto_rawDesc), len(file_proxy_mtproto_config_proto_rawDesc)))
	})
	return file_proxy_mtproto_config_proto_rawDescData
}

var file_proxy_mtproto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_proxy_mtproto_config_proto_goTypes = []any{
	(*Account)(nil),       // 0: v2ray.core.proxy.mtproto.Account
	(*ServerConfig)(nil),  // 1: v2ray.core.proxy.mtproto.ServerConfig
	(*ClientConfig)(nil),  // 2: v2ray.core.proxy.mtproto.ClientConfig
	(*protocol.User)(nil), // 3: v2ray.core.common.protocol.User
}
var file_proxy_mtproto_config_proto_depIdxs = []int32{
	3, // 0: v2ray.core.proxy.mtproto.ServerConfig.user:type_name -> v2ray.core.common.protocol.User
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_proxy_mtproto_config_proto_init() }
func file_proxy_mtproto_config_proto_init() {
	if File_proxy_mtproto_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proxy_mtproto_config_proto_rawDesc), len(file_proxy_mtproto_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proxy_mtproto_config_proto_goTypes,
		DependencyIndexes: file_proxy_mtproto_config_proto_depIdxs,
		MessageInfos:      file_proxy_mtproto_config_proto_msgTypes,
	}.Build()
	File_proxy_mtproto_config_proto = out.File
	file_proxy_mtproto_config_proto_goTypes = nil
	file_proxy_mtproto_config_proto_depIdxs = nil
}
