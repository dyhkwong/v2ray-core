package mirrorenrollment

import (
	serial "github.com/v2fly/v2ray-core/v4/common/serial"
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

type Config struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// This will be handled by the TLS Mirror server, the enrollment part only accepts existing connections.
	PrimaryIngressOutbound  string                 `protobuf:"bytes,1,opt,name=primary_ingress_outbound,json=primaryIngressOutbound,proto3" json:"primary_ingress_outbound,omitempty"`
	PrimaryEgressOutbound   string                 `protobuf:"bytes,2,opt,name=primary_egress_outbound,json=primaryEgressOutbound,proto3" json:"primary_egress_outbound,omitempty"`
	BootstrapIngressUrl     []string               `protobuf:"bytes,3,rep,name=bootstrap_ingress_url,json=bootstrapIngressUrl,proto3" json:"bootstrap_ingress_url,omitempty"`
	BootstrapEgressUrl      []string               `protobuf:"bytes,4,rep,name=bootstrap_egress_url,json=bootstrapEgressUrl,proto3" json:"bootstrap_egress_url,omitempty"`
	BootstrapIngressConfig  []*serial.TypedMessage `protobuf:"bytes,5,rep,name=bootstrap_ingress_config,json=bootstrapIngressConfig,proto3" json:"bootstrap_ingress_config,omitempty"`
	BootstrapEgressConfig   []*serial.TypedMessage `protobuf:"bytes,6,rep,name=bootstrap_egress_config,json=bootstrapEgressConfig,proto3" json:"bootstrap_egress_config,omitempty"`
	BootstrapEgressOutbound string                 `protobuf:"bytes,7,opt,name=bootstrap_egress_outbound,json=bootstrapEgressOutbound,proto3" json:"bootstrap_egress_outbound,omitempty"`
	unknownFields           protoimpl.UnknownFields
	sizeCache               protoimpl.SizeCache
}

func (x *Config) Reset() {
	*x = Config{}
	mi := &file_transport_internet_tlsmirror_mirrorenrollment_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Config) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config) ProtoMessage() {}

func (x *Config) ProtoReflect() protoreflect.Message {
	mi := &file_transport_internet_tlsmirror_mirrorenrollment_config_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Config.ProtoReflect.Descriptor instead.
func (*Config) Descriptor() ([]byte, []int) {
	return file_transport_internet_tlsmirror_mirrorenrollment_config_proto_rawDescGZIP(), []int{0}
}

func (x *Config) GetPrimaryIngressOutbound() string {
	if x != nil {
		return x.PrimaryIngressOutbound
	}
	return ""
}

func (x *Config) GetPrimaryEgressOutbound() string {
	if x != nil {
		return x.PrimaryEgressOutbound
	}
	return ""
}

func (x *Config) GetBootstrapIngressUrl() []string {
	if x != nil {
		return x.BootstrapIngressUrl
	}
	return nil
}

func (x *Config) GetBootstrapEgressUrl() []string {
	if x != nil {
		return x.BootstrapEgressUrl
	}
	return nil
}

func (x *Config) GetBootstrapIngressConfig() []*serial.TypedMessage {
	if x != nil {
		return x.BootstrapIngressConfig
	}
	return nil
}

func (x *Config) GetBootstrapEgressConfig() []*serial.TypedMessage {
	if x != nil {
		return x.BootstrapEgressConfig
	}
	return nil
}

func (x *Config) GetBootstrapEgressOutbound() string {
	if x != nil {
		return x.BootstrapEgressOutbound
	}
	return ""
}

var File_transport_internet_tlsmirror_mirrorenrollment_config_proto protoreflect.FileDescriptor

const file_transport_internet_tlsmirror_mirrorenrollment_config_proto_rawDesc = "" +
	"\n" +
	":transport/internet/tlsmirror/mirrorenrollment/config.proto\x128v2ray.core.transport.internet.tlsmirror.mirrorenrollment\x1a!common/serial/typed_message.proto\"\xde\x03\n" +
	"\x06Config\x128\n" +
	"\x18primary_ingress_outbound\x18\x01 \x01(\tR\x16primaryIngressOutbound\x126\n" +
	"\x17primary_egress_outbound\x18\x02 \x01(\tR\x15primaryEgressOutbound\x122\n" +
	"\x15bootstrap_ingress_url\x18\x03 \x03(\tR\x13bootstrapIngressUrl\x120\n" +
	"\x14bootstrap_egress_url\x18\x04 \x03(\tR\x12bootstrapEgressUrl\x12`\n" +
	"\x18bootstrap_ingress_config\x18\x05 \x03(\v2&.v2ray.core.common.serial.TypedMessageR\x16bootstrapIngressConfig\x12^\n" +
	"\x17bootstrap_egress_config\x18\x06 \x03(\v2&.v2ray.core.common.serial.TypedMessageR\x15bootstrapEgressConfig\x12:\n" +
	"\x19bootstrap_egress_outbound\x18\a \x01(\tR\x17bootstrapEgressOutboundB\xc9\x01\n" +
	"<com.v2ray.core.transport.internet.tlsmirror.mirrorenrollmentP\x01ZLgithub.com/v2fly/v2ray-core/v4/transport/internet/tlsmirror/mirrorenrollment\xaa\x028V2Ray.Core.Transport.Internet.Tlsmirror.MirrorEnrollmentb\x06proto3"

var (
	file_transport_internet_tlsmirror_mirrorenrollment_config_proto_rawDescOnce sync.Once
	file_transport_internet_tlsmirror_mirrorenrollment_config_proto_rawDescData []byte
)

func file_transport_internet_tlsmirror_mirrorenrollment_config_proto_rawDescGZIP() []byte {
	file_transport_internet_tlsmirror_mirrorenrollment_config_proto_rawDescOnce.Do(func() {
		file_transport_internet_tlsmirror_mirrorenrollment_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_transport_internet_tlsmirror_mirrorenrollment_config_proto_rawDesc), len(file_transport_internet_tlsmirror_mirrorenrollment_config_proto_rawDesc)))
	})
	return file_transport_internet_tlsmirror_mirrorenrollment_config_proto_rawDescData
}

var file_transport_internet_tlsmirror_mirrorenrollment_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_transport_internet_tlsmirror_mirrorenrollment_config_proto_goTypes = []any{
	(*Config)(nil),              // 0: v2ray.core.transport.internet.tlsmirror.mirrorenrollment.Config
	(*serial.TypedMessage)(nil), // 1: v2ray.core.common.serial.TypedMessage
}
var file_transport_internet_tlsmirror_mirrorenrollment_config_proto_depIdxs = []int32{
	1, // 0: v2ray.core.transport.internet.tlsmirror.mirrorenrollment.Config.bootstrap_ingress_config:type_name -> v2ray.core.common.serial.TypedMessage
	1, // 1: v2ray.core.transport.internet.tlsmirror.mirrorenrollment.Config.bootstrap_egress_config:type_name -> v2ray.core.common.serial.TypedMessage
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_transport_internet_tlsmirror_mirrorenrollment_config_proto_init() }
func file_transport_internet_tlsmirror_mirrorenrollment_config_proto_init() {
	if File_transport_internet_tlsmirror_mirrorenrollment_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_transport_internet_tlsmirror_mirrorenrollment_config_proto_rawDesc), len(file_transport_internet_tlsmirror_mirrorenrollment_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_transport_internet_tlsmirror_mirrorenrollment_config_proto_goTypes,
		DependencyIndexes: file_transport_internet_tlsmirror_mirrorenrollment_config_proto_depIdxs,
		MessageInfos:      file_transport_internet_tlsmirror_mirrorenrollment_config_proto_msgTypes,
	}.Build()
	File_transport_internet_tlsmirror_mirrorenrollment_config_proto = out.File
	file_transport_internet_tlsmirror_mirrorenrollment_config_proto_goTypes = nil
	file_transport_internet_tlsmirror_mirrorenrollment_config_proto_depIdxs = nil
}
