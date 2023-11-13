package wireguard

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

type ClientConfig_DomainStrategy int32

const (
	ClientConfig_USE_IP     ClientConfig_DomainStrategy = 0
	ClientConfig_USE_IP4    ClientConfig_DomainStrategy = 1
	ClientConfig_USE_IP6    ClientConfig_DomainStrategy = 2
	ClientConfig_PREFER_IP4 ClientConfig_DomainStrategy = 3
	ClientConfig_PREFER_IP6 ClientConfig_DomainStrategy = 4
)

// Enum value maps for ClientConfig_DomainStrategy.
var (
	ClientConfig_DomainStrategy_name = map[int32]string{
		0: "USE_IP",
		1: "USE_IP4",
		2: "USE_IP6",
		3: "PREFER_IP4",
		4: "PREFER_IP6",
	}
	ClientConfig_DomainStrategy_value = map[string]int32{
		"USE_IP":     0,
		"USE_IP4":    1,
		"USE_IP6":    2,
		"PREFER_IP4": 3,
		"PREFER_IP6": 4,
	}
)

func (x ClientConfig_DomainStrategy) Enum() *ClientConfig_DomainStrategy {
	p := new(ClientConfig_DomainStrategy)
	*p = x
	return p
}

func (x ClientConfig_DomainStrategy) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ClientConfig_DomainStrategy) Descriptor() protoreflect.EnumDescriptor {
	return file_proxy_wireguard_config_proto_enumTypes[0].Descriptor()
}

func (ClientConfig_DomainStrategy) Type() protoreflect.EnumType {
	return &file_proxy_wireguard_config_proto_enumTypes[0]
}

func (x ClientConfig_DomainStrategy) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ClientConfig_DomainStrategy.Descriptor instead.
func (ClientConfig_DomainStrategy) EnumDescriptor() ([]byte, []int) {
	return file_proxy_wireguard_config_proto_rawDescGZIP(), []int{1, 0}
}

type PeerConfig struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	PublicKey     string                 `protobuf:"bytes,1,opt,name=public_key,json=publicKey,proto3" json:"public_key,omitempty"`
	PreSharedKey  string                 `protobuf:"bytes,2,opt,name=pre_shared_key,json=preSharedKey,proto3" json:"pre_shared_key,omitempty"`
	Endpoint      string                 `protobuf:"bytes,3,opt,name=endpoint,proto3" json:"endpoint,omitempty"`
	KeepAlive     uint32                 `protobuf:"varint,4,opt,name=keep_alive,json=keepAlive,proto3" json:"keep_alive,omitempty"`
	AllowedIps    []string               `protobuf:"bytes,5,rep,name=allowed_ips,json=allowedIps,proto3" json:"allowed_ips,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *PeerConfig) Reset() {
	*x = PeerConfig{}
	mi := &file_proxy_wireguard_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PeerConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PeerConfig) ProtoMessage() {}

func (x *PeerConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_wireguard_config_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PeerConfig.ProtoReflect.Descriptor instead.
func (*PeerConfig) Descriptor() ([]byte, []int) {
	return file_proxy_wireguard_config_proto_rawDescGZIP(), []int{0}
}

func (x *PeerConfig) GetPublicKey() string {
	if x != nil {
		return x.PublicKey
	}
	return ""
}

func (x *PeerConfig) GetPreSharedKey() string {
	if x != nil {
		return x.PreSharedKey
	}
	return ""
}

func (x *PeerConfig) GetEndpoint() string {
	if x != nil {
		return x.Endpoint
	}
	return ""
}

func (x *PeerConfig) GetKeepAlive() uint32 {
	if x != nil {
		return x.KeepAlive
	}
	return 0
}

func (x *PeerConfig) GetAllowedIps() []string {
	if x != nil {
		return x.AllowedIps
	}
	return nil
}

type ClientConfig struct {
	state          protoimpl.MessageState      `protogen:"open.v1"`
	SecretKey      string                      `protobuf:"bytes,1,opt,name=secret_key,json=secretKey,proto3" json:"secret_key,omitempty"`
	Address        []string                    `protobuf:"bytes,2,rep,name=address,proto3" json:"address,omitempty"`
	Peers          []*PeerConfig               `protobuf:"bytes,3,rep,name=peers,proto3" json:"peers,omitempty"`
	Mtu            int32                       `protobuf:"varint,4,opt,name=mtu,proto3" json:"mtu,omitempty"`
	NumWorkers     int32                       `protobuf:"varint,5,opt,name=num_workers,json=numWorkers,proto3" json:"num_workers,omitempty"`
	Reserved       []byte                      `protobuf:"bytes,6,opt,name=reserved,proto3" json:"reserved,omitempty"`
	DomainStrategy ClientConfig_DomainStrategy `protobuf:"varint,7,opt,name=domain_strategy,json=domainStrategy,proto3,enum=v2ray.core.proxy.wireguard.ClientConfig_DomainStrategy" json:"domain_strategy,omitempty"`
	unknownFields  protoimpl.UnknownFields
	sizeCache      protoimpl.SizeCache
}

func (x *ClientConfig) Reset() {
	*x = ClientConfig{}
	mi := &file_proxy_wireguard_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ClientConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClientConfig) ProtoMessage() {}

func (x *ClientConfig) ProtoReflect() protoreflect.Message {
	mi := &file_proxy_wireguard_config_proto_msgTypes[1]
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
	return file_proxy_wireguard_config_proto_rawDescGZIP(), []int{1}
}

func (x *ClientConfig) GetSecretKey() string {
	if x != nil {
		return x.SecretKey
	}
	return ""
}

func (x *ClientConfig) GetAddress() []string {
	if x != nil {
		return x.Address
	}
	return nil
}

func (x *ClientConfig) GetPeers() []*PeerConfig {
	if x != nil {
		return x.Peers
	}
	return nil
}

func (x *ClientConfig) GetMtu() int32 {
	if x != nil {
		return x.Mtu
	}
	return 0
}

func (x *ClientConfig) GetNumWorkers() int32 {
	if x != nil {
		return x.NumWorkers
	}
	return 0
}

func (x *ClientConfig) GetReserved() []byte {
	if x != nil {
		return x.Reserved
	}
	return nil
}

func (x *ClientConfig) GetDomainStrategy() ClientConfig_DomainStrategy {
	if x != nil {
		return x.DomainStrategy
	}
	return ClientConfig_USE_IP
}

var File_proxy_wireguard_config_proto protoreflect.FileDescriptor

const file_proxy_wireguard_config_proto_rawDesc = "" +
	"\n" +
	"\x1cproxy/wireguard/config.proto\x12\x1av2ray.core.proxy.wireguard\x1a common/protoext/extensions.proto\"\xad\x01\n" +
	"\n" +
	"PeerConfig\x12\x1d\n" +
	"\n" +
	"public_key\x18\x01 \x01(\tR\tpublicKey\x12$\n" +
	"\x0epre_shared_key\x18\x02 \x01(\tR\fpreSharedKey\x12\x1a\n" +
	"\bendpoint\x18\x03 \x01(\tR\bendpoint\x12\x1d\n" +
	"\n" +
	"keep_alive\x18\x04 \x01(\rR\tkeepAlive\x12\x1f\n" +
	"\vallowed_ips\x18\x05 \x03(\tR\n" +
	"allowedIps\"\xa9\x03\n" +
	"\fClientConfig\x12\x1d\n" +
	"\n" +
	"secret_key\x18\x01 \x01(\tR\tsecretKey\x12\x18\n" +
	"\aaddress\x18\x02 \x03(\tR\aaddress\x12<\n" +
	"\x05peers\x18\x03 \x03(\v2&.v2ray.core.proxy.wireguard.PeerConfigR\x05peers\x12\x10\n" +
	"\x03mtu\x18\x04 \x01(\x05R\x03mtu\x12\x1f\n" +
	"\vnum_workers\x18\x05 \x01(\x05R\n" +
	"numWorkers\x12\x1a\n" +
	"\breserved\x18\x06 \x01(\fR\breserved\x12`\n" +
	"\x0fdomain_strategy\x18\a \x01(\x0e27.v2ray.core.proxy.wireguard.ClientConfig.DomainStrategyR\x0edomainStrategy\"V\n" +
	"\x0eDomainStrategy\x12\n" +
	"\n" +
	"\x06USE_IP\x10\x00\x12\v\n" +
	"\aUSE_IP4\x10\x01\x12\v\n" +
	"\aUSE_IP6\x10\x02\x12\x0e\n" +
	"\n" +
	"PREFER_IP4\x10\x03\x12\x0e\n" +
	"\n" +
	"PREFER_IP6\x10\x04:\x19\x82\xb5\x18\x15\n" +
	"\boutbound\x12\twireguardBo\n" +
	"\x1ecom.v2ray.core.proxy.wireguardP\x01Z.github.com/v2fly/v2ray-core/v5/proxy/wireguard\xaa\x02\x1aV2Ray.Core.Proxy.WireGuardb\x06proto3"

var (
	file_proxy_wireguard_config_proto_rawDescOnce sync.Once
	file_proxy_wireguard_config_proto_rawDescData []byte
)

func file_proxy_wireguard_config_proto_rawDescGZIP() []byte {
	file_proxy_wireguard_config_proto_rawDescOnce.Do(func() {
		file_proxy_wireguard_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proxy_wireguard_config_proto_rawDesc), len(file_proxy_wireguard_config_proto_rawDesc)))
	})
	return file_proxy_wireguard_config_proto_rawDescData
}

var file_proxy_wireguard_config_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_proxy_wireguard_config_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_proxy_wireguard_config_proto_goTypes = []any{
	(ClientConfig_DomainStrategy)(0), // 0: v2ray.core.proxy.wireguard.ClientConfig.DomainStrategy
	(*PeerConfig)(nil),               // 1: v2ray.core.proxy.wireguard.PeerConfig
	(*ClientConfig)(nil),             // 2: v2ray.core.proxy.wireguard.ClientConfig
}
var file_proxy_wireguard_config_proto_depIdxs = []int32{
	1, // 0: v2ray.core.proxy.wireguard.ClientConfig.peers:type_name -> v2ray.core.proxy.wireguard.PeerConfig
	0, // 1: v2ray.core.proxy.wireguard.ClientConfig.domain_strategy:type_name -> v2ray.core.proxy.wireguard.ClientConfig.DomainStrategy
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_proxy_wireguard_config_proto_init() }
func file_proxy_wireguard_config_proto_init() {
	if File_proxy_wireguard_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proxy_wireguard_config_proto_rawDesc), len(file_proxy_wireguard_config_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proxy_wireguard_config_proto_goTypes,
		DependencyIndexes: file_proxy_wireguard_config_proto_depIdxs,
		EnumInfos:         file_proxy_wireguard_config_proto_enumTypes,
		MessageInfos:      file_proxy_wireguard_config_proto_msgTypes,
	}.Build()
	File_proxy_wireguard_config_proto = out.File
	file_proxy_wireguard_config_proto_goTypes = nil
	file_proxy_wireguard_config_proto_depIdxs = nil
}
