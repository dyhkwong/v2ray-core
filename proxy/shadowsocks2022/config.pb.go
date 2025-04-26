package shadowsocks2022

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
	state            protoimpl.MessageState `protogen:"open.v1"`
	Method           string                 `protobuf:"bytes,1,opt,name=method,proto3" json:"method,omitempty"`
	Psk              []byte                 `protobuf:"bytes,2,opt,name=psk,proto3" json:"psk,omitempty"`
	Ipsk             [][]byte               `protobuf:"bytes,4,rep,name=ipsk,proto3" json:"ipsk,omitempty"`
	Address          *net.IPOrDomain        `protobuf:"bytes,5,opt,name=address,proto3" json:"address,omitempty"`
	Port             uint32                 `protobuf:"varint,6,opt,name=port,proto3" json:"port,omitempty"`
	Plugin           string                 `protobuf:"bytes,7,opt,name=plugin,proto3" json:"plugin,omitempty"`
	PluginOpts       string                 `protobuf:"bytes,8,opt,name=plugin_opts,json=pluginOpts,proto3" json:"plugin_opts,omitempty"`
	PluginArgs       []string               `protobuf:"bytes,9,rep,name=plugin_args,json=pluginArgs,proto3" json:"plugin_args,omitempty"`
	PluginWorkingDir string                 `protobuf:"bytes,10,opt,name=plugin_working_dir,json=pluginWorkingDir,proto3" json:"plugin_working_dir,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *ClientConfig) Reset() {
	*x = ClientConfig{}
	mi := &file_proxy_shadowsocks2022_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ClientConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClientConfig) ProtoMessage() {}

func (x *ClientConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_shadowsocks2022_config_proto_msgTypes[0]
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
	return file_proxy_shadowsocks2022_config_proto_rawDescGZIP(), []int{0}
}

func (x *ClientConfig) GetMethod() string {
	if x != nil {
		return x.Method
	}
	return ""
}

func (x *ClientConfig) GetPsk() []byte {
	if x != nil {
		return x.Psk
	}
	return nil
}

func (x *ClientConfig) GetIpsk() [][]byte {
	if x != nil {
		return x.Ipsk
	}
	return nil
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

func (x *ClientConfig) GetPlugin() string {
	if x != nil {
		return x.Plugin
	}
	return ""
}

func (x *ClientConfig) GetPluginOpts() string {
	if x != nil {
		return x.PluginOpts
	}
	return ""
}

func (x *ClientConfig) GetPluginArgs() []string {
	if x != nil {
		return x.PluginArgs
	}
	return nil
}

func (x *ClientConfig) GetPluginWorkingDir() string {
	if x != nil {
		return x.PluginWorkingDir
	}
	return ""
}

var File_proxy_shadowsocks2022_config_proto protoreflect.FileDescriptor

const file_proxy_shadowsocks2022_config_proto_rawDesc = "" +
	"\n" +
	"\"proxy/shadowsocks2022/config.proto\x12 v2ray.core.proxy.shadowsocks2022\x1a\x18common/net/address.proto\x1a common/protoext/extensions.proto\"\xc6\x02\n" +
	"\fClientConfig\x12\x16\n" +
	"\x06method\x18\x01 \x01(\tR\x06method\x12\x10\n" +
	"\x03psk\x18\x02 \x01(\fR\x03psk\x12\x12\n" +
	"\x04ipsk\x18\x04 \x03(\fR\x04ipsk\x12;\n" +
	"\aaddress\x18\x05 \x01(\v2!.v2ray.core.common.net.IPOrDomainR\aaddress\x12\x12\n" +
	"\x04port\x18\x06 \x01(\rR\x04port\x12\x16\n" +
	"\x06plugin\x18\a \x01(\tR\x06plugin\x12\x1f\n" +
	"\vplugin_opts\x18\b \x01(\tR\n" +
	"pluginOpts\x12\x1f\n" +
	"\vplugin_args\x18\t \x03(\tR\n" +
	"pluginArgs\x12,\n" +
	"\x12plugin_working_dir\x18\n" +
	" \x01(\tR\x10pluginWorkingDir:\x1f\x82\xb5\x18\x1b\n" +
	"\boutbound\x12\x0fshadowsocks2022B\x81\x01\n" +
	"$com.v2ray.core.proxy.shadowsocks2022P\x01Z4github.com/v2fly/v2ray-core/v5/proxy/shadowsocks2022\xaa\x02 V2Ray.Core.Proxy.Shadowsocks2022b\x06proto3"

var (
	file_proxy_shadowsocks2022_config_proto_rawDescOnce sync.Once
	file_proxy_shadowsocks2022_config_proto_rawDescData []byte
)

func file_proxy_shadowsocks2022_config_proto_rawDescGZIP() []byte {
	file_proxy_shadowsocks2022_config_proto_rawDescOnce.Do(func() {
		file_proxy_shadowsocks2022_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proxy_shadowsocks2022_config_proto_rawDesc), len(file_proxy_shadowsocks2022_config_proto_rawDesc)))
	})
	return file_proxy_shadowsocks2022_config_proto_rawDescData
}

var file_proxy_shadowsocks2022_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_proxy_shadowsocks2022_config_proto_goTypes = []any{
	(*ClientConfig)(nil),   // 0: v2ray.core.proxy.shadowsocks2022.ClientConfig
	(*net.IPOrDomain)(nil), // 1: v2ray.core.common.net.IPOrDomain
}
var file_proxy_shadowsocks2022_config_proto_depIdxs = []int32{
	1, // 0: v2ray.core.proxy.shadowsocks2022.ClientConfig.address:type_name -> v2ray.core.common.net.IPOrDomain
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_proxy_shadowsocks2022_config_proto_init() }
func file_proxy_shadowsocks2022_config_proto_init() {
	if File_proxy_shadowsocks2022_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proxy_shadowsocks2022_config_proto_rawDesc), len(file_proxy_shadowsocks2022_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proxy_shadowsocks2022_config_proto_goTypes,
		DependencyIndexes: file_proxy_shadowsocks2022_config_proto_depIdxs,
		MessageInfos:      file_proxy_shadowsocks2022_config_proto_msgTypes,
	}.Build()
	File_proxy_shadowsocks2022_config_proto = out.File
	file_proxy_shadowsocks2022_config_proto_goTypes = nil
	file_proxy_shadowsocks2022_config_proto_depIdxs = nil
}
