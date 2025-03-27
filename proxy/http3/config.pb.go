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

type ClientConfig struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Address       *net.IPOrDomain        `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Port          uint32                 `protobuf:"varint,2,opt,name=port,proto3" json:"port,omitempty"`
	Level         uint32                 `protobuf:"varint,3,opt,name=level,proto3" json:"level,omitempty"`
	Username      string                 `protobuf:"bytes,4,opt,name=username,proto3" json:"username,omitempty"`
	Password      string                 `protobuf:"bytes,5,opt,name=password,proto3" json:"password,omitempty"`
	Headers       map[string]string      `protobuf:"bytes,6,rep,name=headers,proto3" json:"headers,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	TlsSettings   *tls.Config            `protobuf:"bytes,7,opt,name=tls_settings,json=tlsSettings,proto3" json:"tls_settings,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
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
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *ClientConfig) GetPassword() string {
	if x != nil {
		return x.Password
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

var File_proxy_http3_config_proto protoreflect.FileDescriptor

const file_proxy_http3_config_proto_rawDesc = "" +
	"\n" +
	"\x18proxy/http3/config.proto\x12\x16v2ray.core.proxy.http3\x1a\x18common/net/address.proto\x1a common/protoext/extensions.proto\x1a#transport/internet/tls/config.proto\"\x9b\x03\n" +
	"\fClientConfig\x12;\n" +
	"\aaddress\x18\x01 \x01(\v2!.v2ray.core.common.net.IPOrDomainR\aaddress\x12\x12\n" +
	"\x04port\x18\x02 \x01(\rR\x04port\x12\x14\n" +
	"\x05level\x18\x03 \x01(\rR\x05level\x12\x1a\n" +
	"\busername\x18\x04 \x01(\tR\busername\x12\x1a\n" +
	"\bpassword\x18\x05 \x01(\tR\bpassword\x12K\n" +
	"\aheaders\x18\x06 \x03(\v21.v2ray.core.proxy.http3.ClientConfig.HeadersEntryR\aheaders\x12L\n" +
	"\ftls_settings\x18\a \x01(\v2).v2ray.core.transport.internet.tls.ConfigR\vtlsSettings\x1a:\n" +
	"\fHeadersEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01:\x15\x82\xb5\x18\x11\n" +
	"\boutbound\x12\x05http3Bc\n" +
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

var file_proxy_http3_config_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_proxy_http3_config_proto_goTypes = []any{
	(*ClientConfig)(nil),   // 0: v2ray.core.proxy.http3.ClientConfig
	nil,                    // 1: v2ray.core.proxy.http3.ClientConfig.HeadersEntry
	(*net.IPOrDomain)(nil), // 2: v2ray.core.common.net.IPOrDomain
	(*tls.Config)(nil),     // 3: v2ray.core.transport.internet.tls.Config
}
var file_proxy_http3_config_proto_depIdxs = []int32{
	2, // 0: v2ray.core.proxy.http3.ClientConfig.address:type_name -> v2ray.core.common.net.IPOrDomain
	1, // 1: v2ray.core.proxy.http3.ClientConfig.headers:type_name -> v2ray.core.proxy.http3.ClientConfig.HeadersEntry
	3, // 2: v2ray.core.proxy.http3.ClientConfig.tls_settings:type_name -> v2ray.core.transport.internet.tls.Config
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_proxy_http3_config_proto_init() }
func file_proxy_http3_config_proto_init() {
	if File_proxy_http3_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proxy_http3_config_proto_rawDesc), len(file_proxy_http3_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proxy_http3_config_proto_goTypes,
		DependencyIndexes: file_proxy_http3_config_proto_depIdxs,
		MessageInfos:      file_proxy_http3_config_proto_msgTypes,
	}.Build()
	File_proxy_http3_config_proto = out.File
	file_proxy_http3_config_proto_goTypes = nil
	file_proxy_http3_config_proto_depIdxs = nil
}
