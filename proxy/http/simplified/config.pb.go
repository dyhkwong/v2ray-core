package simplified

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
	return file_proxy_http_simplified_config_proto_enumTypes[0].Descriptor()
}

func (ClientConfig_DomainStrategy) Type() protoreflect.EnumType {
	return &file_proxy_http_simplified_config_proto_enumTypes[0]
}

func (x ClientConfig_DomainStrategy) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ClientConfig_DomainStrategy.Descriptor instead.
func (ClientConfig_DomainStrategy) EnumDescriptor() ([]byte, []int) {
	return file_proxy_http_simplified_config_proto_rawDescGZIP(), []int{1, 0}
}

type ServerConfig struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ServerConfig) Reset() {
	*x = ServerConfig{}
	mi := &file_proxy_http_simplified_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ServerConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServerConfig) ProtoMessage() {}

func (x *ServerConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_http_simplified_config_proto_msgTypes[0]
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
	return file_proxy_http_simplified_config_proto_rawDescGZIP(), []int{0}
}

type ClientConfig struct {
	state   protoimpl.MessageState `protogen:"open.v1"`
	Address *net.IPOrDomain        `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Port    uint32                 `protobuf:"varint,2,opt,name=port,proto3" json:"port,omitempty"`
	// Deprecated: Do not use.
	TrustTunnelUdp bool `protobuf:"varint,1000,opt,name=trust_tunnel_udp,json=trustTunnelUdp,proto3" json:"trust_tunnel_udp,omitempty"`
	// Deprecated: Do not use.
	DomainStrategy ClientConfig_DomainStrategy `protobuf:"varint,1001,opt,name=domain_strategy,json=domainStrategy,proto3,enum=v2ray.core.proxy.http.simplified.ClientConfig_DomainStrategy" json:"domain_strategy,omitempty"`
	unknownFields  protoimpl.UnknownFields
	sizeCache      protoimpl.SizeCache
}

func (x *ClientConfig) Reset() {
	*x = ClientConfig{}
	mi := &file_proxy_http_simplified_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ClientConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClientConfig) ProtoMessage() {}

func (x *ClientConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_http_simplified_config_proto_msgTypes[1]
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
	return file_proxy_http_simplified_config_proto_rawDescGZIP(), []int{1}
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

var File_proxy_http_simplified_config_proto protoreflect.FileDescriptor

const file_proxy_http_simplified_config_proto_rawDesc = "" +
	"\n" +
	"\"proxy/http/simplified/config.proto\x12 v2ray.core.proxy.http.simplified\x1a common/protoext/extensions.proto\x1a\x18common/net/address.proto\"#\n" +
	"\fServerConfig:\x13\x82\xb5\x18\x0f\n" +
	"\ainbound\x12\x04http\"\xe1\x02\n" +
	"\fClientConfig\x12;\n" +
	"\aaddress\x18\x01 \x01(\v2!.v2ray.core.common.net.IPOrDomainR\aaddress\x12\x12\n" +
	"\x04port\x18\x02 \x01(\rR\x04port\x12)\n" +
	"\x10trust_tunnel_udp\x18\xe8\a \x01(\bR\x0etrustTunnelUdp\x12g\n" +
	"\x0fdomain_strategy\x18\xe9\a \x01(\x0e2=.v2ray.core.proxy.http.simplified.ClientConfig.DomainStrategyR\x0edomainStrategy\"V\n" +
	"\x0eDomainStrategy\x12\n" +
	"\n" +
	"\x06USE_IP\x10\x00\x12\v\n" +
	"\aUSE_IP4\x10\x01\x12\v\n" +
	"\aUSE_IP6\x10\x02\x12\x0e\n" +
	"\n" +
	"PREFER_IP4\x10\x03\x12\x0e\n" +
	"\n" +
	"PREFER_IP6\x10\x04:\x14\x82\xb5\x18\x10\n" +
	"\boutbound\x12\x04httpB\x81\x01\n" +
	"$com.v2ray.core.proxy.http.simplifiedP\x01Z4github.com/v2fly/v2ray-core/v5/proxy/http/simplified\xaa\x02 V2Ray.Core.Proxy.Http.Simplifiedb\x06proto3"

var (
	file_proxy_http_simplified_config_proto_rawDescOnce sync.Once
	file_proxy_http_simplified_config_proto_rawDescData []byte
)

func file_proxy_http_simplified_config_proto_rawDescGZIP() []byte {
	file_proxy_http_simplified_config_proto_rawDescOnce.Do(func() {
		file_proxy_http_simplified_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proxy_http_simplified_config_proto_rawDesc), len(file_proxy_http_simplified_config_proto_rawDesc)))
	})
	return file_proxy_http_simplified_config_proto_rawDescData
}

var file_proxy_http_simplified_config_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_proxy_http_simplified_config_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_proxy_http_simplified_config_proto_goTypes = []any{
	(ClientConfig_DomainStrategy)(0), // 0: v2ray.core.proxy.http.simplified.ClientConfig.DomainStrategy
	(*ServerConfig)(nil),             // 1: v2ray.core.proxy.http.simplified.ServerConfig
	(*ClientConfig)(nil),             // 2: v2ray.core.proxy.http.simplified.ClientConfig
	(*net.IPOrDomain)(nil),           // 3: v2ray.core.common.net.IPOrDomain
}
var file_proxy_http_simplified_config_proto_depIdxs = []int32{
	3, // 0: v2ray.core.proxy.http.simplified.ClientConfig.address:type_name -> v2ray.core.common.net.IPOrDomain
	0, // 1: v2ray.core.proxy.http.simplified.ClientConfig.domain_strategy:type_name -> v2ray.core.proxy.http.simplified.ClientConfig.DomainStrategy
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_proxy_http_simplified_config_proto_init() }
func file_proxy_http_simplified_config_proto_init() {
	if File_proxy_http_simplified_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proxy_http_simplified_config_proto_rawDesc), len(file_proxy_http_simplified_config_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proxy_http_simplified_config_proto_goTypes,
		DependencyIndexes: file_proxy_http_simplified_config_proto_depIdxs,
		EnumInfos:         file_proxy_http_simplified_config_proto_enumTypes,
		MessageInfos:      file_proxy_http_simplified_config_proto_msgTypes,
	}.Build()
	File_proxy_http_simplified_config_proto = out.File
	file_proxy_http_simplified_config_proto_goTypes = nil
	file_proxy_http_simplified_config_proto_depIdxs = nil
}
