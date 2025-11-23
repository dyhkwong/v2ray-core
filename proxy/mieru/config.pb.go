package mieru

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
	state          protoimpl.MessageState `protogen:"open.v1"`
	Address        *net.IPOrDomain        `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Port           uint32                 `protobuf:"varint,2,opt,name=port,proto3" json:"port,omitempty"`
	PortRange      []string               `protobuf:"bytes,3,rep,name=port_range,json=portRange,proto3" json:"port_range,omitempty"`
	Username       string                 `protobuf:"bytes,4,opt,name=username,proto3" json:"username,omitempty"`
	Password       string                 `protobuf:"bytes,5,opt,name=password,proto3" json:"password,omitempty"`
	Protocol       string                 `protobuf:"bytes,6,opt,name=protocol,proto3" json:"protocol,omitempty"`
	Multiplexing   string                 `protobuf:"bytes,7,opt,name=multiplexing,proto3" json:"multiplexing,omitempty"`
	HandshakeMode  string                 `protobuf:"bytes,8,opt,name=handshake_mode,json=handshakeMode,proto3" json:"handshake_mode,omitempty"`
	TrafficPattern string                 `protobuf:"bytes,9,opt,name=traffic_pattern,json=trafficPattern,proto3" json:"traffic_pattern,omitempty"`
	unknownFields  protoimpl.UnknownFields
	sizeCache      protoimpl.SizeCache
}

func (x *ClientConfig) Reset() {
	*x = ClientConfig{}
	mi := &file_proxy_mieru_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ClientConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClientConfig) ProtoMessage() {}

func (x *ClientConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_mieru_config_proto_msgTypes[0]
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
	return file_proxy_mieru_config_proto_rawDescGZIP(), []int{0}
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

func (x *ClientConfig) GetPortRange() []string {
	if x != nil {
		return x.PortRange
	}
	return nil
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

func (x *ClientConfig) GetProtocol() string {
	if x != nil {
		return x.Protocol
	}
	return ""
}

func (x *ClientConfig) GetMultiplexing() string {
	if x != nil {
		return x.Multiplexing
	}
	return ""
}

func (x *ClientConfig) GetHandshakeMode() string {
	if x != nil {
		return x.HandshakeMode
	}
	return ""
}

func (x *ClientConfig) GetTrafficPattern() string {
	if x != nil {
		return x.TrafficPattern
	}
	return ""
}

var File_proxy_mieru_config_proto protoreflect.FileDescriptor

const file_proxy_mieru_config_proto_rawDesc = "" +
	"\n" +
	"\x18proxy/mieru/config.proto\x12\x16v2ray.core.proxy.mieru\x1a common/protoext/extensions.proto\x1a\x18common/net/address.proto\"\xdd\x02\n" +
	"\fClientConfig\x12;\n" +
	"\aaddress\x18\x01 \x01(\v2!.v2ray.core.common.net.IPOrDomainR\aaddress\x12\x12\n" +
	"\x04port\x18\x02 \x01(\rR\x04port\x12\x1d\n" +
	"\n" +
	"port_range\x18\x03 \x03(\tR\tportRange\x12\x1a\n" +
	"\busername\x18\x04 \x01(\tR\busername\x12\x1a\n" +
	"\bpassword\x18\x05 \x01(\tR\bpassword\x12\x1a\n" +
	"\bprotocol\x18\x06 \x01(\tR\bprotocol\x12\"\n" +
	"\fmultiplexing\x18\a \x01(\tR\fmultiplexing\x12%\n" +
	"\x0ehandshake_mode\x18\b \x01(\tR\rhandshakeMode\x12'\n" +
	"\x0ftraffic_pattern\x18\t \x01(\tR\x0etrafficPattern:\x15\x82\xb5\x18\x11\n" +
	"\boutbound\x12\x05mieruBc\n" +
	"\x1acom.v2ray.core.proxy.mieruP\x01Z*github.com/v2fly/v2ray-core/v5/proxy/mieru\xaa\x02\x16V2Ray.Core.Proxy.Mierub\x06proto3"

var (
	file_proxy_mieru_config_proto_rawDescOnce sync.Once
	file_proxy_mieru_config_proto_rawDescData []byte
)

func file_proxy_mieru_config_proto_rawDescGZIP() []byte {
	file_proxy_mieru_config_proto_rawDescOnce.Do(func() {
		file_proxy_mieru_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proxy_mieru_config_proto_rawDesc), len(file_proxy_mieru_config_proto_rawDesc)))
	})
	return file_proxy_mieru_config_proto_rawDescData
}

var file_proxy_mieru_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_proxy_mieru_config_proto_goTypes = []any{
	(*ClientConfig)(nil),   // 0: v2ray.core.proxy.mieru.ClientConfig
	(*net.IPOrDomain)(nil), // 1: v2ray.core.common.net.IPOrDomain
}
var file_proxy_mieru_config_proto_depIdxs = []int32{
	1, // 0: v2ray.core.proxy.mieru.ClientConfig.address:type_name -> v2ray.core.common.net.IPOrDomain
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_proxy_mieru_config_proto_init() }
func file_proxy_mieru_config_proto_init() {
	if File_proxy_mieru_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proxy_mieru_config_proto_rawDesc), len(file_proxy_mieru_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proxy_mieru_config_proto_goTypes,
		DependencyIndexes: file_proxy_mieru_config_proto_depIdxs,
		MessageInfos:      file_proxy_mieru_config_proto_msgTypes,
	}.Build()
	File_proxy_mieru_config_proto = out.File
	file_proxy_mieru_config_proto_goTypes = nil
	file_proxy_mieru_config_proto_depIdxs = nil
}
