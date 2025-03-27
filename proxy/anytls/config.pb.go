package anytls

import (
	net "github.com/v2fly/v2ray-core/v5/common/net"
	_ "github.com/v2fly/v2ray-core/v5/common/protoext"
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

type ClientConfig struct {
	state                    protoimpl.MessageState `protogen:"open.v1"`
	Address                  *net.IPOrDomain        `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Port                     uint32                 `protobuf:"varint,2,opt,name=port,proto3" json:"port,omitempty"`
	Password                 string                 `protobuf:"bytes,3,opt,name=password,proto3" json:"password,omitempty"`
	IdleSessionCheckInterval int64                  `protobuf:"varint,4,opt,name=idle_session_check_interval,json=idleSessionCheckInterval,proto3" json:"idle_session_check_interval,omitempty"`
	IdleSessionTimeout       int64                  `protobuf:"varint,5,opt,name=idle_session_timeout,json=idleSessionTimeout,proto3" json:"idle_session_timeout,omitempty"`
	MinIdleSession           int64                  `protobuf:"varint,6,opt,name=min_idle_session,json=minIdleSession,proto3" json:"min_idle_session,omitempty"`
	unknownFields            protoimpl.UnknownFields
	sizeCache                protoimpl.SizeCache
}

func (x *ClientConfig) Reset() {
	*x = ClientConfig{}
	mi := &file_proxy_anytls_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ClientConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClientConfig) ProtoMessage() {}

func (x *ClientConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_anytls_config_proto_msgTypes[0]
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
	return file_proxy_anytls_config_proto_rawDescGZIP(), []int{0}
}

func (x *ClientConfig) GetAddress() *net.IPOrDomain {
	if x != nil {
		return x.Address
	}
	return nil
}

func (x *ClientConfig) GetPort() uint32 {
	if x != nil {
		return x.Port
	}
	return 0
}

func (x *ClientConfig) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

func (x *ClientConfig) GetIdleSessionCheckInterval() int64 {
	if x != nil {
		return x.IdleSessionCheckInterval
	}
	return 0
}

func (x *ClientConfig) GetIdleSessionTimeout() int64 {
	if x != nil {
		return x.IdleSessionTimeout
	}
	return 0
}

func (x *ClientConfig) GetMinIdleSession() int64 {
	if x != nil {
		return x.MinIdleSession
	}
	return 0
}

type User struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Password      string                 `protobuf:"bytes,1,opt,name=password,proto3" json:"password,omitempty"`
	Email         string                 `protobuf:"bytes,2,opt,name=email,proto3" json:"email,omitempty"`
	Level         int32                  `protobuf:"varint,3,opt,name=level,proto3" json:"level,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User) Reset() {
	*x = User{}
	mi := &file_proxy_anytls_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User) ProtoMessage() {}

func (x *User) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_anytls_config_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User.ProtoReflect.Descriptor instead.
func (*User) Descriptor() ([]byte, []int) {
	return file_proxy_anytls_config_proto_rawDescGZIP(), []int{1}
}

func (x *User) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

func (x *User) GetEmail() string {
	if x != nil {
		return x.Email
	}
	return ""
}

func (x *User) GetLevel() int32 {
	if x != nil {
		return x.Level
	}
	return 0
}

type ServerConfig struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Users         []*User                `protobuf:"bytes,1,rep,name=users,proto3" json:"users,omitempty"`
	PaddingScheme []string               `protobuf:"bytes,2,rep,name=padding_scheme,json=paddingScheme,proto3" json:"padding_scheme,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ServerConfig) Reset() {
	*x = ServerConfig{}
	mi := &file_proxy_anytls_config_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ServerConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServerConfig) ProtoMessage() {}

func (x *ServerConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_anytls_config_proto_msgTypes[2]
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
	return file_proxy_anytls_config_proto_rawDescGZIP(), []int{2}
}

func (x *ServerConfig) GetUsers() []*User {
	if x != nil {
		return x.Users
	}
	return nil
}

func (x *ServerConfig) GetPaddingScheme() []string {
	if x != nil {
		return x.PaddingScheme
	}
	return nil
}

var File_proxy_anytls_config_proto protoreflect.FileDescriptor

const file_proxy_anytls_config_proto_rawDesc = "" +
	"\n" +
	"\x19proxy/anytls/config.proto\x12\x17v2ray.core.proxy.anytls\x1a common/protoext/extensions.proto\x1a\x18common/net/address.proto\"\xae\x02\n" +
	"\fClientConfig\x12;\n" +
	"\aaddress\x18\x01 \x01(\v2!.v2ray.core.common.net.IPOrDomainR\aaddress\x12\x12\n" +
	"\x04port\x18\x02 \x01(\rR\x04port\x12\x1a\n" +
	"\bpassword\x18\x03 \x01(\tR\bpassword\x12=\n" +
	"\x1bidle_session_check_interval\x18\x04 \x01(\x03R\x18idleSessionCheckInterval\x120\n" +
	"\x14idle_session_timeout\x18\x05 \x01(\x03R\x12idleSessionTimeout\x12(\n" +
	"\x10min_idle_session\x18\x06 \x01(\x03R\x0eminIdleSession:\x16\x82\xb5\x18\x12\n" +
	"\boutbound\x12\x06anytls\"N\n" +
	"\x04User\x12\x1a\n" +
	"\bpassword\x18\x01 \x01(\tR\bpassword\x12\x14\n" +
	"\x05email\x18\x02 \x01(\tR\x05email\x12\x14\n" +
	"\x05level\x18\x03 \x01(\x05R\x05level\"\x81\x01\n" +
	"\fServerConfig\x123\n" +
	"\x05users\x18\x01 \x03(\v2\x1d.v2ray.core.proxy.anytls.UserR\x05users\x12%\n" +
	"\x0epadding_scheme\x18\x02 \x03(\tR\rpaddingScheme:\x15\x82\xb5\x18\x11\n" +
	"\ainbound\x12\x06anytlsBf\n" +
	"\x1bcom.v2ray.core.proxy.anytlsP\x01Z+github.com/v2fly/v2ray-core/v5/proxy/anytls\xaa\x02\x17V2Ray.Core.Proxy.Anytlsb\x06proto3"

var (
	file_proxy_anytls_config_proto_rawDescOnce sync.Once
	file_proxy_anytls_config_proto_rawDescData []byte
)

func file_proxy_anytls_config_proto_rawDescGZIP() []byte {
	file_proxy_anytls_config_proto_rawDescOnce.Do(func() {
		file_proxy_anytls_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proxy_anytls_config_proto_rawDesc), len(file_proxy_anytls_config_proto_rawDesc)))
	})
	return file_proxy_anytls_config_proto_rawDescData
}

var file_proxy_anytls_config_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_proxy_anytls_config_proto_goTypes = []any{
	(*ClientConfig)(nil),   // 0: v2ray.core.proxy.anytls.ClientConfig
	(*User)(nil),           // 1: v2ray.core.proxy.anytls.User
	(*ServerConfig)(nil),   // 2: v2ray.core.proxy.anytls.ServerConfig
	(*net.IPOrDomain)(nil), // 3: v2ray.core.common.net.IPOrDomain
}
var file_proxy_anytls_config_proto_depIdxs = []int32{
	3, // 0: v2ray.core.proxy.anytls.ClientConfig.address:type_name -> v2ray.core.common.net.IPOrDomain
	1, // 1: v2ray.core.proxy.anytls.ServerConfig.users:type_name -> v2ray.core.proxy.anytls.User
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_proxy_anytls_config_proto_init() }
func file_proxy_anytls_config_proto_init() {
	if File_proxy_anytls_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proxy_anytls_config_proto_rawDesc), len(file_proxy_anytls_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proxy_anytls_config_proto_goTypes,
		DependencyIndexes: file_proxy_anytls_config_proto_depIdxs,
		MessageInfos:      file_proxy_anytls_config_proto_msgTypes,
	}.Build()
	File_proxy_anytls_config_proto = out.File
	file_proxy_anytls_config_proto_goTypes = nil
	file_proxy_anytls_config_proto_depIdxs = nil
}
