package ssh

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

type Config struct {
	state             protoimpl.MessageState `protogen:"open.v1"`
	Address           *net.IPOrDomain        `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Port              uint32                 `protobuf:"varint,2,opt,name=port,proto3" json:"port,omitempty"`
	User              string                 `protobuf:"bytes,3,opt,name=user,proto3" json:"user,omitempty"`
	Password          string                 `protobuf:"bytes,4,opt,name=password,proto3" json:"password,omitempty"`
	PrivateKey        string                 `protobuf:"bytes,5,opt,name=private_key,json=privateKey,proto3" json:"private_key,omitempty"`
	PublicKey         string                 `protobuf:"bytes,6,opt,name=public_key,json=publicKey,proto3" json:"public_key,omitempty"`
	HostKeyAlgorithms []string               `protobuf:"bytes,7,rep,name=host_key_algorithms,json=hostKeyAlgorithms,proto3" json:"host_key_algorithms,omitempty"`
	ClientVersion     string                 `protobuf:"bytes,8,opt,name=client_version,json=clientVersion,proto3" json:"client_version,omitempty"`
	UserLevel         uint32                 `protobuf:"varint,9,opt,name=user_level,json=userLevel,proto3" json:"user_level,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *Config) Reset() {
	*x = Config{}
	mi := &file_proxy_ssh_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Config) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config) ProtoMessage() {}

func (x *Config) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_ssh_config_proto_msgTypes[0]
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
	return file_proxy_ssh_config_proto_rawDescGZIP(), []int{0}
}

func (x *Config) GetAddress() *net.IPOrDomain {
	if x != nil {
		return x.Address
	}
	return nil
}

func (x *Config) GetPort() uint32 {
	if x != nil {
		return x.Port
	}
	return 0
}

func (x *Config) GetUser() string {
	if x != nil {
		return x.User
	}
	return ""
}

func (x *Config) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

func (x *Config) GetPrivateKey() string {
	if x != nil {
		return x.PrivateKey
	}
	return ""
}

func (x *Config) GetPublicKey() string {
	if x != nil {
		return x.PublicKey
	}
	return ""
}

func (x *Config) GetHostKeyAlgorithms() []string {
	if x != nil {
		return x.HostKeyAlgorithms
	}
	return nil
}

func (x *Config) GetClientVersion() string {
	if x != nil {
		return x.ClientVersion
	}
	return ""
}

func (x *Config) GetUserLevel() uint32 {
	if x != nil {
		return x.UserLevel
	}
	return 0
}

var File_proxy_ssh_config_proto protoreflect.FileDescriptor

const file_proxy_ssh_config_proto_rawDesc = "" +
	"\n" +
	"\x16proxy/ssh/config.proto\x12\x14v2ray.core.proxy.ssh\x1a common/protoext/extensions.proto\x1a\x18common/net/address.proto\"\xd4\x02\n" +
	"\x06Config\x12;\n" +
	"\aaddress\x18\x01 \x01(\v2!.v2ray.core.common.net.IPOrDomainR\aaddress\x12\x12\n" +
	"\x04port\x18\x02 \x01(\rR\x04port\x12\x12\n" +
	"\x04user\x18\x03 \x01(\tR\x04user\x12\x1a\n" +
	"\bpassword\x18\x04 \x01(\tR\bpassword\x12\x1f\n" +
	"\vprivate_key\x18\x05 \x01(\tR\n" +
	"privateKey\x12\x1d\n" +
	"\n" +
	"public_key\x18\x06 \x01(\tR\tpublicKey\x12.\n" +
	"\x13host_key_algorithms\x18\a \x03(\tR\x11hostKeyAlgorithms\x12%\n" +
	"\x0eclient_version\x18\b \x01(\tR\rclientVersion\x12\x1d\n" +
	"\n" +
	"user_level\x18\t \x01(\rR\tuserLevel:\x13\x82\xb5\x18\x0f\n" +
	"\boutbound\x12\x03sshB]\n" +
	"\x18com.v2ray.core.proxy.sshP\x01Z(github.com/v2fly/v2ray-core/v5/proxy/ssh\xaa\x02\x14V2Ray.Core.Proxy.SSHb\x06proto3"

var (
	file_proxy_ssh_config_proto_rawDescOnce sync.Once
	file_proxy_ssh_config_proto_rawDescData []byte
)

func file_proxy_ssh_config_proto_rawDescGZIP() []byte {
	file_proxy_ssh_config_proto_rawDescOnce.Do(func() {
		file_proxy_ssh_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proxy_ssh_config_proto_rawDesc), len(file_proxy_ssh_config_proto_rawDesc)))
	})
	return file_proxy_ssh_config_proto_rawDescData
}

var file_proxy_ssh_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_proxy_ssh_config_proto_goTypes = []any{
	(*Config)(nil),         // 0: v2ray.core.proxy.ssh.Config
	(*net.IPOrDomain)(nil), // 1: v2ray.core.common.net.IPOrDomain
}
var file_proxy_ssh_config_proto_depIdxs = []int32{
	1, // 0: v2ray.core.proxy.ssh.Config.address:type_name -> v2ray.core.common.net.IPOrDomain
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_proxy_ssh_config_proto_init() }
func file_proxy_ssh_config_proto_init() {
	if File_proxy_ssh_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proxy_ssh_config_proto_rawDesc), len(file_proxy_ssh_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proxy_ssh_config_proto_goTypes,
		DependencyIndexes: file_proxy_ssh_config_proto_depIdxs,
		MessageInfos:      file_proxy_ssh_config_proto_msgTypes,
	}.Build()
	File_proxy_ssh_config_proto = out.File
	file_proxy_ssh_config_proto_goTypes = nil
	file_proxy_ssh_config_proto_depIdxs = nil
}
