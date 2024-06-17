package splithttp

import (
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
	UseBrowserForwarding bool                   `protobuf:"varint,99,opt,name=use_browser_forwarding,json=useBrowserForwarding,proto3" json:"use_browser_forwarding,omitempty"`
	unknownFields        protoimpl.UnknownFields
	sizeCache            protoimpl.SizeCache
}

func (x *Config) Reset() {
	*x = Config{}
	mi := &file_transport_internet_splithttp_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Config) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config) ProtoMessage() {}

func (x *Config) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use Config.ProtoReflect.Descriptor instead.
func (*Config) Descriptor() ([]byte, []int) {
	return file_transport_internet_splithttp_config_proto_rawDescGZIP(), []int{0}
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

func (x *Config) GetUseBrowserForwarding() bool {
	if x != nil {
		return x.UseBrowserForwarding
	}
	return false
}

var File_transport_internet_splithttp_config_proto protoreflect.FileDescriptor

const file_transport_internet_splithttp_config_proto_rawDesc = "" +
	"\n" +
	")transport/internet/splithttp/config.proto\x12\"v2ray.transport.internet.splithttp\x1a common/protoext/extensions.proto\"\x83\x04\n" +
	"\x06Config\x12\x12\n" +
	"\x04host\x18\x01 \x01(\tR\x04host\x12\x12\n" +
	"\x04path\x18\x02 \x01(\tR\x04path\x12\x12\n" +
	"\x04mode\x18\x03 \x01(\tR\x04mode\x12Q\n" +
	"\aheaders\x18\x04 \x03(\v27.v2ray.transport.internet.splithttp.Config.HeadersEntryR\aheaders\x12$\n" +
	"\rxPaddingBytes\x18\x05 \x01(\tR\rxPaddingBytes\x12\"\n" +
	"\fnoGRPCHeader\x18\x06 \x01(\bR\fnoGRPCHeader\x12.\n" +
	"\x12scMaxEachPostBytes\x18\a \x01(\tR\x12scMaxEachPostBytes\x122\n" +
	"\x14scMinPostsIntervalMs\x18\b \x01(\tR\x14scMinPostsIntervalMs\x12.\n" +
	"\x12scMaxBufferedPosts\x18\t \x01(\x03R\x12scMaxBufferedPosts\x124\n" +
	"\x16use_browser_forwarding\x18c \x01(\bR\x14useBrowserForwarding\x1a:\n" +
	"\fHeadersEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01:\x1a\x82\xb5\x18\x16\n" +
	"\ttransport\x12\tsplithttpB\x8c\x01\n" +
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

var file_transport_internet_splithttp_config_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_transport_internet_splithttp_config_proto_goTypes = []any{
	(*Config)(nil), // 0: v2ray.transport.internet.splithttp.Config
	nil,            // 1: v2ray.transport.internet.splithttp.Config.HeadersEntry
}
var file_transport_internet_splithttp_config_proto_depIdxs = []int32{
	1, // 0: v2ray.transport.internet.splithttp.Config.headers:type_name -> v2ray.transport.internet.splithttp.Config.HeadersEntry
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_transport_internet_splithttp_config_proto_init() }
func file_transport_internet_splithttp_config_proto_init() {
	if File_transport_internet_splithttp_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_transport_internet_splithttp_config_proto_rawDesc), len(file_transport_internet_splithttp_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
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
