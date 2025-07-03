package blackhole

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

type NoneResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *NoneResponse) Reset() {
	*x = NoneResponse{}
	mi := &file_proxy_blackhole_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NoneResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NoneResponse) ProtoMessage() {}

func (x *NoneResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_blackhole_config_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NoneResponse.ProtoReflect.Descriptor instead.
func (*NoneResponse) Descriptor() ([]byte, []int) {
	return file_proxy_blackhole_config_proto_rawDescGZIP(), []int{0}
}

type HTTPResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HTTPResponse) Reset() {
	*x = HTTPResponse{}
	mi := &file_proxy_blackhole_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HTTPResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HTTPResponse) ProtoMessage() {}

func (x *HTTPResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_blackhole_config_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HTTPResponse.ProtoReflect.Descriptor instead.
func (*HTTPResponse) Descriptor() ([]byte, []int) {
	return file_proxy_blackhole_config_proto_rawDescGZIP(), []int{1}
}

type Config struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Response      *serial.TypedMessage   `protobuf:"bytes,1,opt,name=response,proto3" json:"response,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Config) Reset() {
	*x = Config{}
	mi := &file_proxy_blackhole_config_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Config) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config) ProtoMessage() {}

func (x *Config) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_blackhole_config_proto_msgTypes[2]
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
	return file_proxy_blackhole_config_proto_rawDescGZIP(), []int{2}
}

func (x *Config) GetResponse() *serial.TypedMessage {
	if x != nil {
		return x.Response
	}
	return nil
}

var File_proxy_blackhole_config_proto protoreflect.FileDescriptor

const file_proxy_blackhole_config_proto_rawDesc = "" +
	"\n" +
	"\x1cproxy/blackhole/config.proto\x12\x1av2ray.core.proxy.blackhole\x1a!common/serial/typed_message.proto\"\x0e\n" +
	"\fNoneResponse\"\x0e\n" +
	"\fHTTPResponse\"L\n" +
	"\x06Config\x12B\n" +
	"\bresponse\x18\x01 \x01(\v2&.v2ray.core.common.serial.TypedMessageR\bresponseBo\n" +
	"\x1ecom.v2ray.core.proxy.blackholeP\x01Z.github.com/v2fly/v2ray-core/v4/proxy/blackhole\xaa\x02\x1aV2Ray.Core.Proxy.Blackholeb\x06proto3"

var (
	file_proxy_blackhole_config_proto_rawDescOnce sync.Once
	file_proxy_blackhole_config_proto_rawDescData []byte
)

func file_proxy_blackhole_config_proto_rawDescGZIP() []byte {
	file_proxy_blackhole_config_proto_rawDescOnce.Do(func() {
		file_proxy_blackhole_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proxy_blackhole_config_proto_rawDesc), len(file_proxy_blackhole_config_proto_rawDesc)))
	})
	return file_proxy_blackhole_config_proto_rawDescData
}

var file_proxy_blackhole_config_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_proxy_blackhole_config_proto_goTypes = []any{
	(*NoneResponse)(nil),        // 0: v2ray.core.proxy.blackhole.NoneResponse
	(*HTTPResponse)(nil),        // 1: v2ray.core.proxy.blackhole.HTTPResponse
	(*Config)(nil),              // 2: v2ray.core.proxy.blackhole.Config
	(*serial.TypedMessage)(nil), // 3: v2ray.core.common.serial.TypedMessage
}
var file_proxy_blackhole_config_proto_depIdxs = []int32{
	3, // 0: v2ray.core.proxy.blackhole.Config.response:type_name -> v2ray.core.common.serial.TypedMessage
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_proxy_blackhole_config_proto_init() }
func file_proxy_blackhole_config_proto_init() {
	if File_proxy_blackhole_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proxy_blackhole_config_proto_rawDesc), len(file_proxy_blackhole_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proxy_blackhole_config_proto_goTypes,
		DependencyIndexes: file_proxy_blackhole_config_proto_depIdxs,
		MessageInfos:      file_proxy_blackhole_config_proto_msgTypes,
	}.Build()
	File_proxy_blackhole_config_proto = out.File
	file_proxy_blackhole_config_proto_goTypes = nil
	file_proxy_blackhole_config_proto_depIdxs = nil
}
