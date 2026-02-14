package tls

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

type Certificate_Usage int32

const (
	Certificate_ENCIPHERMENT            Certificate_Usage = 0
	Certificate_AUTHORITY_VERIFY        Certificate_Usage = 1
	Certificate_AUTHORITY_ISSUE         Certificate_Usage = 2
	Certificate_AUTHORITY_VERIFY_CLIENT Certificate_Usage = 3
)

// Enum value maps for Certificate_Usage.
var (
	Certificate_Usage_name = map[int32]string{
		0: "ENCIPHERMENT",
		1: "AUTHORITY_VERIFY",
		2: "AUTHORITY_ISSUE",
		3: "AUTHORITY_VERIFY_CLIENT",
	}
	Certificate_Usage_value = map[string]int32{
		"ENCIPHERMENT":            0,
		"AUTHORITY_VERIFY":        1,
		"AUTHORITY_ISSUE":         2,
		"AUTHORITY_VERIFY_CLIENT": 3,
	}
)

func (x Certificate_Usage) Enum() *Certificate_Usage {
	p := new(Certificate_Usage)
	*p = x
	return p
}

func (x Certificate_Usage) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Certificate_Usage) Descriptor() protoreflect.EnumDescriptor {
	return file_transport_internet_tls_config_proto_enumTypes[0].Descriptor()
}

func (Certificate_Usage) Type() protoreflect.EnumType {
	return &file_transport_internet_tls_config_proto_enumTypes[0]
}

func (x Certificate_Usage) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Certificate_Usage.Descriptor instead.
func (Certificate_Usage) EnumDescriptor() ([]byte, []int) {
	return file_transport_internet_tls_config_proto_rawDescGZIP(), []int{0, 0}
}

type Config_TLSVersion int32

const (
	Config_Default Config_TLSVersion = 0
	Config_TLS1_0  Config_TLSVersion = 1
	Config_TLS1_1  Config_TLSVersion = 2
	Config_TLS1_2  Config_TLSVersion = 3
	Config_TLS1_3  Config_TLSVersion = 4
)

// Enum value maps for Config_TLSVersion.
var (
	Config_TLSVersion_name = map[int32]string{
		0: "Default",
		1: "TLS1_0",
		2: "TLS1_1",
		3: "TLS1_2",
		4: "TLS1_3",
	}
	Config_TLSVersion_value = map[string]int32{
		"Default": 0,
		"TLS1_0":  1,
		"TLS1_1":  2,
		"TLS1_2":  3,
		"TLS1_3":  4,
	}
)

func (x Config_TLSVersion) Enum() *Config_TLSVersion {
	p := new(Config_TLSVersion)
	*p = x
	return p
}

func (x Config_TLSVersion) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Config_TLSVersion) Descriptor() protoreflect.EnumDescriptor {
	return file_transport_internet_tls_config_proto_enumTypes[1].Descriptor()
}

func (Config_TLSVersion) Type() protoreflect.EnumType {
	return &file_transport_internet_tls_config_proto_enumTypes[1]
}

func (x Config_TLSVersion) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Config_TLSVersion.Descriptor instead.
func (Config_TLSVersion) EnumDescriptor() ([]byte, []int) {
	return file_transport_internet_tls_config_proto_rawDescGZIP(), []int{1, 0}
}

type Certificate struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// TLS certificate in x509 format.
	Certificate []byte `protobuf:"bytes,1,opt,name=Certificate,proto3" json:"Certificate,omitempty"`
	// TLS key in x509 format.
	Key             []byte            `protobuf:"bytes,2,opt,name=Key,proto3" json:"Key,omitempty"`
	Usage           Certificate_Usage `protobuf:"varint,3,opt,name=usage,proto3,enum=v2ray.core.transport.internet.tls.Certificate_Usage" json:"usage,omitempty"`
	CertificateFile string            `protobuf:"bytes,96001,opt,name=certificate_file,json=certificateFile,proto3" json:"certificate_file,omitempty"`
	KeyFile         string            `protobuf:"bytes,96002,opt,name=key_file,json=keyFile,proto3" json:"key_file,omitempty"`
	unknownFields   protoimpl.UnknownFields
	sizeCache       protoimpl.SizeCache
}

func (x *Certificate) Reset() {
	*x = Certificate{}
	mi := &file_transport_internet_tls_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Certificate) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Certificate) ProtoMessage() {}

func (x *Certificate) ProtoReflect() protoreflect.Message {
	mi := &file_transport_internet_tls_config_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Certificate.ProtoReflect.Descriptor instead.
func (*Certificate) Descriptor() ([]byte, []int) {
	return file_transport_internet_tls_config_proto_rawDescGZIP(), []int{0}
}

func (x *Certificate) GetCertificate() []byte {
	if x != nil {
		return x.Certificate
	}
	return nil
}

func (x *Certificate) GetKey() []byte {
	if x != nil {
		return x.Key
	}
	return nil
}

func (x *Certificate) GetUsage() Certificate_Usage {
	if x != nil {
		return x.Usage
	}
	return Certificate_ENCIPHERMENT
}

func (x *Certificate) GetCertificateFile() string {
	if x != nil {
		return x.CertificateFile
	}
	return ""
}

func (x *Certificate) GetKeyFile() string {
	if x != nil {
		return x.KeyFile
	}
	return ""
}

type Config struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Whether or not to allow self-signed certificates.
	AllowInsecure bool `protobuf:"varint,1,opt,name=allow_insecure,json=allowInsecure,proto3" json:"allow_insecure,omitempty"`
	// List of certificates to be served on server.
	Certificate []*Certificate `protobuf:"bytes,2,rep,name=certificate,proto3" json:"certificate,omitempty"`
	// Override server name.
	ServerName string `protobuf:"bytes,3,opt,name=server_name,json=serverName,proto3" json:"server_name,omitempty"`
	// Lists of string as ALPN values.
	NextProtocol []string `protobuf:"bytes,4,rep,name=next_protocol,json=nextProtocol,proto3" json:"next_protocol,omitempty"`
	// Whether or not to enable session (ticket) resumption.
	EnableSessionResumption bool `protobuf:"varint,5,opt,name=enable_session_resumption,json=enableSessionResumption,proto3" json:"enable_session_resumption,omitempty"`
	// If true, root certificates on the system will not be loaded for
	// verification.
	DisableSystemRoot bool `protobuf:"varint,6,opt,name=disable_system_root,json=disableSystemRoot,proto3" json:"disable_system_root,omitempty"`
	// @Document A pinned certificate chain sha256 hash.
	// @Document If the server's hash does not match this value, the connection will be aborted.
	// @Document This value replace allow_insecure.
	// @Critical
	PinnedPeerCertificateChainSha256 [][]byte `protobuf:"bytes,7,rep,name=pinned_peer_certificate_chain_sha256,json=pinnedPeerCertificateChainSha256,proto3" json:"pinned_peer_certificate_chain_sha256,omitempty"`
	// If true, the client is required to present a certificate.
	VerifyClientCertificate bool `protobuf:"varint,8,opt,name=verify_client_certificate,json=verifyClientCertificate,proto3" json:"verify_client_certificate,omitempty"`
	// Minimum TLS version to support.
	MinVersion Config_TLSVersion `protobuf:"varint,9,opt,name=min_version,json=minVersion,proto3,enum=v2ray.core.transport.internet.tls.Config_TLSVersion" json:"min_version,omitempty"`
	// Maximum TLS version to support.
	MaxVersion Config_TLSVersion `protobuf:"varint,10,opt,name=max_version,json=maxVersion,proto3,enum=v2ray.core.transport.internet.tls.Config_TLSVersion" json:"max_version,omitempty"`
	// Whether or not to allow self-signed certificates when pinned_peer_certificate_chain_sha256 is present.
	AllowInsecureIfPinnedPeerCertificate bool `protobuf:"varint,11,opt,name=allow_insecure_if_pinned_peer_certificate,json=allowInsecureIfPinnedPeerCertificate,proto3" json:"allow_insecure_if_pinned_peer_certificate,omitempty"`
	// cipher suites to to be offered or accepted.
	// This is an developer option.
	Ciphersuites                         []uint32    `protobuf:"varint,19,rep,packed,name=ciphersuites,proto3" json:"ciphersuites,omitempty"`
	PinnedPeerCertificatePublicKeySha256 [][]byte    `protobuf:"bytes,900,rep,name=pinned_peer_certificate_public_key_sha256,json=pinnedPeerCertificatePublicKeySha256,proto3" json:"pinned_peer_certificate_public_key_sha256,omitempty"`
	PinnedPeerCertificateSha256          []string    `protobuf:"bytes,901,rep,name=pinned_peer_certificate_sha256,json=pinnedPeerCertificateSha256,proto3" json:"pinned_peer_certificate_sha256,omitempty"`
	Ech                                  *Config_ECH `protobuf:"bytes,902,opt,name=ech,proto3" json:"ech,omitempty"`
	unknownFields                        protoimpl.UnknownFields
	sizeCache                            protoimpl.SizeCache
}

func (x *Config) Reset() {
	*x = Config{}
	mi := &file_transport_internet_tls_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Config) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config) ProtoMessage() {}

func (x *Config) ProtoReflect() protoreflect.Message {
	mi := &file_transport_internet_tls_config_proto_msgTypes[1]
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
	return file_transport_internet_tls_config_proto_rawDescGZIP(), []int{1}
}

func (x *Config) GetAllowInsecure() bool {
	if x != nil {
		return x.AllowInsecure
	}
	return false
}

func (x *Config) GetCertificate() []*Certificate {
	if x != nil {
		return x.Certificate
	}
	return nil
}

func (x *Config) GetServerName() string {
	if x != nil {
		return x.ServerName
	}
	return ""
}

func (x *Config) GetNextProtocol() []string {
	if x != nil {
		return x.NextProtocol
	}
	return nil
}

func (x *Config) GetEnableSessionResumption() bool {
	if x != nil {
		return x.EnableSessionResumption
	}
	return false
}

func (x *Config) GetDisableSystemRoot() bool {
	if x != nil {
		return x.DisableSystemRoot
	}
	return false
}

func (x *Config) GetPinnedPeerCertificateChainSha256() [][]byte {
	if x != nil {
		return x.PinnedPeerCertificateChainSha256
	}
	return nil
}

func (x *Config) GetVerifyClientCertificate() bool {
	if x != nil {
		return x.VerifyClientCertificate
	}
	return false
}

func (x *Config) GetMinVersion() Config_TLSVersion {
	if x != nil {
		return x.MinVersion
	}
	return Config_Default
}

func (x *Config) GetMaxVersion() Config_TLSVersion {
	if x != nil {
		return x.MaxVersion
	}
	return Config_Default
}

func (x *Config) GetAllowInsecureIfPinnedPeerCertificate() bool {
	if x != nil {
		return x.AllowInsecureIfPinnedPeerCertificate
	}
	return false
}

func (x *Config) GetCiphersuites() []uint32 {
	if x != nil {
		return x.Ciphersuites
	}
	return nil
}

func (x *Config) GetPinnedPeerCertificatePublicKeySha256() [][]byte {
	if x != nil {
		return x.PinnedPeerCertificatePublicKeySha256
	}
	return nil
}

func (x *Config) GetPinnedPeerCertificateSha256() []string {
	if x != nil {
		return x.PinnedPeerCertificateSha256
	}
	return nil
}

func (x *Config) GetEch() *Config_ECH {
	if x != nil {
		return x.Ech
	}
	return nil
}

type Config_ECH struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Enabled       bool                   `protobuf:"varint,1,opt,name=enabled,proto3" json:"enabled,omitempty"`
	Config        []byte                 `protobuf:"bytes,2,opt,name=config,proto3" json:"config,omitempty"`
	QueryDomain   string                 `protobuf:"bytes,3,opt,name=query_domain,json=queryDomain,proto3" json:"query_domain,omitempty"`
	Key           []byte                 `protobuf:"bytes,4,opt,name=key,proto3" json:"key,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Config_ECH) Reset() {
	*x = Config_ECH{}
	mi := &file_transport_internet_tls_config_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Config_ECH) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config_ECH) ProtoMessage() {}

func (x *Config_ECH) ProtoReflect() protoreflect.Message {
	mi := &file_transport_internet_tls_config_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Config_ECH.ProtoReflect.Descriptor instead.
func (*Config_ECH) Descriptor() ([]byte, []int) {
	return file_transport_internet_tls_config_proto_rawDescGZIP(), []int{1, 0}
}

func (x *Config_ECH) GetEnabled() bool {
	if x != nil {
		return x.Enabled
	}
	return false
}

func (x *Config_ECH) GetConfig() []byte {
	if x != nil {
		return x.Config
	}
	return nil
}

func (x *Config_ECH) GetQueryDomain() string {
	if x != nil {
		return x.QueryDomain
	}
	return ""
}

func (x *Config_ECH) GetKey() []byte {
	if x != nil {
		return x.Key
	}
	return nil
}

var File_transport_internet_tls_config_proto protoreflect.FileDescriptor

const file_transport_internet_tls_config_proto_rawDesc = "" +
	"\n" +
	"#transport/internet/tls/config.proto\x12!v2ray.core.transport.internet.tls\x1a common/protoext/extensions.proto\"\xd8\x02\n" +
	"\vCertificate\x12 \n" +
	"\vCertificate\x18\x01 \x01(\fR\vCertificate\x12\x10\n" +
	"\x03Key\x18\x02 \x01(\fR\x03Key\x12J\n" +
	"\x05usage\x18\x03 \x01(\x0e24.v2ray.core.transport.internet.tls.Certificate.UsageR\x05usage\x12>\n" +
	"\x10certificate_file\x18\x81\xee\x05 \x01(\tB\x11\x82\xb5\x18\r\"\vCertificateR\x0fcertificateFile\x12&\n" +
	"\bkey_file\x18\x82\xee\x05 \x01(\tB\t\x82\xb5\x18\x05\"\x03KeyR\akeyFile\"a\n" +
	"\x05Usage\x12\x10\n" +
	"\fENCIPHERMENT\x10\x00\x12\x14\n" +
	"\x10AUTHORITY_VERIFY\x10\x01\x12\x13\n" +
	"\x0fAUTHORITY_ISSUE\x10\x02\x12\x1b\n" +
	"\x17AUTHORITY_VERIFY_CLIENT\x10\x03\"\xb4\t\n" +
	"\x06Config\x12-\n" +
	"\x0eallow_insecure\x18\x01 \x01(\bB\x06\x82\xb5\x18\x02(\x01R\rallowInsecure\x12P\n" +
	"\vcertificate\x18\x02 \x03(\v2..v2ray.core.transport.internet.tls.CertificateR\vcertificate\x12\x1f\n" +
	"\vserver_name\x18\x03 \x01(\tR\n" +
	"serverName\x12#\n" +
	"\rnext_protocol\x18\x04 \x03(\tR\fnextProtocol\x12:\n" +
	"\x19enable_session_resumption\x18\x05 \x01(\bR\x17enableSessionResumption\x12.\n" +
	"\x13disable_system_root\x18\x06 \x01(\bR\x11disableSystemRoot\x12N\n" +
	"$pinned_peer_certificate_chain_sha256\x18\a \x03(\fR pinnedPeerCertificateChainSha256\x12:\n" +
	"\x19verify_client_certificate\x18\b \x01(\bR\x17verifyClientCertificate\x12U\n" +
	"\vmin_version\x18\t \x01(\x0e24.v2ray.core.transport.internet.tls.Config.TLSVersionR\n" +
	"minVersion\x12U\n" +
	"\vmax_version\x18\n" +
	" \x01(\x0e24.v2ray.core.transport.internet.tls.Config.TLSVersionR\n" +
	"maxVersion\x12W\n" +
	")allow_insecure_if_pinned_peer_certificate\x18\v \x01(\bR$allowInsecureIfPinnedPeerCertificate\x12\"\n" +
	"\fciphersuites\x18\x13 \x03(\rR\fciphersuites\x12X\n" +
	")pinned_peer_certificate_public_key_sha256\x18\x84\a \x03(\fR$pinnedPeerCertificatePublicKeySha256\x12D\n" +
	"\x1epinned_peer_certificate_sha256\x18\x85\a \x03(\tR\x1bpinnedPeerCertificateSha256\x12@\n" +
	"\x03ech\x18\x86\a \x01(\v2-.v2ray.core.transport.internet.tls.Config.ECHR\x03ech\x1al\n" +
	"\x03ECH\x12\x18\n" +
	"\aenabled\x18\x01 \x01(\bR\aenabled\x12\x16\n" +
	"\x06config\x18\x02 \x01(\fR\x06config\x12!\n" +
	"\fquery_domain\x18\x03 \x01(\tR\vqueryDomain\x12\x10\n" +
	"\x03key\x18\x04 \x01(\fR\x03key\"I\n" +
	"\n" +
	"TLSVersion\x12\v\n" +
	"\aDefault\x10\x00\x12\n" +
	"\n" +
	"\x06TLS1_0\x10\x01\x12\n" +
	"\n" +
	"\x06TLS1_1\x10\x02\x12\n" +
	"\n" +
	"\x06TLS1_2\x10\x03\x12\n" +
	"\n" +
	"\x06TLS1_3\x10\x04:\x13\x82\xb5\x18\x0f\n" +
	"\bsecurity\x12\x03tlsJ\x04\b\x10\x10\x11J\x04\b\x11\x10\x12J\x04\b\x12\x10\x13B\x84\x01\n" +
	"%com.v2ray.core.transport.internet.tlsP\x01Z5github.com/v2fly/v2ray-core/v5/transport/internet/tls\xaa\x02!V2Ray.Core.Transport.Internet.Tlsb\x06proto3"

var (
	file_transport_internet_tls_config_proto_rawDescOnce sync.Once
	file_transport_internet_tls_config_proto_rawDescData []byte
)

func file_transport_internet_tls_config_proto_rawDescGZIP() []byte {
	file_transport_internet_tls_config_proto_rawDescOnce.Do(func() {
		file_transport_internet_tls_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_transport_internet_tls_config_proto_rawDesc), len(file_transport_internet_tls_config_proto_rawDesc)))
	})
	return file_transport_internet_tls_config_proto_rawDescData
}

var file_transport_internet_tls_config_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_transport_internet_tls_config_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_transport_internet_tls_config_proto_goTypes = []any{
	(Certificate_Usage)(0), // 0: v2ray.core.transport.internet.tls.Certificate.Usage
	(Config_TLSVersion)(0), // 1: v2ray.core.transport.internet.tls.Config.TLSVersion
	(*Certificate)(nil),    // 2: v2ray.core.transport.internet.tls.Certificate
	(*Config)(nil),         // 3: v2ray.core.transport.internet.tls.Config
	(*Config_ECH)(nil),     // 4: v2ray.core.transport.internet.tls.Config.ECH
}
var file_transport_internet_tls_config_proto_depIdxs = []int32{
	0, // 0: v2ray.core.transport.internet.tls.Certificate.usage:type_name -> v2ray.core.transport.internet.tls.Certificate.Usage
	2, // 1: v2ray.core.transport.internet.tls.Config.certificate:type_name -> v2ray.core.transport.internet.tls.Certificate
	1, // 2: v2ray.core.transport.internet.tls.Config.min_version:type_name -> v2ray.core.transport.internet.tls.Config.TLSVersion
	1, // 3: v2ray.core.transport.internet.tls.Config.max_version:type_name -> v2ray.core.transport.internet.tls.Config.TLSVersion
	4, // 4: v2ray.core.transport.internet.tls.Config.ech:type_name -> v2ray.core.transport.internet.tls.Config.ECH
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_transport_internet_tls_config_proto_init() }
func file_transport_internet_tls_config_proto_init() {
	if File_transport_internet_tls_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_transport_internet_tls_config_proto_rawDesc), len(file_transport_internet_tls_config_proto_rawDesc)),
			NumEnums:      2,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_transport_internet_tls_config_proto_goTypes,
		DependencyIndexes: file_transport_internet_tls_config_proto_depIdxs,
		EnumInfos:         file_transport_internet_tls_config_proto_enumTypes,
		MessageInfos:      file_transport_internet_tls_config_proto_msgTypes,
	}.Build()
	File_transport_internet_tls_config_proto = out.File
	file_transport_internet_tls_config_proto_goTypes = nil
	file_transport_internet_tls_config_proto_depIdxs = nil
}
