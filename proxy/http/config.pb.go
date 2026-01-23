package http

import (
	protocol "github.com/v2fly/v2ray-core/v5/common/protocol"
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

type ClientConfig_DomainStrategy int32

const (
	ClientConfig_USE_IP     ClientConfig_DomainStrategy = 0
	ClientConfig_USE_IP4    ClientConfig_DomainStrategy = 1
	ClientConfig_USE_IP6    ClientConfig_DomainStrategy = 2
	ClientConfig_PREFER_IP4 ClientConfig_DomainStrategy = 3
	ClientConfig_PREFER_IP6 ClientConfig_DomainStrategy = 4
)

// Enum value maps for ClientConfig_DomainStrategy.
var (
	ClientConfig_DomainStrategy_name = map[int32]string{
		0: "USE_IP",
		1: "USE_IP4",
		2: "USE_IP6",
		3: "PREFER_IP4",
		4: "PREFER_IP6",
	}
	ClientConfig_DomainStrategy_value = map[string]int32{
		"USE_IP":     0,
		"USE_IP4":    1,
		"USE_IP6":    2,
		"PREFER_IP4": 3,
		"PREFER_IP6": 4,
	}
)

func (x ClientConfig_DomainStrategy) Enum() *ClientConfig_DomainStrategy {
	p := new(ClientConfig_DomainStrategy)
	*p = x
	return p
}

func (x ClientConfig_DomainStrategy) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ClientConfig_DomainStrategy) Descriptor() protoreflect.EnumDescriptor {
	return file_proxy_http_config_proto_enumTypes[0].Descriptor()
}

func (ClientConfig_DomainStrategy) Type() protoreflect.EnumType {
	return &file_proxy_http_config_proto_enumTypes[0]
}

func (x ClientConfig_DomainStrategy) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ClientConfig_DomainStrategy.Descriptor instead.
func (ClientConfig_DomainStrategy) EnumDescriptor() ([]byte, []int) {
	return file_proxy_http_config_proto_rawDescGZIP(), []int{2, 0}
}

type Account struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Username      string                 `protobuf:"bytes,1,opt,name=username,proto3" json:"username,omitempty"`
	Password      string                 `protobuf:"bytes,2,opt,name=password,proto3" json:"password,omitempty"`
	Headers       map[string]string      `protobuf:"bytes,3,rep,name=headers,proto3" json:"headers,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Account) Reset() {
	*x = Account{}
	mi := &file_proxy_http_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Account) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Account) ProtoMessage() {}

func (x *Account) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_http_config_proto_msgTypes[0]
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
	return file_proxy_http_config_proto_rawDescGZIP(), []int{0}
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

func (x *Account) GetHeaders() map[string]string {
	if x != nil {
		return x.Headers
	}
	return nil
}

// Config for HTTP proxy server.
type ServerConfig struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Deprecated: Marked as deprecated in proxy/http/config.proto.
	Timeout          uint32            `protobuf:"varint,1,opt,name=timeout,proto3" json:"timeout,omitempty"`
	Accounts         map[string]string `protobuf:"bytes,2,rep,name=accounts,proto3" json:"accounts,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	AllowTransparent bool              `protobuf:"varint,3,opt,name=allow_transparent,json=allowTransparent,proto3" json:"allow_transparent,omitempty"`
	UserLevel        uint32            `protobuf:"varint,4,opt,name=user_level,json=userLevel,proto3" json:"user_level,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *ServerConfig) Reset() {
	*x = ServerConfig{}
	mi := &file_proxy_http_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ServerConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServerConfig) ProtoMessage() {}

func (x *ServerConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_http_config_proto_msgTypes[1]
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
	return file_proxy_http_config_proto_rawDescGZIP(), []int{1}
}

// Deprecated: Marked as deprecated in proxy/http/config.proto.
func (x *ServerConfig) GetTimeout() uint32 {
	if x != nil {
		return x.Timeout
	}
	return 0
}

func (x *ServerConfig) GetAccounts() map[string]string {
	if x != nil {
		return x.Accounts
	}
	return nil
}

func (x *ServerConfig) GetAllowTransparent() bool {
	if x != nil {
		return x.AllowTransparent
	}
	return false
}

func (x *ServerConfig) GetUserLevel() uint32 {
	if x != nil {
		return x.UserLevel
	}
	return 0
}

// ClientConfig is the protobuf config for HTTP proxy client.
type ClientConfig struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Sever is a list of HTTP server addresses.
	Server []*protocol.ServerEndpoint `protobuf:"bytes,1,rep,name=server,proto3" json:"server,omitempty"`
	// Deprecated: Do not use.
	TrustTunnelUdp bool `protobuf:"varint,1000,opt,name=trust_tunnel_udp,json=trustTunnelUdp,proto3" json:"trust_tunnel_udp,omitempty"`
	// Deprecated: Do not use.
	DomainStrategy ClientConfig_DomainStrategy `protobuf:"varint,1001,opt,name=domain_strategy,json=domainStrategy,proto3,enum=v2ray.core.proxy.http.ClientConfig_DomainStrategy" json:"domain_strategy,omitempty"`
	unknownFields  protoimpl.UnknownFields
	sizeCache      protoimpl.SizeCache
}

func (x *ClientConfig) Reset() {
	*x = ClientConfig{}
	mi := &file_proxy_http_config_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ClientConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClientConfig) ProtoMessage() {}

func (x *ClientConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_http_config_proto_msgTypes[2]
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
	return file_proxy_http_config_proto_rawDescGZIP(), []int{2}
}

func (x *ClientConfig) GetServer() []*protocol.ServerEndpoint {
	if x != nil {
		return x.Server
	}
	return nil
}

func (x *ClientConfig) GetTrustTunnelUdp() bool {
	if x != nil {
		return x.TrustTunnelUdp
	}
	return false
}

func (x *ClientConfig) GetDomainStrategy() ClientConfig_DomainStrategy {
	if x != nil {
		return x.DomainStrategy
	}
	return ClientConfig_USE_IP
}

var File_proxy_http_config_proto protoreflect.FileDescriptor

const file_proxy_http_config_proto_rawDesc = "" +
	"\n" +
	"\x17proxy/http/config.proto\x12\x15v2ray.core.proxy.http\x1a!common/protocol/server_spec.proto\"\xc4\x01\n" +
	"\aAccount\x12\x1a\n" +
	"\busername\x18\x01 \x01(\tR\busername\x12\x1a\n" +
	"\bpassword\x18\x02 \x01(\tR\bpassword\x12E\n" +
	"\aheaders\x18\x03 \x03(\v2+.v2ray.core.proxy.http.Account.HeadersEntryR\aheaders\x1a:\n" +
	"\fHeadersEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"\x84\x02\n" +
	"\fServerConfig\x12\x1c\n" +
	"\atimeout\x18\x01 \x01(\rB\x02\x18\x01R\atimeout\x12M\n" +
	"\baccounts\x18\x02 \x03(\v21.v2ray.core.proxy.http.ServerConfig.AccountsEntryR\baccounts\x12+\n" +
	"\x11allow_transparent\x18\x03 \x01(\bR\x10allowTransparent\x12\x1d\n" +
	"\n" +
	"user_level\x18\x04 \x01(\rR\tuserLevel\x1a;\n" +
	"\rAccountsEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"\xb3\x02\n" +
	"\fClientConfig\x12B\n" +
	"\x06server\x18\x01 \x03(\v2*.v2ray.core.common.protocol.ServerEndpointR\x06server\x12)\n" +
	"\x10trust_tunnel_udp\x18\xe8\a \x01(\bR\x0etrustTunnelUdp\x12\\\n" +
	"\x0fdomain_strategy\x18\xe9\a \x01(\x0e22.v2ray.core.proxy.http.ClientConfig.DomainStrategyR\x0edomainStrategy\"V\n" +
	"\x0eDomainStrategy\x12\n" +
	"\n" +
	"\x06USE_IP\x10\x00\x12\v\n" +
	"\aUSE_IP4\x10\x01\x12\v\n" +
	"\aUSE_IP6\x10\x02\x12\x0e\n" +
	"\n" +
	"PREFER_IP4\x10\x03\x12\x0e\n" +
	"\n" +
	"PREFER_IP6\x10\x04B`\n" +
	"\x19com.v2ray.core.proxy.httpP\x01Z)github.com/v2fly/v2ray-core/v5/proxy/http\xaa\x02\x15V2Ray.Core.Proxy.Httpb\x06proto3"

var (
	file_proxy_http_config_proto_rawDescOnce sync.Once
	file_proxy_http_config_proto_rawDescData []byte
)

func file_proxy_http_config_proto_rawDescGZIP() []byte {
	file_proxy_http_config_proto_rawDescOnce.Do(func() {
		file_proxy_http_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proxy_http_config_proto_rawDesc), len(file_proxy_http_config_proto_rawDesc)))
	})
	return file_proxy_http_config_proto_rawDescData
}

var file_proxy_http_config_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_proxy_http_config_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_proxy_http_config_proto_goTypes = []any{
	(ClientConfig_DomainStrategy)(0), // 0: v2ray.core.proxy.http.ClientConfig.DomainStrategy
	(*Account)(nil),                  // 1: v2ray.core.proxy.http.Account
	(*ServerConfig)(nil),             // 2: v2ray.core.proxy.http.ServerConfig
	(*ClientConfig)(nil),             // 3: v2ray.core.proxy.http.ClientConfig
	nil,                              // 4: v2ray.core.proxy.http.Account.HeadersEntry
	nil,                              // 5: v2ray.core.proxy.http.ServerConfig.AccountsEntry
	(*protocol.ServerEndpoint)(nil),  // 6: v2ray.core.common.protocol.ServerEndpoint
}
var file_proxy_http_config_proto_depIdxs = []int32{
	4, // 0: v2ray.core.proxy.http.Account.headers:type_name -> v2ray.core.proxy.http.Account.HeadersEntry
	5, // 1: v2ray.core.proxy.http.ServerConfig.accounts:type_name -> v2ray.core.proxy.http.ServerConfig.AccountsEntry
	6, // 2: v2ray.core.proxy.http.ClientConfig.server:type_name -> v2ray.core.common.protocol.ServerEndpoint
	0, // 3: v2ray.core.proxy.http.ClientConfig.domain_strategy:type_name -> v2ray.core.proxy.http.ClientConfig.DomainStrategy
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_proxy_http_config_proto_init() }
func file_proxy_http_config_proto_init() {
	if File_proxy_http_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proxy_http_config_proto_rawDesc), len(file_proxy_http_config_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proxy_http_config_proto_goTypes,
		DependencyIndexes: file_proxy_http_config_proto_depIdxs,
		EnumInfos:         file_proxy_http_config_proto_enumTypes,
		MessageInfos:      file_proxy_http_config_proto_msgTypes,
	}.Build()
	File_proxy_http_config_proto = out.File
	file_proxy_http_config_proto_goTypes = nil
	file_proxy_http_config_proto_depIdxs = nil
}
