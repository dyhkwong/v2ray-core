package splithttp

import (
	net "github.com/v2fly/v2ray-core/v5/common/net"
	_ "github.com/v2fly/v2ray-core/v5/common/protoext"
	internet "github.com/v2fly/v2ray-core/v5/transport/internet"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	anypb "google.golang.org/protobuf/types/known/anypb"
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

type XmuxConfig struct {
	state            protoimpl.MessageState `protogen:"open.v1"`
	MaxConcurrency   string                 `protobuf:"bytes,1,opt,name=maxConcurrency,proto3" json:"maxConcurrency,omitempty"`
	MaxConnections   string                 `protobuf:"bytes,2,opt,name=maxConnections,proto3" json:"maxConnections,omitempty"`
	CMaxReuseTimes   string                 `protobuf:"bytes,3,opt,name=cMaxReuseTimes,proto3" json:"cMaxReuseTimes,omitempty"`
	HMaxRequestTimes string                 `protobuf:"bytes,4,opt,name=hMaxRequestTimes,proto3" json:"hMaxRequestTimes,omitempty"`
	HMaxReusableSecs string                 `protobuf:"bytes,5,opt,name=hMaxReusableSecs,proto3" json:"hMaxReusableSecs,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *XmuxConfig) Reset() {
	*x = XmuxConfig{}
	mi := &file_transport_internet_splithttp_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *XmuxConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*XmuxConfig) ProtoMessage() {}

func (x *XmuxConfig) ProtoReflect() protoreflect.Message {
	mi := &file_transport_internet_splithttp_config_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use XmuxConfig.ProtoReflect.Descriptor instead.
func (*XmuxConfig) Descriptor() ([]byte, []int) {
	return file_transport_internet_splithttp_config_proto_rawDescGZIP(), []int{0}
}

func (x *XmuxConfig) GetMaxConcurrency() string {
	if x != nil {
		return x.MaxConcurrency
	}
	return ""
}

func (x *XmuxConfig) GetMaxConnections() string {
	if x != nil {
		return x.MaxConnections
	}
	return ""
}

func (x *XmuxConfig) GetCMaxReuseTimes() string {
	if x != nil {
		return x.CMaxReuseTimes
	}
	return ""
}

func (x *XmuxConfig) GetHMaxRequestTimes() string {
	if x != nil {
		return x.HMaxRequestTimes
	}
	return ""
}

func (x *XmuxConfig) GetHMaxReusableSecs() string {
	if x != nil {
		return x.HMaxReusableSecs
	}
	return ""
}

type DownloadConfig struct {
	state             protoimpl.MessageState `protogen:"open.v1"`
	Address           *net.IPOrDomain        `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Port              uint32                 `protobuf:"varint,2,opt,name=port,proto3" json:"port,omitempty"`
	TransportSettings *anypb.Any             `protobuf:"bytes,3,opt,name=transport_settings,json=transportSettings,proto3" json:"transport_settings,omitempty"`
	SecurityType      string                 `protobuf:"bytes,4,opt,name=security_type,json=securityType,proto3" json:"security_type,omitempty"`
	SecuritySettings  *anypb.Any             `protobuf:"bytes,5,opt,name=security_settings,json=securitySettings,proto3" json:"security_settings,omitempty"`
	SocketSettings    *internet.SocketConfig `protobuf:"bytes,6,opt,name=socket_settings,json=socketSettings,proto3" json:"socket_settings,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *DownloadConfig) Reset() {
	*x = DownloadConfig{}
	mi := &file_transport_internet_splithttp_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DownloadConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DownloadConfig) ProtoMessage() {}

func (x *DownloadConfig) ProtoReflect() protoreflect.Message {
	mi := &file_transport_internet_splithttp_config_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DownloadConfig.ProtoReflect.Descriptor instead.
func (*DownloadConfig) Descriptor() ([]byte, []int) {
	return file_transport_internet_splithttp_config_proto_rawDescGZIP(), []int{1}
}

func (x *DownloadConfig) GetAddress() *net.IPOrDomain {
	if x != nil {
		return x.Address
	}
	return nil
}

func (x *DownloadConfig) GetPort() uint32 {
	if x != nil {
		return x.Port
	}
	return 0
}

func (x *DownloadConfig) GetTransportSettings() *anypb.Any {
	if x != nil {
		return x.TransportSettings
	}
	return nil
}

func (x *DownloadConfig) GetSecurityType() string {
	if x != nil {
		return x.SecurityType
	}
	return ""
}

func (x *DownloadConfig) GetSecuritySettings() *anypb.Any {
	if x != nil {
		return x.SecuritySettings
	}
	return nil
}

func (x *DownloadConfig) GetSocketSettings() *internet.SocketConfig {
	if x != nil {
		return x.SocketSettings
	}
	return nil
}

type Config struct {
	state                protoimpl.MessageState `protogen:"open.v1"`
	Host                 string                 `protobuf:"bytes,1,opt,name=host,proto3" json:"host,omitempty"`
	Path                 string                 `protobuf:"bytes,2,opt,name=path,proto3" json:"path,omitempty"`
	Mode                 string                 `protobuf:"bytes,3,opt,name=mode,proto3" json:"mode,omitempty"`
	Headers              map[string]string      `protobuf:"bytes,4,rep,name=headers,proto3" json:"headers,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	XPaddingBytes        string                 `protobuf:"bytes,5,opt,name=xPaddingBytes,proto3" json:"xPaddingBytes,omitempty"`
	NoGRPCHeader         bool                   `protobuf:"varint,6,opt,name=noGRPCHeader,proto3" json:"noGRPCHeader,omitempty"`
	ScMaxEachPostBytes   string                 `protobuf:"bytes,7,opt,name=scMaxEachPostBytes,proto3" json:"scMaxEachPostBytes,omitempty"`
	ScMinPostsIntervalMs string                 `protobuf:"bytes,8,opt,name=scMinPostsIntervalMs,proto3" json:"scMinPostsIntervalMs,omitempty"`
	ScMaxBufferedPosts   int64                  `protobuf:"varint,9,opt,name=scMaxBufferedPosts,proto3" json:"scMaxBufferedPosts,omitempty"`
	Xmux                 *XmuxConfig            `protobuf:"bytes,10,opt,name=xmux,proto3,oneof" json:"xmux,omitempty"`
	DownloadSettings     *DownloadConfig        `protobuf:"bytes,11,opt,name=downloadSettings,proto3,oneof" json:"downloadSettings,omitempty"`
	UseBrowserForwarding bool                   `protobuf:"varint,99,opt,name=use_browser_forwarding,json=useBrowserForwarding,proto3" json:"use_browser_forwarding,omitempty"`
	unknownFields        protoimpl.UnknownFields
	sizeCache            protoimpl.SizeCache
}

func (x *Config) Reset() {
	*x = Config{}
	mi := &file_transport_internet_splithttp_config_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Config) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config) ProtoMessage() {}

func (x *Config) ProtoReflect() protoreflect.Message {
	mi := &file_transport_internet_splithttp_config_proto_msgTypes[2]
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
	return file_transport_internet_splithttp_config_proto_rawDescGZIP(), []int{2}
}

func (x *Config) GetHost() string {
	if x != nil {
		return x.Host
	}
	return ""
}

func (x *Config) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

func (x *Config) GetMode() string {
	if x != nil {
		return x.Mode
	}
	return ""
}

func (x *Config) GetHeaders() map[string]string {
	if x != nil {
		return x.Headers
	}
	return nil
}

func (x *Config) GetXPaddingBytes() string {
	if x != nil {
		return x.XPaddingBytes
	}
	return ""
}

func (x *Config) GetNoGRPCHeader() bool {
	if x != nil {
		return x.NoGRPCHeader
	}
	return false
}

func (x *Config) GetScMaxEachPostBytes() string {
	if x != nil {
		return x.ScMaxEachPostBytes
	}
	return ""
}

func (x *Config) GetScMinPostsIntervalMs() string {
	if x != nil {
		return x.ScMinPostsIntervalMs
	}
	return ""
}

func (x *Config) GetScMaxBufferedPosts() int64 {
	if x != nil {
		return x.ScMaxBufferedPosts
	}
	return 0
}

func (x *Config) GetXmux() *XmuxConfig {
	if x != nil {
		return x.Xmux
	}
	return nil
}

func (x *Config) GetDownloadSettings() *DownloadConfig {
	if x != nil {
		return x.DownloadSettings
	}
	return nil
}

func (x *Config) GetUseBrowserForwarding() bool {
	if x != nil {
		return x.UseBrowserForwarding
	}
	return false
}

var File_transport_internet_splithttp_config_proto protoreflect.FileDescriptor

const file_transport_internet_splithttp_config_proto_rawDesc = "" +
	"\n" +
	")transport/internet/splithttp/config.proto\x12\"v2ray.transport.internet.splithttp\x1a\x18common/net/address.proto\x1a common/protoext/extensions.proto\x1a\x19google/protobuf/any.proto\x1a\x1ftransport/internet/config.proto\"\xdc\x01\n" +
	"\n" +
	"XmuxConfig\x12&\n" +
	"\x0emaxConcurrency\x18\x01 \x01(\tR\x0emaxConcurrency\x12&\n" +
	"\x0emaxConnections\x18\x02 \x01(\tR\x0emaxConnections\x12&\n" +
	"\x0ecMaxReuseTimes\x18\x03 \x01(\tR\x0ecMaxReuseTimes\x12*\n" +
	"\x10hMaxRequestTimes\x18\x04 \x01(\tR\x10hMaxRequestTimes\x12*\n" +
	"\x10hMaxReusableSecs\x18\x05 \x01(\tR\x10hMaxReusableSecs\"\xe4\x02\n" +
	"\x0eDownloadConfig\x12;\n" +
	"\aaddress\x18\x01 \x01(\v2!.v2ray.core.common.net.IPOrDomainR\aaddress\x12\x12\n" +
	"\x04port\x18\x02 \x01(\rR\x04port\x12C\n" +
	"\x12transport_settings\x18\x03 \x01(\v2\x14.google.protobuf.AnyR\x11transportSettings\x12#\n" +
	"\rsecurity_type\x18\x04 \x01(\tR\fsecurityType\x12A\n" +
	"\x11security_settings\x18\x05 \x01(\v2\x14.google.protobuf.AnyR\x10securitySettings\x12T\n" +
	"\x0fsocket_settings\x18\x06 \x01(\v2+.v2ray.core.transport.internet.SocketConfigR\x0esocketSettings\"\xcf\x05\n" +
	"\x06Config\x12\x12\n" +
	"\x04host\x18\x01 \x01(\tR\x04host\x12\x12\n" +
	"\x04path\x18\x02 \x01(\tR\x04path\x12\x12\n" +
	"\x04mode\x18\x03 \x01(\tR\x04mode\x12Q\n" +
	"\aheaders\x18\x04 \x03(\v27.v2ray.transport.internet.splithttp.Config.HeadersEntryR\aheaders\x12$\n" +
	"\rxPaddingBytes\x18\x05 \x01(\tR\rxPaddingBytes\x12\"\n" +
	"\fnoGRPCHeader\x18\x06 \x01(\bR\fnoGRPCHeader\x12.\n" +
	"\x12scMaxEachPostBytes\x18\a \x01(\tR\x12scMaxEachPostBytes\x122\n" +
	"\x14scMinPostsIntervalMs\x18\b \x01(\tR\x14scMinPostsIntervalMs\x12.\n" +
	"\x12scMaxBufferedPosts\x18\t \x01(\x03R\x12scMaxBufferedPosts\x12G\n" +
	"\x04xmux\x18\n" +
	" \x01(\v2..v2ray.transport.internet.splithttp.XmuxConfigH\x00R\x04xmux\x88\x01\x01\x12c\n" +
	"\x10downloadSettings\x18\v \x01(\v22.v2ray.transport.internet.splithttp.DownloadConfigH\x01R\x10downloadSettings\x88\x01\x01\x124\n" +
	"\x16use_browser_forwarding\x18c \x01(\bR\x14useBrowserForwarding\x1a:\n" +
	"\fHeadersEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01:\x1a\x82\xb5\x18\x16\n" +
	"\ttransport\x12\tsplithttpB\a\n" +
	"\x05_xmuxB\x13\n" +
	"\x11_downloadSettingsB\x8c\x01\n" +
	"&com.v2ray.transport.internet.splithttpP\x01Z;github.com/v2fly/v2ray-core/v5/transport/internet/splithttp\xaa\x02\"V2Ray.Transport.Internet.SplitHttpb\x06proto3"

var (
	file_transport_internet_splithttp_config_proto_rawDescOnce sync.Once
	file_transport_internet_splithttp_config_proto_rawDescData []byte
)

func file_transport_internet_splithttp_config_proto_rawDescGZIP() []byte {
	file_transport_internet_splithttp_config_proto_rawDescOnce.Do(func() {
		file_transport_internet_splithttp_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_transport_internet_splithttp_config_proto_rawDesc), len(file_transport_internet_splithttp_config_proto_rawDesc)))
	})
	return file_transport_internet_splithttp_config_proto_rawDescData
}

var file_transport_internet_splithttp_config_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_transport_internet_splithttp_config_proto_goTypes = []any{
	(*XmuxConfig)(nil),            // 0: v2ray.transport.internet.splithttp.XmuxConfig
	(*DownloadConfig)(nil),        // 1: v2ray.transport.internet.splithttp.DownloadConfig
	(*Config)(nil),                // 2: v2ray.transport.internet.splithttp.Config
	nil,                           // 3: v2ray.transport.internet.splithttp.Config.HeadersEntry
	(*net.IPOrDomain)(nil),        // 4: v2ray.core.common.net.IPOrDomain
	(*anypb.Any)(nil),             // 5: google.protobuf.Any
	(*internet.SocketConfig)(nil), // 6: v2ray.core.transport.internet.SocketConfig
}
var file_transport_internet_splithttp_config_proto_depIdxs = []int32{
	4, // 0: v2ray.transport.internet.splithttp.DownloadConfig.address:type_name -> v2ray.core.common.net.IPOrDomain
	5, // 1: v2ray.transport.internet.splithttp.DownloadConfig.transport_settings:type_name -> google.protobuf.Any
	5, // 2: v2ray.transport.internet.splithttp.DownloadConfig.security_settings:type_name -> google.protobuf.Any
	6, // 3: v2ray.transport.internet.splithttp.DownloadConfig.socket_settings:type_name -> v2ray.core.transport.internet.SocketConfig
	3, // 4: v2ray.transport.internet.splithttp.Config.headers:type_name -> v2ray.transport.internet.splithttp.Config.HeadersEntry
	0, // 5: v2ray.transport.internet.splithttp.Config.xmux:type_name -> v2ray.transport.internet.splithttp.XmuxConfig
	1, // 6: v2ray.transport.internet.splithttp.Config.downloadSettings:type_name -> v2ray.transport.internet.splithttp.DownloadConfig
	7, // [7:7] is the sub-list for method output_type
	7, // [7:7] is the sub-list for method input_type
	7, // [7:7] is the sub-list for extension type_name
	7, // [7:7] is the sub-list for extension extendee
	0, // [0:7] is the sub-list for field type_name
}

func init() { file_transport_internet_splithttp_config_proto_init() }
func file_transport_internet_splithttp_config_proto_init() {
	if File_transport_internet_splithttp_config_proto != nil {
		return
	}
	file_transport_internet_splithttp_config_proto_msgTypes[2].OneofWrappers = []any{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_transport_internet_splithttp_config_proto_rawDesc), len(file_transport_internet_splithttp_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_transport_internet_splithttp_config_proto_goTypes,
		DependencyIndexes: file_transport_internet_splithttp_config_proto_depIdxs,
		MessageInfos:      file_transport_internet_splithttp_config_proto_msgTypes,
	}.Build()
	File_transport_internet_splithttp_config_proto = out.File
	file_transport_internet_splithttp_config_proto_goTypes = nil
	file_transport_internet_splithttp_config_proto_depIdxs = nil
}
