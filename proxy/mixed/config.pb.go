package mixed

import (
	net "github.com/v2fly/v2ray-core/v5/common/net"
	packetaddr "github.com/v2fly/v2ray-core/v5/common/net/packetaddr"
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

// AuthType is the authentication type of Socks proxy.
type AuthType int32

const (
	// NO_AUTH is for anonymous authentication.
	AuthType_NO_AUTH AuthType = 0
	// PASSWORD is for username/password authentication.
	AuthType_PASSWORD AuthType = 1
)

// Enum value maps for AuthType.
var (
	AuthType_name = map[int32]string{
		0: "NO_AUTH",
		1: "PASSWORD",
	}
	AuthType_value = map[string]int32{
		"NO_AUTH":  0,
		"PASSWORD": 1,
	}
)

func (x AuthType) Enum() *AuthType {
	p := new(AuthType)
	*p = x
	return p
}

func (x AuthType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (AuthType) Descriptor() protoreflect.EnumDescriptor {
	return file_proxy_mixed_config_proto_enumTypes[0].Descriptor()
}

func (AuthType) Type() protoreflect.EnumType {
	return &file_proxy_mixed_config_proto_enumTypes[0]
}

func (x AuthType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use AuthType.Descriptor instead.
func (AuthType) EnumDescriptor() ([]byte, []int) {
	return file_proxy_mixed_config_proto_rawDescGZIP(), []int{0}
}

// Account represents a Socks/HTTP account.
type Account struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Username      string                 `protobuf:"bytes,1,opt,name=username,proto3" json:"username,omitempty"`
	Password      string                 `protobuf:"bytes,2,opt,name=password,proto3" json:"password,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Account) Reset() {
	*x = Account{}
	mi := &file_proxy_mixed_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Account) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Account) ProtoMessage() {}

func (x *Account) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_mixed_config_proto_msgTypes[0]
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
	return file_proxy_mixed_config_proto_rawDescGZIP(), []int{0}
}

func (x *Account) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *Account) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

// ServerConfig is the protobuf config for Mixed server.
type ServerConfig struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Deprecated: Marked as deprecated in proxy/mixed/config.proto.
	AuthType AuthType          `protobuf:"varint,1,opt,name=auth_type,json=authType,proto3,enum=v2ray.core.proxy.mixed.AuthType" json:"auth_type,omitempty"`
	Accounts map[string]string `protobuf:"bytes,2,rep,name=accounts,proto3" json:"accounts,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// Deprecated: Marked as deprecated in proxy/mixed/config.proto.
	Timeout   uint32 `protobuf:"varint,3,opt,name=timeout,proto3" json:"timeout,omitempty"`
	UserLevel uint32 `protobuf:"varint,4,opt,name=user_level,json=userLevel,proto3" json:"user_level,omitempty"`
	// Socks
	UdpEnabled     bool                      `protobuf:"varint,5,opt,name=udp_enabled,json=udpEnabled,proto3" json:"udp_enabled,omitempty"`
	Address        *net.IPOrDomain           `protobuf:"bytes,6,opt,name=address,proto3" json:"address,omitempty"`
	PacketEncoding packetaddr.PacketAddrType `protobuf:"varint,7,opt,name=packet_encoding,json=packetEncoding,proto3,enum=v2ray.core.net.packetaddr.PacketAddrType" json:"packet_encoding,omitempty"`
	DeferLastReply bool                      `protobuf:"varint,8,opt,name=defer_last_reply,json=deferLastReply,proto3" json:"defer_last_reply,omitempty"`
	// HTTP
	AllowTransparent bool `protobuf:"varint,9,opt,name=allow_transparent,json=allowTransparent,proto3" json:"allow_transparent,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *ServerConfig) Reset() {
	*x = ServerConfig{}
	mi := &file_proxy_mixed_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ServerConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServerConfig) ProtoMessage() {}

func (x *ServerConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_mixed_config_proto_msgTypes[1]
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
	return file_proxy_mixed_config_proto_rawDescGZIP(), []int{1}
}

// Deprecated: Marked as deprecated in proxy/mixed/config.proto.
func (x *ServerConfig) GetAuthType() AuthType {
	if x != nil {
		return x.AuthType
	}
	return AuthType_NO_AUTH
}

func (x *ServerConfig) GetAccounts() map[string]string {
	if x != nil {
		return x.Accounts
	}
	return nil
}

// Deprecated: Marked as deprecated in proxy/mixed/config.proto.
func (x *ServerConfig) GetTimeout() uint32 {
	if x != nil {
		return x.Timeout
	}
	return 0
}

func (x *ServerConfig) GetUserLevel() uint32 {
	if x != nil {
		return x.UserLevel
	}
	return 0
}

func (x *ServerConfig) GetUdpEnabled() bool {
	if x != nil {
		return x.UdpEnabled
	}
	return false
}

func (x *ServerConfig) GetAddress() *net.IPOrDomain {
	if x != nil {
		return x.Address
	}
	return nil
}

func (x *ServerConfig) GetPacketEncoding() packetaddr.PacketAddrType {
	if x != nil {
		return x.PacketEncoding
	}
	return packetaddr.PacketAddrType(0)
}

func (x *ServerConfig) GetDeferLastReply() bool {
	if x != nil {
		return x.DeferLastReply
	}
	return false
}

func (x *ServerConfig) GetAllowTransparent() bool {
	if x != nil {
		return x.AllowTransparent
	}
	return false
}

var File_proxy_mixed_config_proto protoreflect.FileDescriptor

const file_proxy_mixed_config_proto_rawDesc = "" +
	"\n" +
	"\x18proxy/mixed/config.proto\x12\x16v2ray.core.proxy.mixed\x1a\x18common/net/address.proto\x1a\"common/net/packetaddr/config.proto\"A\n" +
	"\aAccount\x12\x1a\n" +
	"\busername\x18\x01 \x01(\tR\busername\x12\x1a\n" +
	"\bpassword\x18\x02 \x01(\tR\bpassword\"\xa4\x04\n" +
	"\fServerConfig\x12A\n" +
	"\tauth_type\x18\x01 \x01(\x0e2 .v2ray.core.proxy.mixed.AuthTypeB\x02\x18\x01R\bauthType\x12N\n" +
	"\baccounts\x18\x02 \x03(\v22.v2ray.core.proxy.mixed.ServerConfig.AccountsEntryR\baccounts\x12\x1c\n" +
	"\atimeout\x18\x03 \x01(\rB\x02\x18\x01R\atimeout\x12\x1d\n" +
	"\n" +
	"user_level\x18\x04 \x01(\rR\tuserLevel\x12\x1f\n" +
	"\vudp_enabled\x18\x05 \x01(\bR\n" +
	"udpEnabled\x12;\n" +
	"\aaddress\x18\x06 \x01(\v2!.v2ray.core.common.net.IPOrDomainR\aaddress\x12R\n" +
	"\x0fpacket_encoding\x18\a \x01(\x0e2).v2ray.core.net.packetaddr.PacketAddrTypeR\x0epacketEncoding\x12(\n" +
	"\x10defer_last_reply\x18\b \x01(\bR\x0edeferLastReply\x12+\n" +
	"\x11allow_transparent\x18\t \x01(\bR\x10allowTransparent\x1a;\n" +
	"\rAccountsEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01*%\n" +
	"\bAuthType\x12\v\n" +
	"\aNO_AUTH\x10\x00\x12\f\n" +
	"\bPASSWORD\x10\x01Bc\n" +
	"\x1acom.v2ray.core.proxy.mixedP\x01Z*github.com/v2fly/v2ray-core/v5/proxy/mixed\xaa\x02\x16V2Ray.Core.Proxy.Mixedb\x06proto3"

var (
	file_proxy_mixed_config_proto_rawDescOnce sync.Once
	file_proxy_mixed_config_proto_rawDescData []byte
)

func file_proxy_mixed_config_proto_rawDescGZIP() []byte {
	file_proxy_mixed_config_proto_rawDescOnce.Do(func() {
		file_proxy_mixed_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proxy_mixed_config_proto_rawDesc), len(file_proxy_mixed_config_proto_rawDesc)))
	})
	return file_proxy_mixed_config_proto_rawDescData
}

var file_proxy_mixed_config_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_proxy_mixed_config_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_proxy_mixed_config_proto_goTypes = []any{
	(AuthType)(0),                  // 0: v2ray.core.proxy.mixed.AuthType
	(*Account)(nil),                // 1: v2ray.core.proxy.mixed.Account
	(*ServerConfig)(nil),           // 2: v2ray.core.proxy.mixed.ServerConfig
	nil,                            // 3: v2ray.core.proxy.mixed.ServerConfig.AccountsEntry
	(*net.IPOrDomain)(nil),         // 4: v2ray.core.common.net.IPOrDomain
	(packetaddr.PacketAddrType)(0), // 5: v2ray.core.net.packetaddr.PacketAddrType
}
var file_proxy_mixed_config_proto_depIdxs = []int32{
	0, // 0: v2ray.core.proxy.mixed.ServerConfig.auth_type:type_name -> v2ray.core.proxy.mixed.AuthType
	3, // 1: v2ray.core.proxy.mixed.ServerConfig.accounts:type_name -> v2ray.core.proxy.mixed.ServerConfig.AccountsEntry
	4, // 2: v2ray.core.proxy.mixed.ServerConfig.address:type_name -> v2ray.core.common.net.IPOrDomain
	5, // 3: v2ray.core.proxy.mixed.ServerConfig.packet_encoding:type_name -> v2ray.core.net.packetaddr.PacketAddrType
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_proxy_mixed_config_proto_init() }
func file_proxy_mixed_config_proto_init() {
	if File_proxy_mixed_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proxy_mixed_config_proto_rawDesc), len(file_proxy_mixed_config_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proxy_mixed_config_proto_goTypes,
		DependencyIndexes: file_proxy_mixed_config_proto_depIdxs,
		EnumInfos:         file_proxy_mixed_config_proto_enumTypes,
		MessageInfos:      file_proxy_mixed_config_proto_msgTypes,
	}.Build()
	File_proxy_mixed_config_proto = out.File
	file_proxy_mixed_config_proto_goTypes = nil
	file_proxy_mixed_config_proto_depIdxs = nil
}
