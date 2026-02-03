package http3

import (
	net "github.com/v2fly/v2ray-core/v5/common/net"
	_ "github.com/v2fly/v2ray-core/v5/common/protoext"
	tls "github.com/v2fly/v2ray-core/v5/transport/internet/tls"
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
	return file_proxy_http3_config_proto_enumTypes[0].Descriptor()
}

func (ClientConfig_DomainStrategy) Type() protoreflect.EnumType {
	return &file_proxy_http3_config_proto_enumTypes[0]
}

func (x ClientConfig_DomainStrategy) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ClientConfig_DomainStrategy.Descriptor instead.
func (ClientConfig_DomainStrategy) EnumDescriptor() ([]byte, []int) {
	return file_proxy_http3_config_proto_rawDescGZIP(), []int{0, 0}
}

type ClientConfig struct {
	state          protoimpl.MessageState      `protogen:"open.v1"`
	Address        *net.IPOrDomain             `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Port           uint32                      `protobuf:"varint,2,opt,name=port,proto3" json:"port,omitempty"`
	Level          uint32                      `protobuf:"varint,3,opt,name=level,proto3" json:"level,omitempty"`
	Username       *string                     `protobuf:"bytes,4,opt,name=username,proto3,oneof" json:"username,omitempty"`
	Password       *string                     `protobuf:"bytes,5,opt,name=password,proto3,oneof" json:"password,omitempty"`
	Headers        map[string]string           `protobuf:"bytes,6,rep,name=headers,proto3" json:"headers,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	TlsSettings    *tls.Config                 `protobuf:"bytes,7,opt,name=tls_settings,json=tlsSettings,proto3" json:"tls_settings,omitempty"`
	TrustTunnelUdp bool                        `protobuf:"varint,1000,opt,name=trust_tunnel_udp,json=trustTunnelUdp,proto3" json:"trust_tunnel_udp,omitempty"`
	DomainStrategy ClientConfig_DomainStrategy `protobuf:"varint,1001,opt,name=domain_strategy,json=domainStrategy,proto3,enum=v2ray.core.proxy.http3.ClientConfig_DomainStrategy" json:"domain_strategy,omitempty"`
	// Deprecated: Do not use.
	RandomHeaderPadding bool `protobuf:"varint,1002,opt,name=random_header_padding,json=randomHeaderPadding,proto3" json:"random_header_padding,omitempty"`
	unknownFields       protoimpl.UnknownFields
	sizeCache           protoimpl.SizeCache
}

func (x *ClientConfig) Reset() {
	*x = ClientConfig{}
	mi := &file_proxy_http3_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ClientConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClientConfig) ProtoMessage() {}

func (x *ClientConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_http3_config_proto_msgTypes[0]
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
	return file_proxy_http3_config_proto_rawDescGZIP(), []int{0}
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

func (x *ClientConfig) GetLevel() uint32 {
	if x != nil {
		return x.Level
	}
	return 0
}

func (x *ClientConfig) GetUsername() string {
	if x != nil && x.Username != nil {
		return *x.Username
	}
	return ""
}

func (x *ClientConfig) GetPassword() string {
	if x != nil && x.Password != nil {
		return *x.Password
	}
	return ""
}

func (x *ClientConfig) GetHeaders() map[string]string {
	if x != nil {
		return x.Headers
	}
	return nil
}

func (x *ClientConfig) GetTlsSettings() *tls.Config {
	if x != nil {
		return x.TlsSettings
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

func (x *ClientConfig) GetRandomHeaderPadding() bool {
	if x != nil {
		return x.RandomHeaderPadding
	}
	return false
}

var File_proxy_http3_config_proto protoreflect.FileDescriptor

const file_proxy_http3_config_proto_rawDesc = "" +
	"\n" +
	"\x18proxy/http3/config.proto\x12\x16v2ray.core.proxy.http3\x1a\x18common/net/address.proto\x1a common/protoext/extensions.proto\x1a#transport/internet/tls/config.proto\"\xd6\x05\n" +
	"\fClientConfig\x12;\n" +
	"\aaddress\x18\x01 \x01(\v2!.v2ray.core.common.net.IPOrDomainR\aaddress\x12\x12\n" +
	"\x04port\x18\x02 \x01(\rR\x04port\x12\x14\n" +
	"\x05level\x18\x03 \x01(\rR\x05level\x12\x1f\n" +
	"\busername\x18\x04 \x01(\tH\x00R\busername\x88\x01\x01\x12\x1f\n" +
	"\bpassword\x18\x05 \x01(\tH\x01R\bpassword\x88\x01\x01\x12K\n" +
	"\aheaders\x18\x06 \x03(\v21.v2ray.core.proxy.http3.ClientConfig.HeadersEntryR\aheaders\x12L\n" +
	"\ftls_settings\x18\a \x01(\v2).v2ray.core.transport.internet.tls.ConfigR\vtlsSettings\x12)\n" +
	"\x10trust_tunnel_udp\x18\xe8\a \x01(\bR\x0etrustTunnelUdp\x12]\n" +
	"\x0fdomain_strategy\x18\xe9\a \x01(\x0e23.v2ray.core.proxy.http3.ClientConfig.DomainStrategyR\x0edomainStrategy\x123\n" +
	"\x15random_header_padding\x18\xea\a \x01(\bR\x13randomHeaderPadding\x1a:\n" +
	"\fHeadersEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"V\n" +
	"\x0eDomainStrategy\x12\n" +
	"\n" +
	"\x06USE_IP\x10\x00\x12\v\n" +
	"\aUSE_IP4\x10\x01\x12\v\n" +
	"\aUSE_IP6\x10\x02\x12\x0e\n" +
	"\n" +
	"PREFER_IP4\x10\x03\x12\x0e\n" +
	"\n" +
	"PREFER_IP6\x10\x04:\x15\x82\xb5\x18\x11\n" +
	"\boutbound\x12\x05http3B\v\n" +
	"\t_usernameB\v\n" +
	"\t_passwordBc\n" +
	"\x1acom.v2ray.core.proxy.http3P\x01Z*github.com/v2fly/v2ray-core/v5/proxy/http3\xaa\x02\x16V2Ray.Core.Proxy.Http3b\x06proto3"

var (
	file_proxy_http3_config_proto_rawDescOnce sync.Once
	file_proxy_http3_config_proto_rawDescData []byte
)

func file_proxy_http3_config_proto_rawDescGZIP() []byte {
	file_proxy_http3_config_proto_rawDescOnce.Do(func() {
		file_proxy_http3_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proxy_http3_config_proto_rawDesc), len(file_proxy_http3_config_proto_rawDesc)))
	})
	return file_proxy_http3_config_proto_rawDescData
}

var file_proxy_http3_config_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_proxy_http3_config_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_proxy_http3_config_proto_goTypes = []any{
	(ClientConfig_DomainStrategy)(0), // 0: v2ray.core.proxy.http3.ClientConfig.DomainStrategy
	(*ClientConfig)(nil),             // 1: v2ray.core.proxy.http3.ClientConfig
	nil,                              // 2: v2ray.core.proxy.http3.ClientConfig.HeadersEntry
	(*net.IPOrDomain)(nil),           // 3: v2ray.core.common.net.IPOrDomain
	(*tls.Config)(nil),               // 4: v2ray.core.transport.internet.tls.Config
}
var file_proxy_http3_config_proto_depIdxs = []int32{
	3, // 0: v2ray.core.proxy.http3.ClientConfig.address:type_name -> v2ray.core.common.net.IPOrDomain
	2, // 1: v2ray.core.proxy.http3.ClientConfig.headers:type_name -> v2ray.core.proxy.http3.ClientConfig.HeadersEntry
	4, // 2: v2ray.core.proxy.http3.ClientConfig.tls_settings:type_name -> v2ray.core.transport.internet.tls.Config
	0, // 3: v2ray.core.proxy.http3.ClientConfig.domain_strategy:type_name -> v2ray.core.proxy.http3.ClientConfig.DomainStrategy
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_proxy_http3_config_proto_init() }
func file_proxy_http3_config_proto_init() {
	if File_proxy_http3_config_proto != nil {
		return
	}
	file_proxy_http3_config_proto_msgTypes[0].OneofWrappers = []any{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proxy_http3_config_proto_rawDesc), len(file_proxy_http3_config_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proxy_http3_config_proto_goTypes,
		DependencyIndexes: file_proxy_http3_config_proto_depIdxs,
		EnumInfos:         file_proxy_http3_config_proto_enumTypes,
		MessageInfos:      file_proxy_http3_config_proto_msgTypes,
	}.Build()
	File_proxy_http3_config_proto = out.File
	file_proxy_http3_config_proto_goTypes = nil
	file_proxy_http3_config_proto_depIdxs = nil
}
