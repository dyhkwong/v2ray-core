package shadowtls

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

type ClientConfig struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Address       *net.IPOrDomain        `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Port          uint32                 `protobuf:"varint,2,opt,name=port,proto3" json:"port,omitempty"`
	Password      string                 `protobuf:"bytes,3,opt,name=password,proto3" json:"password,omitempty"`
	Version       uint32                 `protobuf:"varint,4,opt,name=version,proto3" json:"version,omitempty"`
	TlsSettings   *tls.Config            `protobuf:"bytes,5,opt,name=tls_settings,json=tlsSettings,proto3" json:"tls_settings,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ClientConfig) Reset() {
	*x = ClientConfig{}
	mi := &file_proxy_shadowtls_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ClientConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClientConfig) ProtoMessage() {}

func (x *ClientConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_shadowtls_config_proto_msgTypes[0]
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
	return file_proxy_shadowtls_config_proto_rawDescGZIP(), []int{0}
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

func (x *ClientConfig) GetVersion() uint32 {
	if x != nil {
		return x.Version
	}
	return 0
}

func (x *ClientConfig) GetTlsSettings() *tls.Config {
	if x != nil {
		return x.TlsSettings
	}
	return nil
}

var File_proxy_shadowtls_config_proto protoreflect.FileDescriptor

const file_proxy_shadowtls_config_proto_rawDesc = "" +
	"\n" +
	"\x1cproxy/shadowtls/config.proto\x12\x1av2ray.core.proxy.shadowtls\x1a common/protoext/extensions.proto\x1a\x18common/net/address.proto\x1a#transport/internet/tls/config.proto\"\xfe\x01\n" +
	"\fClientConfig\x12;\n" +
	"\aaddress\x18\x01 \x01(\v2!.v2ray.core.common.net.IPOrDomainR\aaddress\x12\x12\n" +
	"\x04port\x18\x02 \x01(\rR\x04port\x12\x1a\n" +
	"\bpassword\x18\x03 \x01(\tR\bpassword\x12\x18\n" +
	"\aversion\x18\x04 \x01(\rR\aversion\x12L\n" +
	"\ftls_settings\x18\x05 \x01(\v2).v2ray.core.transport.internet.tls.ConfigR\vtlsSettings:\x19\x82\xb5\x18\x15\n" +
	"\boutbound\x12\tshadowtlsBo\n" +
	"\x1ecom.v2ray.core.proxy.shadowtlsP\x01Z.github.com/v2fly/v2ray-core/v5/proxy/shadowtls\xaa\x02\x1aV2Ray.Core.Proxy.ShadowTLSb\x06proto3"

var (
	file_proxy_shadowtls_config_proto_rawDescOnce sync.Once
	file_proxy_shadowtls_config_proto_rawDescData []byte
)

func file_proxy_shadowtls_config_proto_rawDescGZIP() []byte {
	file_proxy_shadowtls_config_proto_rawDescOnce.Do(func() {
		file_proxy_shadowtls_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proxy_shadowtls_config_proto_rawDesc), len(file_proxy_shadowtls_config_proto_rawDesc)))
	})
	return file_proxy_shadowtls_config_proto_rawDescData
}

var file_proxy_shadowtls_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_proxy_shadowtls_config_proto_goTypes = []any{
	(*ClientConfig)(nil),   // 0: v2ray.core.proxy.shadowtls.ClientConfig
	(*net.IPOrDomain)(nil), // 1: v2ray.core.common.net.IPOrDomain
	(*tls.Config)(nil),     // 2: v2ray.core.transport.internet.tls.Config
}
var file_proxy_shadowtls_config_proto_depIdxs = []int32{
	1, // 0: v2ray.core.proxy.shadowtls.ClientConfig.address:type_name -> v2ray.core.common.net.IPOrDomain
	2, // 1: v2ray.core.proxy.shadowtls.ClientConfig.tls_settings:type_name -> v2ray.core.transport.internet.tls.Config
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_proxy_shadowtls_config_proto_init() }
func file_proxy_shadowtls_config_proto_init() {
	if File_proxy_shadowtls_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proxy_shadowtls_config_proto_rawDesc), len(file_proxy_shadowtls_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proxy_shadowtls_config_proto_goTypes,
		DependencyIndexes: file_proxy_shadowtls_config_proto_depIdxs,
		MessageInfos:      file_proxy_shadowtls_config_proto_msgTypes,
	}.Build()
	File_proxy_shadowtls_config_proto = out.File
	file_proxy_shadowtls_config_proto_goTypes = nil
	file_proxy_shadowtls_config_proto_depIdxs = nil
}
