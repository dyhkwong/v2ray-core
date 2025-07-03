package dns

import (
	fakedns "github.com/v2fly/v2ray-core/v4/app/dns/fakedns"
	router "github.com/v2fly/v2ray-core/v4/app/router"
	net "github.com/v2fly/v2ray-core/v4/common/net"
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

type DomainMatchingType int32

const (
	DomainMatchingType_Full      DomainMatchingType = 0
	DomainMatchingType_Subdomain DomainMatchingType = 1
	DomainMatchingType_Keyword   DomainMatchingType = 2
	DomainMatchingType_Regex     DomainMatchingType = 3
)

// Enum value maps for DomainMatchingType.
var (
	DomainMatchingType_name = map[int32]string{
		0: "Full",
		1: "Subdomain",
		2: "Keyword",
		3: "Regex",
	}
	DomainMatchingType_value = map[string]int32{
		"Full":      0,
		"Subdomain": 1,
		"Keyword":   2,
		"Regex":     3,
	}
)

func (x DomainMatchingType) Enum() *DomainMatchingType {
	p := new(DomainMatchingType)
	*p = x
	return p
}

func (x DomainMatchingType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (DomainMatchingType) Descriptor() protoreflect.EnumDescriptor {
	return file_app_dns_config_proto_enumTypes[0].Descriptor()
}

func (DomainMatchingType) Type() protoreflect.EnumType {
	return &file_app_dns_config_proto_enumTypes[0]
}

func (x DomainMatchingType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use DomainMatchingType.Descriptor instead.
func (DomainMatchingType) EnumDescriptor() ([]byte, []int) {
	return file_app_dns_config_proto_rawDescGZIP(), []int{0}
}

type QueryStrategy int32

const (
	QueryStrategy_USE_IP  QueryStrategy = 0
	QueryStrategy_USE_IP4 QueryStrategy = 1
	QueryStrategy_USE_IP6 QueryStrategy = 2
)

// Enum value maps for QueryStrategy.
var (
	QueryStrategy_name = map[int32]string{
		0: "USE_IP",
		1: "USE_IP4",
		2: "USE_IP6",
	}
	QueryStrategy_value = map[string]int32{
		"USE_IP":  0,
		"USE_IP4": 1,
		"USE_IP6": 2,
	}
)

func (x QueryStrategy) Enum() *QueryStrategy {
	p := new(QueryStrategy)
	*p = x
	return p
}

func (x QueryStrategy) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (QueryStrategy) Descriptor() protoreflect.EnumDescriptor {
	return file_app_dns_config_proto_enumTypes[1].Descriptor()
}

func (QueryStrategy) Type() protoreflect.EnumType {
	return &file_app_dns_config_proto_enumTypes[1]
}

func (x QueryStrategy) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use QueryStrategy.Descriptor instead.
func (QueryStrategy) EnumDescriptor() ([]byte, []int) {
	return file_app_dns_config_proto_rawDescGZIP(), []int{1}
}

type CacheStrategy int32

const (
	CacheStrategy_CacheEnabled  CacheStrategy = 0
	CacheStrategy_CacheDisabled CacheStrategy = 1
)

// Enum value maps for CacheStrategy.
var (
	CacheStrategy_name = map[int32]string{
		0: "CacheEnabled",
		1: "CacheDisabled",
	}
	CacheStrategy_value = map[string]int32{
		"CacheEnabled":  0,
		"CacheDisabled": 1,
	}
)

func (x CacheStrategy) Enum() *CacheStrategy {
	p := new(CacheStrategy)
	*p = x
	return p
}

func (x CacheStrategy) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (CacheStrategy) Descriptor() protoreflect.EnumDescriptor {
	return file_app_dns_config_proto_enumTypes[2].Descriptor()
}

func (CacheStrategy) Type() protoreflect.EnumType {
	return &file_app_dns_config_proto_enumTypes[2]
}

func (x CacheStrategy) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use CacheStrategy.Descriptor instead.
func (CacheStrategy) EnumDescriptor() ([]byte, []int) {
	return file_app_dns_config_proto_rawDescGZIP(), []int{2}
}

type FallbackStrategy int32

const (
	FallbackStrategy_Enabled            FallbackStrategy = 0
	FallbackStrategy_Disabled           FallbackStrategy = 1
	FallbackStrategy_DisabledIfAnyMatch FallbackStrategy = 2
)

// Enum value maps for FallbackStrategy.
var (
	FallbackStrategy_name = map[int32]string{
		0: "Enabled",
		1: "Disabled",
		2: "DisabledIfAnyMatch",
	}
	FallbackStrategy_value = map[string]int32{
		"Enabled":            0,
		"Disabled":           1,
		"DisabledIfAnyMatch": 2,
	}
)

func (x FallbackStrategy) Enum() *FallbackStrategy {
	p := new(FallbackStrategy)
	*p = x
	return p
}

func (x FallbackStrategy) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (FallbackStrategy) Descriptor() protoreflect.EnumDescriptor {
	return file_app_dns_config_proto_enumTypes[3].Descriptor()
}

func (FallbackStrategy) Type() protoreflect.EnumType {
	return &file_app_dns_config_proto_enumTypes[3]
}

func (x FallbackStrategy) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use FallbackStrategy.Descriptor instead.
func (FallbackStrategy) EnumDescriptor() ([]byte, []int) {
	return file_app_dns_config_proto_rawDescGZIP(), []int{3}
}

type NameServer struct {
	state             protoimpl.MessageState       `protogen:"open.v1"`
	Address           *net.Endpoint                `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	ClientIp          []byte                       `protobuf:"bytes,5,opt,name=client_ip,json=clientIp,proto3" json:"client_ip,omitempty"`
	Tag               string                       `protobuf:"bytes,7,opt,name=tag,proto3" json:"tag,omitempty"`
	PrioritizedDomain []*NameServer_PriorityDomain `protobuf:"bytes,2,rep,name=prioritized_domain,json=prioritizedDomain,proto3" json:"prioritized_domain,omitempty"`
	Geoip             []*router.GeoIP              `protobuf:"bytes,3,rep,name=geoip,proto3" json:"geoip,omitempty"`
	OriginalRules     []*NameServer_OriginalRule   `protobuf:"bytes,4,rep,name=original_rules,json=originalRules,proto3" json:"original_rules,omitempty"`
	FakeDns           *fakedns.FakeDnsPoolMulti    `protobuf:"bytes,11,opt,name=fake_dns,json=fakeDns,proto3" json:"fake_dns,omitempty"`
	// Deprecated. Use fallback_strategy.
	//
	// Deprecated: Marked as deprecated in app/dns/config.proto.
	SkipFallback     bool              `protobuf:"varint,6,opt,name=skipFallback,proto3" json:"skipFallback,omitempty"`
	QueryStrategy    *QueryStrategy    `protobuf:"varint,8,opt,name=query_strategy,json=queryStrategy,proto3,enum=v2ray.core.app.dns.QueryStrategy,oneof" json:"query_strategy,omitempty"`
	CacheStrategy    *CacheStrategy    `protobuf:"varint,9,opt,name=cache_strategy,json=cacheStrategy,proto3,enum=v2ray.core.app.dns.CacheStrategy,oneof" json:"cache_strategy,omitempty"`
	FallbackStrategy *FallbackStrategy `protobuf:"varint,10,opt,name=fallback_strategy,json=fallbackStrategy,proto3,enum=v2ray.core.app.dns.FallbackStrategy,oneof" json:"fallback_strategy,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *NameServer) Reset() {
	*x = NameServer{}
	mi := &file_app_dns_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NameServer) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NameServer) ProtoMessage() {}

func (x *NameServer) ProtoReflect() protoreflect.Message {
	mi := &file_app_dns_config_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NameServer.ProtoReflect.Descriptor instead.
func (*NameServer) Descriptor() ([]byte, []int) {
	return file_app_dns_config_proto_rawDescGZIP(), []int{0}
}

func (x *NameServer) GetAddress() *net.Endpoint {
	if x != nil {
		return x.Address
	}
	return nil
}

func (x *NameServer) GetClientIp() []byte {
	if x != nil {
		return x.ClientIp
	}
	return nil
}

func (x *NameServer) GetTag() string {
	if x != nil {
		return x.Tag
	}
	return ""
}

func (x *NameServer) GetPrioritizedDomain() []*NameServer_PriorityDomain {
	if x != nil {
		return x.PrioritizedDomain
	}
	return nil
}

func (x *NameServer) GetGeoip() []*router.GeoIP {
	if x != nil {
		return x.Geoip
	}
	return nil
}

func (x *NameServer) GetOriginalRules() []*NameServer_OriginalRule {
	if x != nil {
		return x.OriginalRules
	}
	return nil
}

func (x *NameServer) GetFakeDns() *fakedns.FakeDnsPoolMulti {
	if x != nil {
		return x.FakeDns
	}
	return nil
}

// Deprecated: Marked as deprecated in app/dns/config.proto.
func (x *NameServer) GetSkipFallback() bool {
	if x != nil {
		return x.SkipFallback
	}
	return false
}

func (x *NameServer) GetQueryStrategy() QueryStrategy {
	if x != nil && x.QueryStrategy != nil {
		return *x.QueryStrategy
	}
	return QueryStrategy_USE_IP
}

func (x *NameServer) GetCacheStrategy() CacheStrategy {
	if x != nil && x.CacheStrategy != nil {
		return *x.CacheStrategy
	}
	return CacheStrategy_CacheEnabled
}

func (x *NameServer) GetFallbackStrategy() FallbackStrategy {
	if x != nil && x.FallbackStrategy != nil {
		return *x.FallbackStrategy
	}
	return FallbackStrategy_Enabled
}

type Config struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Nameservers used by this DNS. Only traditional UDP servers are support at
	// the moment. A special value 'localhost' as a domain address can be set to
	// use DNS on local system.
	//
	// Deprecated: Marked as deprecated in app/dns/config.proto.
	NameServers []*net.Endpoint `protobuf:"bytes,1,rep,name=NameServers,proto3" json:"NameServers,omitempty"`
	// NameServer list used by this DNS client.
	NameServer []*NameServer `protobuf:"bytes,5,rep,name=name_server,json=nameServer,proto3" json:"name_server,omitempty"`
	// Static hosts. Domain to IP.
	// Deprecated. Use static_hosts.
	//
	// Deprecated: Marked as deprecated in app/dns/config.proto.
	Hosts map[string]*net.IPOrDomain `protobuf:"bytes,2,rep,name=Hosts,proto3" json:"Hosts,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// Client IP for EDNS client subnet. Must be 4 bytes (IPv4) or 16 bytes
	// (IPv6).
	ClientIp []byte `protobuf:"bytes,3,opt,name=client_ip,json=clientIp,proto3" json:"client_ip,omitempty"`
	// Static domain-ip mapping in DNS server.
	StaticHosts []*Config_HostMapping `protobuf:"bytes,4,rep,name=static_hosts,json=staticHosts,proto3" json:"static_hosts,omitempty"`
	// Global fakedns object.
	FakeDns *fakedns.FakeDnsPoolMulti `protobuf:"bytes,16,opt,name=fake_dns,json=fakeDns,proto3" json:"fake_dns,omitempty"`
	// Tag is the inbound tag of DNS client.
	Tag string `protobuf:"bytes,6,opt,name=tag,proto3" json:"tag,omitempty"`
	// Domain matcher to use
	DomainMatcher string `protobuf:"bytes,15,opt,name=domain_matcher,json=domainMatcher,proto3" json:"domain_matcher,omitempty"`
	// DisableCache disables DNS cache
	// Deprecated. Use cache_strategy.
	//
	// Deprecated: Marked as deprecated in app/dns/config.proto.
	DisableCache bool `protobuf:"varint,8,opt,name=disableCache,proto3" json:"disableCache,omitempty"`
	// Deprecated. Use fallback_strategy.
	//
	// Deprecated: Marked as deprecated in app/dns/config.proto.
	DisableFallback bool `protobuf:"varint,10,opt,name=disableFallback,proto3" json:"disableFallback,omitempty"`
	// Deprecated. Use fallback_strategy.
	//
	// Deprecated: Marked as deprecated in app/dns/config.proto.
	DisableFallbackIfMatch bool `protobuf:"varint,11,opt,name=disableFallbackIfMatch,proto3" json:"disableFallbackIfMatch,omitempty"`
	// Default query strategy (IPv4, IPv6, or both) for each name server.
	QueryStrategy QueryStrategy `protobuf:"varint,9,opt,name=query_strategy,json=queryStrategy,proto3,enum=v2ray.core.app.dns.QueryStrategy" json:"query_strategy,omitempty"`
	// Default cache strategy for each name server.
	CacheStrategy CacheStrategy `protobuf:"varint,12,opt,name=cache_strategy,json=cacheStrategy,proto3,enum=v2ray.core.app.dns.CacheStrategy" json:"cache_strategy,omitempty"`
	// Default fallback strategy for each name server.
	FallbackStrategy FallbackStrategy `protobuf:"varint,13,opt,name=fallback_strategy,json=fallbackStrategy,proto3,enum=v2ray.core.app.dns.FallbackStrategy" json:"fallback_strategy,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *Config) Reset() {
	*x = Config{}
	mi := &file_app_dns_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Config) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config) ProtoMessage() {}

func (x *Config) ProtoReflect() protoreflect.Message {
	mi := &file_app_dns_config_proto_msgTypes[1]
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
	return file_app_dns_config_proto_rawDescGZIP(), []int{1}
}

// Deprecated: Marked as deprecated in app/dns/config.proto.
func (x *Config) GetNameServers() []*net.Endpoint {
	if x != nil {
		return x.NameServers
	}
	return nil
}

func (x *Config) GetNameServer() []*NameServer {
	if x != nil {
		return x.NameServer
	}
	return nil
}

// Deprecated: Marked as deprecated in app/dns/config.proto.
func (x *Config) GetHosts() map[string]*net.IPOrDomain {
	if x != nil {
		return x.Hosts
	}
	return nil
}

func (x *Config) GetClientIp() []byte {
	if x != nil {
		return x.ClientIp
	}
	return nil
}

func (x *Config) GetStaticHosts() []*Config_HostMapping {
	if x != nil {
		return x.StaticHosts
	}
	return nil
}

func (x *Config) GetFakeDns() *fakedns.FakeDnsPoolMulti {
	if x != nil {
		return x.FakeDns
	}
	return nil
}

func (x *Config) GetTag() string {
	if x != nil {
		return x.Tag
	}
	return ""
}

func (x *Config) GetDomainMatcher() string {
	if x != nil {
		return x.DomainMatcher
	}
	return ""
}

// Deprecated: Marked as deprecated in app/dns/config.proto.
func (x *Config) GetDisableCache() bool {
	if x != nil {
		return x.DisableCache
	}
	return false
}

// Deprecated: Marked as deprecated in app/dns/config.proto.
func (x *Config) GetDisableFallback() bool {
	if x != nil {
		return x.DisableFallback
	}
	return false
}

// Deprecated: Marked as deprecated in app/dns/config.proto.
func (x *Config) GetDisableFallbackIfMatch() bool {
	if x != nil {
		return x.DisableFallbackIfMatch
	}
	return false
}

func (x *Config) GetQueryStrategy() QueryStrategy {
	if x != nil {
		return x.QueryStrategy
	}
	return QueryStrategy_USE_IP
}

func (x *Config) GetCacheStrategy() CacheStrategy {
	if x != nil {
		return x.CacheStrategy
	}
	return CacheStrategy_CacheEnabled
}

func (x *Config) GetFallbackStrategy() FallbackStrategy {
	if x != nil {
		return x.FallbackStrategy
	}
	return FallbackStrategy_Enabled
}

type NameServer_PriorityDomain struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Type          DomainMatchingType     `protobuf:"varint,1,opt,name=type,proto3,enum=v2ray.core.app.dns.DomainMatchingType" json:"type,omitempty"`
	Domain        string                 `protobuf:"bytes,2,opt,name=domain,proto3" json:"domain,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *NameServer_PriorityDomain) Reset() {
	*x = NameServer_PriorityDomain{}
	mi := &file_app_dns_config_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NameServer_PriorityDomain) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NameServer_PriorityDomain) ProtoMessage() {}

func (x *NameServer_PriorityDomain) ProtoReflect() protoreflect.Message {
	mi := &file_app_dns_config_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NameServer_PriorityDomain.ProtoReflect.Descriptor instead.
func (*NameServer_PriorityDomain) Descriptor() ([]byte, []int) {
	return file_app_dns_config_proto_rawDescGZIP(), []int{0, 0}
}

func (x *NameServer_PriorityDomain) GetType() DomainMatchingType {
	if x != nil {
		return x.Type
	}
	return DomainMatchingType_Full
}

func (x *NameServer_PriorityDomain) GetDomain() string {
	if x != nil {
		return x.Domain
	}
	return ""
}

type NameServer_OriginalRule struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Rule          string                 `protobuf:"bytes,1,opt,name=rule,proto3" json:"rule,omitempty"`
	Size          uint32                 `protobuf:"varint,2,opt,name=size,proto3" json:"size,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *NameServer_OriginalRule) Reset() {
	*x = NameServer_OriginalRule{}
	mi := &file_app_dns_config_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NameServer_OriginalRule) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NameServer_OriginalRule) ProtoMessage() {}

func (x *NameServer_OriginalRule) ProtoReflect() protoreflect.Message {
	mi := &file_app_dns_config_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NameServer_OriginalRule.ProtoReflect.Descriptor instead.
func (*NameServer_OriginalRule) Descriptor() ([]byte, []int) {
	return file_app_dns_config_proto_rawDescGZIP(), []int{0, 1}
}

func (x *NameServer_OriginalRule) GetRule() string {
	if x != nil {
		return x.Rule
	}
	return ""
}

func (x *NameServer_OriginalRule) GetSize() uint32 {
	if x != nil {
		return x.Size
	}
	return 0
}

type Config_HostMapping struct {
	state  protoimpl.MessageState `protogen:"open.v1"`
	Type   DomainMatchingType     `protobuf:"varint,1,opt,name=type,proto3,enum=v2ray.core.app.dns.DomainMatchingType" json:"type,omitempty"`
	Domain string                 `protobuf:"bytes,2,opt,name=domain,proto3" json:"domain,omitempty"`
	Ip     [][]byte               `protobuf:"bytes,3,rep,name=ip,proto3" json:"ip,omitempty"`
	// ProxiedDomain indicates the mapped domain has the same IP address on this
	// domain. V2Ray will use this domain for IP queries.
	ProxiedDomain string `protobuf:"bytes,4,opt,name=proxied_domain,json=proxiedDomain,proto3" json:"proxied_domain,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Config_HostMapping) Reset() {
	*x = Config_HostMapping{}
	mi := &file_app_dns_config_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Config_HostMapping) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config_HostMapping) ProtoMessage() {}

func (x *Config_HostMapping) ProtoReflect() protoreflect.Message {
	mi := &file_app_dns_config_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Config_HostMapping.ProtoReflect.Descriptor instead.
func (*Config_HostMapping) Descriptor() ([]byte, []int) {
	return file_app_dns_config_proto_rawDescGZIP(), []int{1, 1}
}

func (x *Config_HostMapping) GetType() DomainMatchingType {
	if x != nil {
		return x.Type
	}
	return DomainMatchingType_Full
}

func (x *Config_HostMapping) GetDomain() string {
	if x != nil {
		return x.Domain
	}
	return ""
}

func (x *Config_HostMapping) GetIp() [][]byte {
	if x != nil {
		return x.Ip
	}
	return nil
}

func (x *Config_HostMapping) GetProxiedDomain() string {
	if x != nil {
		return x.ProxiedDomain
	}
	return ""
}

var File_app_dns_config_proto protoreflect.FileDescriptor

const file_app_dns_config_proto_rawDesc = "" +
	"\n" +
	"\x14app/dns/config.proto\x12\x12v2ray.core.app.dns\x1a\x18common/net/address.proto\x1a\x1ccommon/net/destination.proto\x1a\x17app/router/config.proto\x1a\x1dapp/dns/fakedns/fakedns.proto\"\x9d\a\n" +
	"\n" +
	"NameServer\x129\n" +
	"\aaddress\x18\x01 \x01(\v2\x1f.v2ray.core.common.net.EndpointR\aaddress\x12\x1b\n" +
	"\tclient_ip\x18\x05 \x01(\fR\bclientIp\x12\x10\n" +
	"\x03tag\x18\a \x01(\tR\x03tag\x12\\\n" +
	"\x12prioritized_domain\x18\x02 \x03(\v2-.v2ray.core.app.dns.NameServer.PriorityDomainR\x11prioritizedDomain\x122\n" +
	"\x05geoip\x18\x03 \x03(\v2\x1c.v2ray.core.app.router.GeoIPR\x05geoip\x12R\n" +
	"\x0eoriginal_rules\x18\x04 \x03(\v2+.v2ray.core.app.dns.NameServer.OriginalRuleR\roriginalRules\x12G\n" +
	"\bfake_dns\x18\v \x01(\v2,.v2ray.core.app.dns.fakedns.FakeDnsPoolMultiR\afakeDns\x12&\n" +
	"\fskipFallback\x18\x06 \x01(\bB\x02\x18\x01R\fskipFallback\x12M\n" +
	"\x0equery_strategy\x18\b \x01(\x0e2!.v2ray.core.app.dns.QueryStrategyH\x00R\rqueryStrategy\x88\x01\x01\x12M\n" +
	"\x0ecache_strategy\x18\t \x01(\x0e2!.v2ray.core.app.dns.CacheStrategyH\x01R\rcacheStrategy\x88\x01\x01\x12V\n" +
	"\x11fallback_strategy\x18\n" +
	" \x01(\x0e2$.v2ray.core.app.dns.FallbackStrategyH\x02R\x10fallbackStrategy\x88\x01\x01\x1ad\n" +
	"\x0ePriorityDomain\x12:\n" +
	"\x04type\x18\x01 \x01(\x0e2&.v2ray.core.app.dns.DomainMatchingTypeR\x04type\x12\x16\n" +
	"\x06domain\x18\x02 \x01(\tR\x06domain\x1a6\n" +
	"\fOriginalRule\x12\x12\n" +
	"\x04rule\x18\x01 \x01(\tR\x04rule\x12\x12\n" +
	"\x04size\x18\x02 \x01(\rR\x04sizeB\x11\n" +
	"\x0f_query_strategyB\x11\n" +
	"\x0f_cache_strategyB\x14\n" +
	"\x12_fallback_strategy\"\xb2\b\n" +
	"\x06Config\x12E\n" +
	"\vNameServers\x18\x01 \x03(\v2\x1f.v2ray.core.common.net.EndpointB\x02\x18\x01R\vNameServers\x12?\n" +
	"\vname_server\x18\x05 \x03(\v2\x1e.v2ray.core.app.dns.NameServerR\n" +
	"nameServer\x12?\n" +
	"\x05Hosts\x18\x02 \x03(\v2%.v2ray.core.app.dns.Config.HostsEntryB\x02\x18\x01R\x05Hosts\x12\x1b\n" +
	"\tclient_ip\x18\x03 \x01(\fR\bclientIp\x12I\n" +
	"\fstatic_hosts\x18\x04 \x03(\v2&.v2ray.core.app.dns.Config.HostMappingR\vstaticHosts\x12G\n" +
	"\bfake_dns\x18\x10 \x01(\v2,.v2ray.core.app.dns.fakedns.FakeDnsPoolMultiR\afakeDns\x12\x10\n" +
	"\x03tag\x18\x06 \x01(\tR\x03tag\x12%\n" +
	"\x0edomain_matcher\x18\x0f \x01(\tR\rdomainMatcher\x12&\n" +
	"\fdisableCache\x18\b \x01(\bB\x02\x18\x01R\fdisableCache\x12,\n" +
	"\x0fdisableFallback\x18\n" +
	" \x01(\bB\x02\x18\x01R\x0fdisableFallback\x12:\n" +
	"\x16disableFallbackIfMatch\x18\v \x01(\bB\x02\x18\x01R\x16disableFallbackIfMatch\x12H\n" +
	"\x0equery_strategy\x18\t \x01(\x0e2!.v2ray.core.app.dns.QueryStrategyR\rqueryStrategy\x12H\n" +
	"\x0ecache_strategy\x18\f \x01(\x0e2!.v2ray.core.app.dns.CacheStrategyR\rcacheStrategy\x12Q\n" +
	"\x11fallback_strategy\x18\r \x01(\x0e2$.v2ray.core.app.dns.FallbackStrategyR\x10fallbackStrategy\x1a[\n" +
	"\n" +
	"HostsEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x127\n" +
	"\x05value\x18\x02 \x01(\v2!.v2ray.core.common.net.IPOrDomainR\x05value:\x028\x01\x1a\x98\x01\n" +
	"\vHostMapping\x12:\n" +
	"\x04type\x18\x01 \x01(\x0e2&.v2ray.core.app.dns.DomainMatchingTypeR\x04type\x12\x16\n" +
	"\x06domain\x18\x02 \x01(\tR\x06domain\x12\x0e\n" +
	"\x02ip\x18\x03 \x03(\fR\x02ip\x12%\n" +
	"\x0eproxied_domain\x18\x04 \x01(\tR\rproxiedDomainJ\x04\b\a\x10\b*E\n" +
	"\x12DomainMatchingType\x12\b\n" +
	"\x04Full\x10\x00\x12\r\n" +
	"\tSubdomain\x10\x01\x12\v\n" +
	"\aKeyword\x10\x02\x12\t\n" +
	"\x05Regex\x10\x03*5\n" +
	"\rQueryStrategy\x12\n" +
	"\n" +
	"\x06USE_IP\x10\x00\x12\v\n" +
	"\aUSE_IP4\x10\x01\x12\v\n" +
	"\aUSE_IP6\x10\x02*4\n" +
	"\rCacheStrategy\x12\x10\n" +
	"\fCacheEnabled\x10\x00\x12\x11\n" +
	"\rCacheDisabled\x10\x01*E\n" +
	"\x10FallbackStrategy\x12\v\n" +
	"\aEnabled\x10\x00\x12\f\n" +
	"\bDisabled\x10\x01\x12\x16\n" +
	"\x12DisabledIfAnyMatch\x10\x02BW\n" +
	"\x16com.v2ray.core.app.dnsP\x01Z&github.com/v2fly/v2ray-core/v4/app/dns\xaa\x02\x12V2Ray.Core.App.Dnsb\x06proto3"

var (
	file_app_dns_config_proto_rawDescOnce sync.Once
	file_app_dns_config_proto_rawDescData []byte
)

func file_app_dns_config_proto_rawDescGZIP() []byte {
	file_app_dns_config_proto_rawDescOnce.Do(func() {
		file_app_dns_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_app_dns_config_proto_rawDesc), len(file_app_dns_config_proto_rawDesc)))
	})
	return file_app_dns_config_proto_rawDescData
}

var file_app_dns_config_proto_enumTypes = make([]protoimpl.EnumInfo, 4)
var file_app_dns_config_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_app_dns_config_proto_goTypes = []any{
	(DomainMatchingType)(0),           // 0: v2ray.core.app.dns.DomainMatchingType
	(QueryStrategy)(0),                // 1: v2ray.core.app.dns.QueryStrategy
	(CacheStrategy)(0),                // 2: v2ray.core.app.dns.CacheStrategy
	(FallbackStrategy)(0),             // 3: v2ray.core.app.dns.FallbackStrategy
	(*NameServer)(nil),                // 4: v2ray.core.app.dns.NameServer
	(*Config)(nil),                    // 5: v2ray.core.app.dns.Config
	(*NameServer_PriorityDomain)(nil), // 6: v2ray.core.app.dns.NameServer.PriorityDomain
	(*NameServer_OriginalRule)(nil),   // 7: v2ray.core.app.dns.NameServer.OriginalRule
	nil,                               // 8: v2ray.core.app.dns.Config.HostsEntry
	(*Config_HostMapping)(nil),        // 9: v2ray.core.app.dns.Config.HostMapping
	(*net.Endpoint)(nil),              // 10: v2ray.core.common.net.Endpoint
	(*router.GeoIP)(nil),              // 11: v2ray.core.app.router.GeoIP
	(*fakedns.FakeDnsPoolMulti)(nil),  // 12: v2ray.core.app.dns.fakedns.FakeDnsPoolMulti
	(*net.IPOrDomain)(nil),            // 13: v2ray.core.common.net.IPOrDomain
}
var file_app_dns_config_proto_depIdxs = []int32{
	10, // 0: v2ray.core.app.dns.NameServer.address:type_name -> v2ray.core.common.net.Endpoint
	6,  // 1: v2ray.core.app.dns.NameServer.prioritized_domain:type_name -> v2ray.core.app.dns.NameServer.PriorityDomain
	11, // 2: v2ray.core.app.dns.NameServer.geoip:type_name -> v2ray.core.app.router.GeoIP
	7,  // 3: v2ray.core.app.dns.NameServer.original_rules:type_name -> v2ray.core.app.dns.NameServer.OriginalRule
	12, // 4: v2ray.core.app.dns.NameServer.fake_dns:type_name -> v2ray.core.app.dns.fakedns.FakeDnsPoolMulti
	1,  // 5: v2ray.core.app.dns.NameServer.query_strategy:type_name -> v2ray.core.app.dns.QueryStrategy
	2,  // 6: v2ray.core.app.dns.NameServer.cache_strategy:type_name -> v2ray.core.app.dns.CacheStrategy
	3,  // 7: v2ray.core.app.dns.NameServer.fallback_strategy:type_name -> v2ray.core.app.dns.FallbackStrategy
	10, // 8: v2ray.core.app.dns.Config.NameServers:type_name -> v2ray.core.common.net.Endpoint
	4,  // 9: v2ray.core.app.dns.Config.name_server:type_name -> v2ray.core.app.dns.NameServer
	8,  // 10: v2ray.core.app.dns.Config.Hosts:type_name -> v2ray.core.app.dns.Config.HostsEntry
	9,  // 11: v2ray.core.app.dns.Config.static_hosts:type_name -> v2ray.core.app.dns.Config.HostMapping
	12, // 12: v2ray.core.app.dns.Config.fake_dns:type_name -> v2ray.core.app.dns.fakedns.FakeDnsPoolMulti
	1,  // 13: v2ray.core.app.dns.Config.query_strategy:type_name -> v2ray.core.app.dns.QueryStrategy
	2,  // 14: v2ray.core.app.dns.Config.cache_strategy:type_name -> v2ray.core.app.dns.CacheStrategy
	3,  // 15: v2ray.core.app.dns.Config.fallback_strategy:type_name -> v2ray.core.app.dns.FallbackStrategy
	0,  // 16: v2ray.core.app.dns.NameServer.PriorityDomain.type:type_name -> v2ray.core.app.dns.DomainMatchingType
	13, // 17: v2ray.core.app.dns.Config.HostsEntry.value:type_name -> v2ray.core.common.net.IPOrDomain
	0,  // 18: v2ray.core.app.dns.Config.HostMapping.type:type_name -> v2ray.core.app.dns.DomainMatchingType
	19, // [19:19] is the sub-list for method output_type
	19, // [19:19] is the sub-list for method input_type
	19, // [19:19] is the sub-list for extension type_name
	19, // [19:19] is the sub-list for extension extendee
	0,  // [0:19] is the sub-list for field type_name
}

func init() { file_app_dns_config_proto_init() }
func file_app_dns_config_proto_init() {
	if File_app_dns_config_proto != nil {
		return
	}
	file_app_dns_config_proto_msgTypes[0].OneofWrappers = []any{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_app_dns_config_proto_rawDesc), len(file_app_dns_config_proto_rawDesc)),
			NumEnums:      4,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_app_dns_config_proto_goTypes,
		DependencyIndexes: file_app_dns_config_proto_depIdxs,
		EnumInfos:         file_app_dns_config_proto_enumTypes,
		MessageInfos:      file_app_dns_config_proto_msgTypes,
	}.Build()
	File_app_dns_config_proto = out.File
	file_app_dns_config_proto_goTypes = nil
	file_app_dns_config_proto_depIdxs = nil
}
