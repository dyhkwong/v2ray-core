package shadowsocks_2022

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

type ServerConfig struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Method        string                 `protobuf:"bytes,1,opt,name=method,proto3" json:"method,omitempty"`
	Key           string                 `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`
	Email         string                 `protobuf:"bytes,3,opt,name=email,proto3" json:"email,omitempty"`
	Level         int32                  `protobuf:"varint,4,opt,name=level,proto3" json:"level,omitempty"`
	Network       []net.Network          `protobuf:"varint,5,rep,packed,name=network,proto3,enum=v2ray.core.common.net.Network" json:"network,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ServerConfig) Reset() {
	*x = ServerConfig{}
	mi := &file_proxy_shadowsocks_2022_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ServerConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServerConfig) ProtoMessage() {}

func (x *ServerConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_shadowsocks_2022_config_proto_msgTypes[0]
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
	return file_proxy_shadowsocks_2022_config_proto_rawDescGZIP(), []int{0}
}

func (x *ServerConfig) GetMethod() string {
	if x != nil {
		return x.Method
	}
	return ""
}

func (x *ServerConfig) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *ServerConfig) GetEmail() string {
	if x != nil {
		return x.Email
	}
	return ""
}

func (x *ServerConfig) GetLevel() int32 {
	if x != nil {
		return x.Level
	}
	return 0
}

func (x *ServerConfig) GetNetwork() []net.Network {
	if x != nil {
		return x.Network
	}
	return nil
}

type MultiUserServerConfig struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Method        string                 `protobuf:"bytes,1,opt,name=method,proto3" json:"method,omitempty"`
	Key           string                 `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`
	Users         []*User                `protobuf:"bytes,3,rep,name=users,proto3" json:"users,omitempty"`
	Network       []net.Network          `protobuf:"varint,4,rep,packed,name=network,proto3,enum=v2ray.core.common.net.Network" json:"network,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *MultiUserServerConfig) Reset() {
	*x = MultiUserServerConfig{}
	mi := &file_proxy_shadowsocks_2022_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *MultiUserServerConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MultiUserServerConfig) ProtoMessage() {}

func (x *MultiUserServerConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_shadowsocks_2022_config_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MultiUserServerConfig.ProtoReflect.Descriptor instead.
func (*MultiUserServerConfig) Descriptor() ([]byte, []int) {
	return file_proxy_shadowsocks_2022_config_proto_rawDescGZIP(), []int{1}
}

func (x *MultiUserServerConfig) GetMethod() string {
	if x != nil {
		return x.Method
	}
	return ""
}

func (x *MultiUserServerConfig) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *MultiUserServerConfig) GetUsers() []*User {
	if x != nil {
		return x.Users
	}
	return nil
}

func (x *MultiUserServerConfig) GetNetwork() []net.Network {
	if x != nil {
		return x.Network
	}
	return nil
}

type RelayDestination struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Key           string                 `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Address       *net.IPOrDomain        `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"`
	Port          uint32                 `protobuf:"varint,3,opt,name=port,proto3" json:"port,omitempty"`
	Email         string                 `protobuf:"bytes,4,opt,name=email,proto3" json:"email,omitempty"`
	Level         int32                  `protobuf:"varint,5,opt,name=level,proto3" json:"level,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RelayDestination) Reset() {
	*x = RelayDestination{}
	mi := &file_proxy_shadowsocks_2022_config_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RelayDestination) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RelayDestination) ProtoMessage() {}

func (x *RelayDestination) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_shadowsocks_2022_config_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RelayDestination.ProtoReflect.Descriptor instead.
func (*RelayDestination) Descriptor() ([]byte, []int) {
	return file_proxy_shadowsocks_2022_config_proto_rawDescGZIP(), []int{2}
}

func (x *RelayDestination) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *RelayDestination) GetAddress() *net.IPOrDomain {
	if x != nil {
		return x.Address
	}
	return nil
}

func (x *RelayDestination) GetPort() uint32 {
	if x != nil {
		return x.Port
	}
	return 0
}

func (x *RelayDestination) GetEmail() string {
	if x != nil {
		return x.Email
	}
	return ""
}

func (x *RelayDestination) GetLevel() int32 {
	if x != nil {
		return x.Level
	}
	return 0
}

type RelayServerConfig struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Method        string                 `protobuf:"bytes,1,opt,name=method,proto3" json:"method,omitempty"`
	Key           string                 `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`
	Destinations  []*RelayDestination    `protobuf:"bytes,3,rep,name=destinations,proto3" json:"destinations,omitempty"`
	Network       []net.Network          `protobuf:"varint,4,rep,packed,name=network,proto3,enum=v2ray.core.common.net.Network" json:"network,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RelayServerConfig) Reset() {
	*x = RelayServerConfig{}
	mi := &file_proxy_shadowsocks_2022_config_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RelayServerConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RelayServerConfig) ProtoMessage() {}

func (x *RelayServerConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_shadowsocks_2022_config_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RelayServerConfig.ProtoReflect.Descriptor instead.
func (*RelayServerConfig) Descriptor() ([]byte, []int) {
	return file_proxy_shadowsocks_2022_config_proto_rawDescGZIP(), []int{3}
}

func (x *RelayServerConfig) GetMethod() string {
	if x != nil {
		return x.Method
	}
	return ""
}

func (x *RelayServerConfig) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *RelayServerConfig) GetDestinations() []*RelayDestination {
	if x != nil {
		return x.Destinations
	}
	return nil
}

func (x *RelayServerConfig) GetNetwork() []net.Network {
	if x != nil {
		return x.Network
	}
	return nil
}

type User struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Key           string                 `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Email         string                 `protobuf:"bytes,2,opt,name=email,proto3" json:"email,omitempty"`
	Level         int32                  `protobuf:"varint,3,opt,name=level,proto3" json:"level,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User) Reset() {
	*x = User{}
	mi := &file_proxy_shadowsocks_2022_config_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User) ProtoMessage() {}

func (x *User) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_shadowsocks_2022_config_proto_msgTypes[4]
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
	return file_proxy_shadowsocks_2022_config_proto_rawDescGZIP(), []int{4}
}

func (x *User) GetKey() string {
	if x != nil {
		return x.Key
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

type ClientConfig struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Address       *net.IPOrDomain        `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Port          uint32                 `protobuf:"varint,2,opt,name=port,proto3" json:"port,omitempty"`
	Method        string                 `protobuf:"bytes,3,opt,name=method,proto3" json:"method,omitempty"`
	Key           string                 `protobuf:"bytes,4,opt,name=key,proto3" json:"key,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ClientConfig) Reset() {
	*x = ClientConfig{}
	mi := &file_proxy_shadowsocks_2022_config_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ClientConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClientConfig) ProtoMessage() {}

func (x *ClientConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_shadowsocks_2022_config_proto_msgTypes[5]
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
	return file_proxy_shadowsocks_2022_config_proto_rawDescGZIP(), []int{5}
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

func (x *ClientConfig) GetMethod() string {
	if x != nil {
		return x.Method
	}
	return ""
}

func (x *ClientConfig) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

var File_proxy_shadowsocks_2022_config_proto protoreflect.FileDescriptor

const file_proxy_shadowsocks_2022_config_proto_rawDesc = "" +
	"\n" +
	"#proxy/shadowsocks_2022/config.proto\x12!v2ray.core.proxy.shadowsocks_2022\x1a common/protoext/extensions.proto\x1a\x18common/net/network.proto\x1a\x18common/net/address.proto\"\xbf\x01\n" +
	"\fServerConfig\x12\x16\n" +
	"\x06method\x18\x01 \x01(\tR\x06method\x12\x10\n" +
	"\x03key\x18\x02 \x01(\tR\x03key\x12\x14\n" +
	"\x05email\x18\x03 \x01(\tR\x05email\x12\x14\n" +
	"\x05level\x18\x04 \x01(\x05R\x05level\x128\n" +
	"\anetwork\x18\x05 \x03(\x0e2\x1e.v2ray.core.common.net.NetworkR\anetwork:\x1f\x82\xb5\x18\x1b\n" +
	"\ainbound\x12\x10shadowsocks-2022\"\xe1\x01\n" +
	"\x15MultiUserServerConfig\x12\x16\n" +
	"\x06method\x18\x01 \x01(\tR\x06method\x12\x10\n" +
	"\x03key\x18\x02 \x01(\tR\x03key\x12=\n" +
	"\x05users\x18\x03 \x03(\v2'.v2ray.core.proxy.shadowsocks_2022.UserR\x05users\x128\n" +
	"\anetwork\x18\x04 \x03(\x0e2\x1e.v2ray.core.common.net.NetworkR\anetwork:%\x82\xb5\x18!\n" +
	"\ainbound\x12\x16shadowsocks-2022-multi\"\xa1\x01\n" +
	"\x10RelayDestination\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12;\n" +
	"\aaddress\x18\x02 \x01(\v2!.v2ray.core.common.net.IPOrDomainR\aaddress\x12\x12\n" +
	"\x04port\x18\x03 \x01(\rR\x04port\x12\x14\n" +
	"\x05email\x18\x04 \x01(\tR\x05email\x12\x14\n" +
	"\x05level\x18\x05 \x01(\x05R\x05level\"\xf7\x01\n" +
	"\x11RelayServerConfig\x12\x16\n" +
	"\x06method\x18\x01 \x01(\tR\x06method\x12\x10\n" +
	"\x03key\x18\x02 \x01(\tR\x03key\x12W\n" +
	"\fdestinations\x18\x03 \x03(\v23.v2ray.core.proxy.shadowsocks_2022.RelayDestinationR\fdestinations\x128\n" +
	"\anetwork\x18\x04 \x03(\x0e2\x1e.v2ray.core.common.net.NetworkR\anetwork:%\x82\xb5\x18!\n" +
	"\ainbound\x12\x16shadowsocks-2022-relay\"D\n" +
	"\x04User\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05email\x18\x02 \x01(\tR\x05email\x12\x14\n" +
	"\x05level\x18\x03 \x01(\x05R\x05level\"\xab\x01\n" +
	"\fClientConfig\x12;\n" +
	"\aaddress\x18\x01 \x01(\v2!.v2ray.core.common.net.IPOrDomainR\aaddress\x12\x12\n" +
	"\x04port\x18\x02 \x01(\rR\x04port\x12\x16\n" +
	"\x06method\x18\x03 \x01(\tR\x06method\x12\x10\n" +
	"\x03key\x18\x04 \x01(\tR\x03key: \x82\xb5\x18\x1c\n" +
	"\boutbound\x12\x10shadowsocks-2022B\x84\x01\n" +
	"%com.v2ray.core.proxy.shadowsocks_2022P\x01Z5github.com/v2fly/v2ray-core/v5/proxy/shadowsocks_2022\xaa\x02!V2Ray.Core.Proxy.Shadowsocks_2022b\x06proto3"

var (
	file_proxy_shadowsocks_2022_config_proto_rawDescOnce sync.Once
	file_proxy_shadowsocks_2022_config_proto_rawDescData []byte
)

func file_proxy_shadowsocks_2022_config_proto_rawDescGZIP() []byte {
	file_proxy_shadowsocks_2022_config_proto_rawDescOnce.Do(func() {
		file_proxy_shadowsocks_2022_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proxy_shadowsocks_2022_config_proto_rawDesc), len(file_proxy_shadowsocks_2022_config_proto_rawDesc)))
	})
	return file_proxy_shadowsocks_2022_config_proto_rawDescData
}

var file_proxy_shadowsocks_2022_config_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_proxy_shadowsocks_2022_config_proto_goTypes = []any{
	(*ServerConfig)(nil),          // 0: v2ray.core.proxy.shadowsocks_2022.ServerConfig
	(*MultiUserServerConfig)(nil), // 1: v2ray.core.proxy.shadowsocks_2022.MultiUserServerConfig
	(*RelayDestination)(nil),      // 2: v2ray.core.proxy.shadowsocks_2022.RelayDestination
	(*RelayServerConfig)(nil),     // 3: v2ray.core.proxy.shadowsocks_2022.RelayServerConfig
	(*User)(nil),                  // 4: v2ray.core.proxy.shadowsocks_2022.User
	(*ClientConfig)(nil),          // 5: v2ray.core.proxy.shadowsocks_2022.ClientConfig
	(net.Network)(0),              // 6: v2ray.core.common.net.Network
	(*net.IPOrDomain)(nil),        // 7: v2ray.core.common.net.IPOrDomain
}
var file_proxy_shadowsocks_2022_config_proto_depIdxs = []int32{
	6, // 0: v2ray.core.proxy.shadowsocks_2022.ServerConfig.network:type_name -> v2ray.core.common.net.Network
	4, // 1: v2ray.core.proxy.shadowsocks_2022.MultiUserServerConfig.users:type_name -> v2ray.core.proxy.shadowsocks_2022.User
	6, // 2: v2ray.core.proxy.shadowsocks_2022.MultiUserServerConfig.network:type_name -> v2ray.core.common.net.Network
	7, // 3: v2ray.core.proxy.shadowsocks_2022.RelayDestination.address:type_name -> v2ray.core.common.net.IPOrDomain
	2, // 4: v2ray.core.proxy.shadowsocks_2022.RelayServerConfig.destinations:type_name -> v2ray.core.proxy.shadowsocks_2022.RelayDestination
	6, // 5: v2ray.core.proxy.shadowsocks_2022.RelayServerConfig.network:type_name -> v2ray.core.common.net.Network
	7, // 6: v2ray.core.proxy.shadowsocks_2022.ClientConfig.address:type_name -> v2ray.core.common.net.IPOrDomain
	7, // [7:7] is the sub-list for method output_type
	7, // [7:7] is the sub-list for method input_type
	7, // [7:7] is the sub-list for extension type_name
	7, // [7:7] is the sub-list for extension extendee
	0, // [0:7] is the sub-list for field type_name
}

func init() { file_proxy_shadowsocks_2022_config_proto_init() }
func file_proxy_shadowsocks_2022_config_proto_init() {
	if File_proxy_shadowsocks_2022_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proxy_shadowsocks_2022_config_proto_rawDesc), len(file_proxy_shadowsocks_2022_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proxy_shadowsocks_2022_config_proto_goTypes,
		DependencyIndexes: file_proxy_shadowsocks_2022_config_proto_depIdxs,
		MessageInfos:      file_proxy_shadowsocks_2022_config_proto_msgTypes,
	}.Build()
	File_proxy_shadowsocks_2022_config_proto = out.File
	file_proxy_shadowsocks_2022_config_proto_goTypes = nil
	file_proxy_shadowsocks_2022_config_proto_depIdxs = nil
}
