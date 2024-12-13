package tuic

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
	state             protoimpl.MessageState `protogen:"open.v1"`
	Address           *net.IPOrDomain        `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Port              uint32                 `protobuf:"varint,2,opt,name=port,proto3" json:"port,omitempty"`
	Uuid              string                 `protobuf:"bytes,3,opt,name=uuid,proto3" json:"uuid,omitempty"`
	Password          string                 `protobuf:"bytes,4,opt,name=password,proto3" json:"password,omitempty"`
	CongestionControl string                 `protobuf:"bytes,5,opt,name=congestion_control,json=congestionControl,proto3" json:"congestion_control,omitempty"`
	UdpRelayMode      string                 `protobuf:"bytes,6,opt,name=udp_relay_mode,json=udpRelayMode,proto3" json:"udp_relay_mode,omitempty"`
	ZeroRttHandshake  bool                   `protobuf:"varint,7,opt,name=zero_rtt_handshake,json=zeroRttHandshake,proto3" json:"zero_rtt_handshake,omitempty"`
	Heartbeat         uint32                 `protobuf:"varint,8,opt,name=heartbeat,proto3" json:"heartbeat,omitempty"`
	DisableSni        bool                   `protobuf:"varint,9,opt,name=disable_sni,json=disableSni,proto3" json:"disable_sni,omitempty"`
	TlsSettings       *tls.Config            `protobuf:"bytes,10,opt,name=tls_settings,json=tlsSettings,proto3" json:"tls_settings,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *ClientConfig) Reset() {
	*x = ClientConfig{}
	mi := &file_proxy_tuic_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ClientConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClientConfig) ProtoMessage() {}

func (x *ClientConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_tuic_config_proto_msgTypes[0]
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
	return file_proxy_tuic_config_proto_rawDescGZIP(), []int{0}
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

func (x *ClientConfig) GetUuid() string {
	if x != nil {
		return x.Uuid
	}
	return ""
}

func (x *ClientConfig) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

func (x *ClientConfig) GetCongestionControl() string {
	if x != nil {
		return x.CongestionControl
	}
	return ""
}

func (x *ClientConfig) GetUdpRelayMode() string {
	if x != nil {
		return x.UdpRelayMode
	}
	return ""
}

func (x *ClientConfig) GetZeroRttHandshake() bool {
	if x != nil {
		return x.ZeroRttHandshake
	}
	return false
}

func (x *ClientConfig) GetHeartbeat() uint32 {
	if x != nil {
		return x.Heartbeat
	}
	return 0
}

func (x *ClientConfig) GetDisableSni() bool {
	if x != nil {
		return x.DisableSni
	}
	return false
}

func (x *ClientConfig) GetTlsSettings() *tls.Config {
	if x != nil {
		return x.TlsSettings
	}
	return nil
}

var File_proxy_tuic_config_proto protoreflect.FileDescriptor

const file_proxy_tuic_config_proto_rawDesc = "" +
	"\n" +
	"\x17proxy/tuic/config.proto\x12\x15v2ray.core.proxy.tuic\x1a common/protoext/extensions.proto\x1a\x18common/net/address.proto\x1a#transport/internet/tls/config.proto\"\xb5\x03\n" +
	"\fClientConfig\x12;\n" +
	"\aaddress\x18\x01 \x01(\v2!.v2ray.core.common.net.IPOrDomainR\aaddress\x12\x12\n" +
	"\x04port\x18\x02 \x01(\rR\x04port\x12\x12\n" +
	"\x04uuid\x18\x03 \x01(\tR\x04uuid\x12\x1a\n" +
	"\bpassword\x18\x04 \x01(\tR\bpassword\x12-\n" +
	"\x12congestion_control\x18\x05 \x01(\tR\x11congestionControl\x12$\n" +
	"\x0eudp_relay_mode\x18\x06 \x01(\tR\fudpRelayMode\x12,\n" +
	"\x12zero_rtt_handshake\x18\a \x01(\bR\x10zeroRttHandshake\x12\x1c\n" +
	"\theartbeat\x18\b \x01(\rR\theartbeat\x12\x1f\n" +
	"\vdisable_sni\x18\t \x01(\bR\n" +
	"disableSni\x12L\n" +
	"\ftls_settings\x18\n" +
	" \x01(\v2).v2ray.core.transport.internet.tls.ConfigR\vtlsSettings:\x14\x82\xb5\x18\x10\n" +
	"\boutbound\x12\x04tuicB`\n" +
	"\x19com.v2ray.core.proxy.tuicP\x01Z)github.com/v2fly/v2ray-core/v5/proxy/tuic\xaa\x02\x15V2Ray.Core.Proxy.Tuicb\x06proto3"

var (
	file_proxy_tuic_config_proto_rawDescOnce sync.Once
	file_proxy_tuic_config_proto_rawDescData []byte
)

func file_proxy_tuic_config_proto_rawDescGZIP() []byte {
	file_proxy_tuic_config_proto_rawDescOnce.Do(func() {
		file_proxy_tuic_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proxy_tuic_config_proto_rawDesc), len(file_proxy_tuic_config_proto_rawDesc)))
	})
	return file_proxy_tuic_config_proto_rawDescData
}

var file_proxy_tuic_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_proxy_tuic_config_proto_goTypes = []any{
	(*ClientConfig)(nil),   // 0: v2ray.core.proxy.tuic.ClientConfig
	(*net.IPOrDomain)(nil), // 1: v2ray.core.common.net.IPOrDomain
	(*tls.Config)(nil),     // 2: v2ray.core.transport.internet.tls.Config
}
var file_proxy_tuic_config_proto_depIdxs = []int32{
	1, // 0: v2ray.core.proxy.tuic.ClientConfig.address:type_name -> v2ray.core.common.net.IPOrDomain
	2, // 1: v2ray.core.proxy.tuic.ClientConfig.tls_settings:type_name -> v2ray.core.transport.internet.tls.Config
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_proxy_tuic_config_proto_init() }
func file_proxy_tuic_config_proto_init() {
	if File_proxy_tuic_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proxy_tuic_config_proto_rawDesc), len(file_proxy_tuic_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proxy_tuic_config_proto_goTypes,
		DependencyIndexes: file_proxy_tuic_config_proto_depIdxs,
		MessageInfos:      file_proxy_tuic_config_proto_msgTypes,
	}.Build()
	File_proxy_tuic_config_proto = out.File
	file_proxy_tuic_config_proto_goTypes = nil
	file_proxy_tuic_config_proto_depIdxs = nil
}
