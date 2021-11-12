package shadowsocks

import (
	net "github.com/v2fly/v2ray-core/v5/common/net"
	packetaddr "github.com/v2fly/v2ray-core/v5/common/net/packetaddr"
	protocol "github.com/v2fly/v2ray-core/v5/common/protocol"
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

type CipherType int32

const (
	CipherType_UNKNOWN            CipherType = 0
	CipherType_AES_128_GCM        CipherType = 1
	CipherType_AES_256_GCM        CipherType = 2
	CipherType_CHACHA20_POLY1305  CipherType = 3
	CipherType_NONE               CipherType = 4
	CipherType_XCHACHA20_POLY1305 CipherType = 5
	CipherType_AES_192_GCM        CipherType = 6
	CipherType_AES_128_CTR        CipherType = 7
	CipherType_AES_192_CTR        CipherType = 8
	CipherType_AES_256_CTR        CipherType = 9
	CipherType_AES_128_CFB        CipherType = 10
	CipherType_AES_192_CFB        CipherType = 11
	CipherType_AES_256_CFB        CipherType = 12
	CipherType_AES_128_CFB8       CipherType = 13
	CipherType_AES_192_CFB8       CipherType = 14
	CipherType_AES_256_CFB8       CipherType = 15
	CipherType_AES_128_OFB        CipherType = 16
	CipherType_AES_192_OFB        CipherType = 17
	CipherType_AES_256_OFB        CipherType = 18
	CipherType_RC4                CipherType = 19
	CipherType_RC4_MD5            CipherType = 20
	CipherType_RC4_MD5_6          CipherType = 21
	CipherType_BF_CFB             CipherType = 22
	CipherType_CAST5_CFB          CipherType = 23
	CipherType_DES_CFB            CipherType = 24
	CipherType_RC2_CFB            CipherType = 25
	CipherType_SEED_CFB           CipherType = 26
	CipherType_CAMELLIA_128_CFB   CipherType = 27
	CipherType_CAMELLIA_192_CFB   CipherType = 28
	CipherType_CAMELLIA_256_CFB   CipherType = 29
	CipherType_CAMELLIA_128_CFB8  CipherType = 30
	CipherType_CAMELLIA_192_CFB8  CipherType = 31
	CipherType_CAMELLIA_256_CFB8  CipherType = 32
	CipherType_SALSA20            CipherType = 33
	CipherType_CHACHA20           CipherType = 34
	CipherType_CHACHA20_IETF      CipherType = 35
	CipherType_XCHACHA20          CipherType = 36
	CipherType_TABLE              CipherType = 37
)

// Enum value maps for CipherType.
var (
	CipherType_name = map[int32]string{
		0:  "UNKNOWN",
		1:  "AES_128_GCM",
		2:  "AES_256_GCM",
		3:  "CHACHA20_POLY1305",
		4:  "NONE",
		5:  "XCHACHA20_POLY1305",
		6:  "AES_192_GCM",
		7:  "AES_128_CTR",
		8:  "AES_192_CTR",
		9:  "AES_256_CTR",
		10: "AES_128_CFB",
		11: "AES_192_CFB",
		12: "AES_256_CFB",
		13: "AES_128_CFB8",
		14: "AES_192_CFB8",
		15: "AES_256_CFB8",
		16: "AES_128_OFB",
		17: "AES_192_OFB",
		18: "AES_256_OFB",
		19: "RC4",
		20: "RC4_MD5",
		21: "RC4_MD5_6",
		22: "BF_CFB",
		23: "CAST5_CFB",
		24: "DES_CFB",
		25: "RC2_CFB",
		26: "SEED_CFB",
		27: "CAMELLIA_128_CFB",
		28: "CAMELLIA_192_CFB",
		29: "CAMELLIA_256_CFB",
		30: "CAMELLIA_128_CFB8",
		31: "CAMELLIA_192_CFB8",
		32: "CAMELLIA_256_CFB8",
		33: "SALSA20",
		34: "CHACHA20",
		35: "CHACHA20_IETF",
		36: "XCHACHA20",
		37: "TABLE",
	}
	CipherType_value = map[string]int32{
		"UNKNOWN":            0,
		"AES_128_GCM":        1,
		"AES_256_GCM":        2,
		"CHACHA20_POLY1305":  3,
		"NONE":               4,
		"XCHACHA20_POLY1305": 5,
		"AES_192_GCM":        6,
		"AES_128_CTR":        7,
		"AES_192_CTR":        8,
		"AES_256_CTR":        9,
		"AES_128_CFB":        10,
		"AES_192_CFB":        11,
		"AES_256_CFB":        12,
		"AES_128_CFB8":       13,
		"AES_192_CFB8":       14,
		"AES_256_CFB8":       15,
		"AES_128_OFB":        16,
		"AES_192_OFB":        17,
		"AES_256_OFB":        18,
		"RC4":                19,
		"RC4_MD5":            20,
		"RC4_MD5_6":          21,
		"BF_CFB":             22,
		"CAST5_CFB":          23,
		"DES_CFB":            24,
		"RC2_CFB":            25,
		"SEED_CFB":           26,
		"CAMELLIA_128_CFB":   27,
		"CAMELLIA_192_CFB":   28,
		"CAMELLIA_256_CFB":   29,
		"CAMELLIA_128_CFB8":  30,
		"CAMELLIA_192_CFB8":  31,
		"CAMELLIA_256_CFB8":  32,
		"SALSA20":            33,
		"CHACHA20":           34,
		"CHACHA20_IETF":      35,
		"XCHACHA20":          36,
		"TABLE":              37,
	}
)

func (x CipherType) Enum() *CipherType {
	p := new(CipherType)
	*p = x
	return p
}

func (x CipherType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (CipherType) Descriptor() protoreflect.EnumDescriptor {
	return file_proxy_shadowsocks_config_proto_enumTypes[0].Descriptor()
}

func (CipherType) Type() protoreflect.EnumType {
	return &file_proxy_shadowsocks_config_proto_enumTypes[0]
}

func (x CipherType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use CipherType.Descriptor instead.
func (CipherType) EnumDescriptor() ([]byte, []int) {
	return file_proxy_shadowsocks_config_proto_rawDescGZIP(), []int{0}
}

type Account struct {
	state                          protoimpl.MessageState `protogen:"open.v1"`
	Password                       string                 `protobuf:"bytes,1,opt,name=password,proto3" json:"password,omitempty"`
	CipherType                     CipherType             `protobuf:"varint,2,opt,name=cipher_type,json=cipherType,proto3,enum=v2ray.core.proxy.shadowsocks.CipherType" json:"cipher_type,omitempty"`
	IvCheck                        bool                   `protobuf:"varint,3,opt,name=iv_check,json=ivCheck,proto3" json:"iv_check,omitempty"`
	ExperimentReducedIvHeadEntropy bool                   `protobuf:"varint,90001,opt,name=experiment_reduced_iv_head_entropy,json=experimentReducedIvHeadEntropy,proto3" json:"experiment_reduced_iv_head_entropy,omitempty"`
	unknownFields                  protoimpl.UnknownFields
	sizeCache                      protoimpl.SizeCache
}

func (x *Account) Reset() {
	*x = Account{}
	mi := &file_proxy_shadowsocks_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Account) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Account) ProtoMessage() {}

func (x *Account) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_shadowsocks_config_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Account.ProtoReflect.Descriptor instead.
func (*Account) Descriptor() ([]byte, []int) {
	return file_proxy_shadowsocks_config_proto_rawDescGZIP(), []int{0}
}

func (x *Account) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

func (x *Account) GetCipherType() CipherType {
	if x != nil {
		return x.CipherType
	}
	return CipherType_UNKNOWN
}

func (x *Account) GetIvCheck() bool {
	if x != nil {
		return x.IvCheck
	}
	return false
}

func (x *Account) GetExperimentReducedIvHeadEntropy() bool {
	if x != nil {
		return x.ExperimentReducedIvHeadEntropy
	}
	return false
}

type ServerConfig struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// UdpEnabled specified whether or not to enable UDP for Shadowsocks.
	// Deprecated. Use 'network' field.
	//
	// Deprecated: Marked as deprecated in proxy/shadowsocks/config.proto.
	UdpEnabled       bool                      `protobuf:"varint,1,opt,name=udp_enabled,json=udpEnabled,proto3" json:"udp_enabled,omitempty"`
	User             *protocol.User            `protobuf:"bytes,2,opt,name=user,proto3" json:"user,omitempty"`
	Network          []net.Network             `protobuf:"varint,3,rep,packed,name=network,proto3,enum=v2ray.core.common.net.Network" json:"network,omitempty"`
	PacketEncoding   packetaddr.PacketAddrType `protobuf:"varint,4,opt,name=packet_encoding,json=packetEncoding,proto3,enum=v2ray.core.net.packetaddr.PacketAddrType" json:"packet_encoding,omitempty"`
	Plugin           string                    `protobuf:"bytes,5,opt,name=plugin,proto3" json:"plugin,omitempty"`
	PluginOpts       string                    `protobuf:"bytes,6,opt,name=plugin_opts,json=pluginOpts,proto3" json:"plugin_opts,omitempty"`
	PluginArgs       []string                  `protobuf:"bytes,7,rep,name=plugin_args,json=pluginArgs,proto3" json:"plugin_args,omitempty"`
	PluginWorkingDir string                    `protobuf:"bytes,8,opt,name=plugin_working_dir,json=pluginWorkingDir,proto3" json:"plugin_working_dir,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *ServerConfig) Reset() {
	*x = ServerConfig{}
	mi := &file_proxy_shadowsocks_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ServerConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServerConfig) ProtoMessage() {}

func (x *ServerConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_shadowsocks_config_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ServerConfig.ProtoReflect.Descriptor instead.
func (*ServerConfig) Descriptor() ([]byte, []int) {
	return file_proxy_shadowsocks_config_proto_rawDescGZIP(), []int{1}
}

// Deprecated: Marked as deprecated in proxy/shadowsocks/config.proto.
func (x *ServerConfig) GetUdpEnabled() bool {
	if x != nil {
		return x.UdpEnabled
	}
	return false
}

func (x *ServerConfig) GetUser() *protocol.User {
	if x != nil {
		return x.User
	}
	return nil
}

func (x *ServerConfig) GetNetwork() []net.Network {
	if x != nil {
		return x.Network
	}
	return nil
}

func (x *ServerConfig) GetPacketEncoding() packetaddr.PacketAddrType {
	if x != nil {
		return x.PacketEncoding
	}
	return packetaddr.PacketAddrType(0)
}

func (x *ServerConfig) GetPlugin() string {
	if x != nil {
		return x.Plugin
	}
	return ""
}

func (x *ServerConfig) GetPluginOpts() string {
	if x != nil {
		return x.PluginOpts
	}
	return ""
}

func (x *ServerConfig) GetPluginArgs() []string {
	if x != nil {
		return x.PluginArgs
	}
	return nil
}

func (x *ServerConfig) GetPluginWorkingDir() string {
	if x != nil {
		return x.PluginWorkingDir
	}
	return ""
}

type ClientConfig struct {
	state            protoimpl.MessageState     `protogen:"open.v1"`
	Server           []*protocol.ServerEndpoint `protobuf:"bytes,1,rep,name=server,proto3" json:"server,omitempty"`
	Plugin           string                     `protobuf:"bytes,2,opt,name=plugin,proto3" json:"plugin,omitempty"`
	PluginOpts       string                     `protobuf:"bytes,3,opt,name=plugin_opts,json=pluginOpts,proto3" json:"plugin_opts,omitempty"`
	PluginArgs       []string                   `protobuf:"bytes,4,rep,name=plugin_args,json=pluginArgs,proto3" json:"plugin_args,omitempty"`
	PluginWorkingDir string                     `protobuf:"bytes,5,opt,name=plugin_working_dir,json=pluginWorkingDir,proto3" json:"plugin_working_dir,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *ClientConfig) Reset() {
	*x = ClientConfig{}
	mi := &file_proxy_shadowsocks_config_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ClientConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClientConfig) ProtoMessage() {}

func (x *ClientConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_shadowsocks_config_proto_msgTypes[2]
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
	return file_proxy_shadowsocks_config_proto_rawDescGZIP(), []int{2}
}

func (x *ClientConfig) GetServer() []*protocol.ServerEndpoint {
	if x != nil {
		return x.Server
	}
	return nil
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

var File_proxy_shadowsocks_config_proto protoreflect.FileDescriptor

const file_proxy_shadowsocks_config_proto_rawDesc = "" +
	"\n" +
	"\x1eproxy/shadowsocks/config.proto\x12\x1cv2ray.core.proxy.shadowsocks\x1a\x18common/net/network.proto\x1a\x1acommon/protocol/user.proto\x1a!common/protocol/server_spec.proto\x1a\"common/net/packetaddr/config.proto\"\xd9\x01\n" +
	"\aAccount\x12\x1a\n" +
	"\bpassword\x18\x01 \x01(\tR\bpassword\x12I\n" +
	"\vcipher_type\x18\x02 \x01(\x0e2(.v2ray.core.proxy.shadowsocks.CipherTypeR\n" +
	"cipherType\x12\x19\n" +
	"\biv_check\x18\x03 \x01(\bR\aivCheck\x12L\n" +
	"\"experiment_reduced_iv_head_entropy\x18\x91\xbf\x05 \x01(\bR\x1eexperimentReducedIvHeadEntropy\"\xff\x02\n" +
	"\fServerConfig\x12#\n" +
	"\vudp_enabled\x18\x01 \x01(\bB\x02\x18\x01R\n" +
	"udpEnabled\x124\n" +
	"\x04user\x18\x02 \x01(\v2 .v2ray.core.common.protocol.UserR\x04user\x128\n" +
	"\anetwork\x18\x03 \x03(\x0e2\x1e.v2ray.core.common.net.NetworkR\anetwork\x12R\n" +
	"\x0fpacket_encoding\x18\x04 \x01(\x0e2).v2ray.core.net.packetaddr.PacketAddrTypeR\x0epacketEncoding\x12\x16\n" +
	"\x06plugin\x18\x05 \x01(\tR\x06plugin\x12\x1f\n" +
	"\vplugin_opts\x18\x06 \x01(\tR\n" +
	"pluginOpts\x12\x1f\n" +
	"\vplugin_args\x18\a \x03(\tR\n" +
	"pluginArgs\x12,\n" +
	"\x12plugin_working_dir\x18\b \x01(\tR\x10pluginWorkingDir\"\xda\x01\n" +
	"\fClientConfig\x12B\n" +
	"\x06server\x18\x01 \x03(\v2*.v2ray.core.common.protocol.ServerEndpointR\x06server\x12\x16\n" +
	"\x06plugin\x18\x02 \x01(\tR\x06plugin\x12\x1f\n" +
	"\vplugin_opts\x18\x03 \x01(\tR\n" +
	"pluginOpts\x12\x1f\n" +
	"\vplugin_args\x18\x04 \x03(\tR\n" +
	"pluginArgs\x12,\n" +
	"\x12plugin_working_dir\x18\x05 \x01(\tR\x10pluginWorkingDir*\x8b\x05\n" +
	"\n" +
	"CipherType\x12\v\n" +
	"\aUNKNOWN\x10\x00\x12\x0f\n" +
	"\vAES_128_GCM\x10\x01\x12\x0f\n" +
	"\vAES_256_GCM\x10\x02\x12\x15\n" +
	"\x11CHACHA20_POLY1305\x10\x03\x12\b\n" +
	"\x04NONE\x10\x04\x12\x16\n" +
	"\x12XCHACHA20_POLY1305\x10\x05\x12\x0f\n" +
	"\vAES_192_GCM\x10\x06\x12\x0f\n" +
	"\vAES_128_CTR\x10\a\x12\x0f\n" +
	"\vAES_192_CTR\x10\b\x12\x0f\n" +
	"\vAES_256_CTR\x10\t\x12\x0f\n" +
	"\vAES_128_CFB\x10\n" +
	"\x12\x0f\n" +
	"\vAES_192_CFB\x10\v\x12\x0f\n" +
	"\vAES_256_CFB\x10\f\x12\x10\n" +
	"\fAES_128_CFB8\x10\r\x12\x10\n" +
	"\fAES_192_CFB8\x10\x0e\x12\x10\n" +
	"\fAES_256_CFB8\x10\x0f\x12\x0f\n" +
	"\vAES_128_OFB\x10\x10\x12\x0f\n" +
	"\vAES_192_OFB\x10\x11\x12\x0f\n" +
	"\vAES_256_OFB\x10\x12\x12\a\n" +
	"\x03RC4\x10\x13\x12\v\n" +
	"\aRC4_MD5\x10\x14\x12\r\n" +
	"\tRC4_MD5_6\x10\x15\x12\n" +
	"\n" +
	"\x06BF_CFB\x10\x16\x12\r\n" +
	"\tCAST5_CFB\x10\x17\x12\v\n" +
	"\aDES_CFB\x10\x18\x12\v\n" +
	"\aRC2_CFB\x10\x19\x12\f\n" +
	"\bSEED_CFB\x10\x1a\x12\x14\n" +
	"\x10CAMELLIA_128_CFB\x10\x1b\x12\x14\n" +
	"\x10CAMELLIA_192_CFB\x10\x1c\x12\x14\n" +
	"\x10CAMELLIA_256_CFB\x10\x1d\x12\x15\n" +
	"\x11CAMELLIA_128_CFB8\x10\x1e\x12\x15\n" +
	"\x11CAMELLIA_192_CFB8\x10\x1f\x12\x15\n" +
	"\x11CAMELLIA_256_CFB8\x10 \x12\v\n" +
	"\aSALSA20\x10!\x12\f\n" +
	"\bCHACHA20\x10\"\x12\x11\n" +
	"\rCHACHA20_IETF\x10#\x12\r\n" +
	"\tXCHACHA20\x10$\x12\t\n" +
	"\x05TABLE\x10%Bu\n" +
	" com.v2ray.core.proxy.shadowsocksP\x01Z0github.com/v2fly/v2ray-core/v5/proxy/shadowsocks\xaa\x02\x1cV2Ray.Core.Proxy.Shadowsocksb\x06proto3"

var (
	file_proxy_shadowsocks_config_proto_rawDescOnce sync.Once
	file_proxy_shadowsocks_config_proto_rawDescData []byte
)

func file_proxy_shadowsocks_config_proto_rawDescGZIP() []byte {
	file_proxy_shadowsocks_config_proto_rawDescOnce.Do(func() {
		file_proxy_shadowsocks_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proxy_shadowsocks_config_proto_rawDesc), len(file_proxy_shadowsocks_config_proto_rawDesc)))
	})
	return file_proxy_shadowsocks_config_proto_rawDescData
}

var file_proxy_shadowsocks_config_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_proxy_shadowsocks_config_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_proxy_shadowsocks_config_proto_goTypes = []any{
	(CipherType)(0),                 // 0: v2ray.core.proxy.shadowsocks.CipherType
	(*Account)(nil),                 // 1: v2ray.core.proxy.shadowsocks.Account
	(*ServerConfig)(nil),            // 2: v2ray.core.proxy.shadowsocks.ServerConfig
	(*ClientConfig)(nil),            // 3: v2ray.core.proxy.shadowsocks.ClientConfig
	(*protocol.User)(nil),           // 4: v2ray.core.common.protocol.User
	(net.Network)(0),                // 5: v2ray.core.common.net.Network
	(packetaddr.PacketAddrType)(0),  // 6: v2ray.core.net.packetaddr.PacketAddrType
	(*protocol.ServerEndpoint)(nil), // 7: v2ray.core.common.protocol.ServerEndpoint
}
var file_proxy_shadowsocks_config_proto_depIdxs = []int32{
	0, // 0: v2ray.core.proxy.shadowsocks.Account.cipher_type:type_name -> v2ray.core.proxy.shadowsocks.CipherType
	4, // 1: v2ray.core.proxy.shadowsocks.ServerConfig.user:type_name -> v2ray.core.common.protocol.User
	5, // 2: v2ray.core.proxy.shadowsocks.ServerConfig.network:type_name -> v2ray.core.common.net.Network
	6, // 3: v2ray.core.proxy.shadowsocks.ServerConfig.packet_encoding:type_name -> v2ray.core.net.packetaddr.PacketAddrType
	7, // 4: v2ray.core.proxy.shadowsocks.ClientConfig.server:type_name -> v2ray.core.common.protocol.ServerEndpoint
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_proxy_shadowsocks_config_proto_init() }
func file_proxy_shadowsocks_config_proto_init() {
	if File_proxy_shadowsocks_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proxy_shadowsocks_config_proto_rawDesc), len(file_proxy_shadowsocks_config_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proxy_shadowsocks_config_proto_goTypes,
		DependencyIndexes: file_proxy_shadowsocks_config_proto_depIdxs,
		EnumInfos:         file_proxy_shadowsocks_config_proto_enumTypes,
		MessageInfos:      file_proxy_shadowsocks_config_proto_msgTypes,
	}.Build()
	File_proxy_shadowsocks_config_proto = out.File
	file_proxy_shadowsocks_config_proto_goTypes = nil
	file_proxy_shadowsocks_config_proto_depIdxs = nil
}
