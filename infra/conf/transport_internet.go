package conf

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v4/common/platform/filesystem"
	"github.com/v2fly/v2ray-core/v4/common/protocol"
	"github.com/v2fly/v2ray-core/v4/common/serial"
	"github.com/v2fly/v2ray-core/v4/infra/conf/cfgcommon"
	"github.com/v2fly/v2ray-core/v4/transport/internet"
	"github.com/v2fly/v2ray-core/v4/transport/internet/domainsocket"
	httpheader "github.com/v2fly/v2ray-core/v4/transport/internet/headers/http"
	"github.com/v2fly/v2ray-core/v4/transport/internet/http"
	"github.com/v2fly/v2ray-core/v4/transport/internet/httpupgrade"
	"github.com/v2fly/v2ray-core/v4/transport/internet/hysteria2"
	"github.com/v2fly/v2ray-core/v4/transport/internet/kcp"
	"github.com/v2fly/v2ray-core/v4/transport/internet/quic"
	"github.com/v2fly/v2ray-core/v4/transport/internet/request/stereotype/meek"
	"github.com/v2fly/v2ray-core/v4/transport/internet/request/stereotype/mekya"
	"github.com/v2fly/v2ray-core/v4/transport/internet/tcp"
	"github.com/v2fly/v2ray-core/v4/transport/internet/tls"
	"github.com/v2fly/v2ray-core/v4/transport/internet/tls/utls"
	"github.com/v2fly/v2ray-core/v4/transport/internet/websocket"
)

var (
	kcpHeaderLoader = NewJSONConfigLoader(ConfigCreatorCache{
		"none":         func() interface{} { return new(NoOpAuthenticator) },
		"srtp":         func() interface{} { return new(SRTPAuthenticator) },
		"utp":          func() interface{} { return new(UTPAuthenticator) },
		"wechat-video": func() interface{} { return new(WechatVideoAuthenticator) },
		"dtls":         func() interface{} { return new(DTLSAuthenticator) },
		"wireguard":    func() interface{} { return new(WireguardAuthenticator) },
	}, "type", "")

	tcpHeaderLoader = NewJSONConfigLoader(ConfigCreatorCache{
		"none": func() interface{} { return new(NoOpConnectionAuthenticator) },
		"http": func() interface{} { return new(Authenticator) },
	}, "type", "")
)

type KCPConfig struct {
	Mtu             *uint32         `json:"mtu"`
	Tti             *uint32         `json:"tti"`
	UpCap           *uint32         `json:"uplinkCapacity"`
	DownCap         *uint32         `json:"downlinkCapacity"`
	Congestion      *bool           `json:"congestion"`
	ReadBufferSize  *uint32         `json:"readBufferSize"`
	WriteBufferSize *uint32         `json:"writeBufferSize"`
	HeaderConfig    json.RawMessage `json:"header"`
	Seed            *string         `json:"seed"`
}

// Build implements Buildable.
func (c *KCPConfig) Build() (proto.Message, error) {
	config := new(kcp.Config)

	if c.Mtu != nil {
		mtu := *c.Mtu
		if mtu < 576 || mtu > 1460 {
			return nil, newError("invalid mKCP MTU size: ", mtu).AtError()
		}
		config.Mtu = &kcp.MTU{Value: mtu}
	}
	if c.Tti != nil {
		tti := *c.Tti
		if tti < 10 || tti > 100 {
			return nil, newError("invalid mKCP TTI: ", tti).AtError()
		}
		config.Tti = &kcp.TTI{Value: tti}
	}
	if c.UpCap != nil {
		config.UplinkCapacity = &kcp.UplinkCapacity{Value: *c.UpCap}
	}
	if c.DownCap != nil {
		config.DownlinkCapacity = &kcp.DownlinkCapacity{Value: *c.DownCap}
	}
	if c.Congestion != nil {
		config.Congestion = *c.Congestion
	}
	if c.ReadBufferSize != nil {
		size := *c.ReadBufferSize
		if size > 0 {
			config.ReadBuffer = &kcp.ReadBuffer{Size: size * 1024 * 1024}
		} else {
			config.ReadBuffer = &kcp.ReadBuffer{Size: 512 * 1024}
		}
	}
	if c.WriteBufferSize != nil {
		size := *c.WriteBufferSize
		if size > 0 {
			config.WriteBuffer = &kcp.WriteBuffer{Size: size * 1024 * 1024}
		} else {
			config.WriteBuffer = &kcp.WriteBuffer{Size: 512 * 1024}
		}
	}
	if len(c.HeaderConfig) > 0 {
		headerConfig, _, err := kcpHeaderLoader.Load(c.HeaderConfig)
		if err != nil {
			return nil, newError("invalid mKCP header config.").Base(err).AtError()
		}
		ts, err := headerConfig.(Buildable).Build()
		if err != nil {
			return nil, newError("invalid mKCP header config").Base(err).AtError()
		}
		config.HeaderConfig = serial.ToTypedMessage(ts)
	}

	if c.Seed != nil {
		config.Seed = &kcp.EncryptionSeed{Seed: *c.Seed}
	}

	return config, nil
}

type TCPConfig struct {
	HeaderConfig        json.RawMessage `json:"header"`
	AcceptProxyProtocol bool            `json:"acceptProxyProtocol"`
}

// Build implements Buildable.
func (c *TCPConfig) Build() (proto.Message, error) {
	config := new(tcp.Config)
	if len(c.HeaderConfig) > 0 {
		headerConfig, _, err := tcpHeaderLoader.Load(c.HeaderConfig)
		if err != nil {
			return nil, newError("invalid TCP header config").Base(err).AtError()
		}
		ts, err := headerConfig.(Buildable).Build()
		if err != nil {
			return nil, newError("invalid TCP header config").Base(err).AtError()
		}
		config.HeaderSettings = serial.ToTypedMessage(ts)
	}
	if c.AcceptProxyProtocol {
		config.AcceptProxyProtocol = c.AcceptProxyProtocol
	}
	return config, nil
}

type Hy2ConfigCongestion struct {
	Type     string `json:"type"`
	UpMbps   uint64 `json:"up_mbps"`
	DownMbps uint64 `json:"down_mbps"`
}

type Hy2Config struct {
	Password              string              `json:"password"`
	Congestion            Hy2ConfigCongestion `json:"congestion"`
	UseUDPExtension       bool                `json:"use_udp_extension"`
	IgnoreClientBandwidth bool                `json:"ignore_client_bandwidth"`
}

// Build implements Buildable.
func (c *Hy2Config) Build() (proto.Message, error) {
	return &hysteria2.Config{
		Password: c.Password,
		Congestion: &hysteria2.Congestion{
			Type:     c.Congestion.Type,
			DownMbps: c.Congestion.DownMbps,
			UpMbps:   c.Congestion.UpMbps,
		},
		UseUdpExtension:       c.UseUDPExtension,
		IgnoreClientBandwidth: c.IgnoreClientBandwidth,
	}, nil
}

type WebSocketConfig struct {
	Path                 string            `json:"path"`
	Headers              map[string]string `json:"headers"`
	AcceptProxyProtocol  bool              `json:"acceptProxyProtocol"`
	MaxEarlyData         int32             `json:"maxEarlyData"`
	UseBrowserForwarding bool              `json:"useBrowserForwarding"`
	EarlyDataHeaderName  string            `json:"earlyDataHeaderName"`
}

// Build implements Buildable.
func (c *WebSocketConfig) Build() (proto.Message, error) {
	path := c.Path
	header := make([]*websocket.Header, 0, 32)
	for key, value := range c.Headers {
		header = append(header, &websocket.Header{
			Key:   key,
			Value: value,
		})
	}
	config := &websocket.Config{
		Path:                 path,
		Header:               header,
		MaxEarlyData:         c.MaxEarlyData,
		UseBrowserForwarding: c.UseBrowserForwarding,
		EarlyDataHeaderName:  c.EarlyDataHeaderName,
	}
	if c.AcceptProxyProtocol {
		config.AcceptProxyProtocol = c.AcceptProxyProtocol
	}
	return config, nil
}

type HTTPConfig struct {
	Host    *cfgcommon.StringList            `json:"host"`
	Path    string                           `json:"path"`
	Method  string                           `json:"method"`
	Headers map[string]*cfgcommon.StringList `json:"headers"`
}

// Build implements Buildable.
func (c *HTTPConfig) Build() (proto.Message, error) {
	config := &http.Config{
		Path: c.Path,
	}
	if c.Host != nil {
		config.Host = []string(*c.Host)
	}
	if c.Method != "" {
		config.Method = c.Method
	}
	if len(c.Headers) > 0 {
		config.Header = make([]*httpheader.Header, 0, len(c.Headers))
		headerNames := sortMapKeys(c.Headers)
		for _, key := range headerNames {
			value := c.Headers[key]
			if value == nil {
				return nil, newError("empty HTTP header value: " + key).AtError()
			}
			config.Header = append(config.Header, &httpheader.Header{
				Name:  key,
				Value: append([]string(nil), (*value)...),
			})
		}
	}
	return config, nil
}

type HTTPUpgradeHeaderConfig struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type HTTPUpgradeConfig struct {
	Host                string                    `json:"host"`
	Path                string                    `json:"path"`
	MaxEarlyData        int32                     `json:"maxEarlyData"`
	EarlyDataHeaderName string                    `json:"earlyDataHeaderName"`
	Header              []HTTPUpgradeHeaderConfig `json:"header"`
}

// Build implements Buildable.
func (c *HTTPUpgradeConfig) Build() (proto.Message, error) {
	config := &httpupgrade.Config{
		Host:                c.Host,
		Path:                c.Path,
		MaxEarlyData:        c.MaxEarlyData,
		EarlyDataHeaderName: c.EarlyDataHeaderName,
	}
	for _, header := range c.Header {
		config.Header = append(config.Header, &httpupgrade.Header{
			Key:   header.Key,
			Value: header.Value,
		})
	}
	return config, nil
}

type QUICConfig struct {
	Header   json.RawMessage `json:"header"`
	Security string          `json:"security"`
	Key      string          `json:"key"`
}

// Build implements Buildable.
func (c *QUICConfig) Build() (proto.Message, error) {
	config := &quic.Config{
		Key: c.Key,
	}

	if len(c.Header) > 0 {
		headerConfig, _, err := kcpHeaderLoader.Load(c.Header)
		if err != nil {
			return nil, newError("invalid QUIC header config.").Base(err).AtError()
		}
		ts, err := headerConfig.(Buildable).Build()
		if err != nil {
			return nil, newError("invalid QUIC header config").Base(err).AtError()
		}
		config.Header = serial.ToTypedMessage(ts)
	}

	var st protocol.SecurityType
	switch strings.ToLower(c.Security) {
	case "aes-128-gcm":
		st = protocol.SecurityType_AES128_GCM
	case "chacha20-poly1305":
		st = protocol.SecurityType_CHACHA20_POLY1305
	default:
		st = protocol.SecurityType_NONE
	}

	config.Security = &protocol.SecurityConfig{
		Type: st,
	}

	return config, nil
}

type DomainSocketConfig struct {
	Path     string `json:"path"`
	Abstract bool   `json:"abstract"`
	Padding  bool   `json:"padding"`
}

// Build implements Buildable.
func (c *DomainSocketConfig) Build() (proto.Message, error) {
	return &domainsocket.Config{
		Path:     c.Path,
		Abstract: c.Abstract,
		Padding:  c.Padding,
	}, nil
}

type MeekConfig struct {
	URL string `json:"url"`
}

// Build implements Buildable.
func (c *MeekConfig) Build() (proto.Message, error) {
	return &meek.Config{Url: c.URL}, nil
}

type MekyaConfig struct {
	KCP                            *KCPConfig `json:"kcp"`
	MaxWriteDelay                  int32      `json:"maxWriteDelay"`
	MaxRequestSize                 int32      `json:"maxRequestSize"`
	PollingIntervalInitial         int32      `json:"pollingIntervalInitial"`
	MaxWriteSize                   int32      `json:"maxWriteSize"`
	MaxWriteDurationMs             int32      `json:"maxWriteDurationMs"`
	MaxSimultaneousWriteConnection int32      `json:"maxSimultaneousWriteConnection"`
	PacketWritingBuffer            int32      `json:"packetWritingBuffer"`
	URL                            string     `json:"url"`
	H2PoolSize                     int32      `json:"h2PoolSize"`
}

// Build implements Buildable.
func (c *MekyaConfig) Build() (proto.Message, error) {
	config := &mekya.Config{
		MaxWriteDelay:                  c.MaxWriteDelay,
		MaxRequestSize:                 c.MaxRequestSize,
		PollingIntervalInitial:         c.PollingIntervalInitial,
		MaxWriteSize:                   c.MaxWriteSize,
		MaxWriteDurationMs:             c.MaxWriteDurationMs,
		MaxSimultaneousWriteConnection: c.MaxSimultaneousWriteConnection,
		PacketWritingBuffer:            c.PacketWritingBuffer,
		Url:                            c.URL,
		H2PoolSize:                     c.H2PoolSize,
	}
	if c.KCP != nil {
		kcpConfig, err := c.KCP.Build()
		if err != nil {
			return nil, err
		}
		config.Kcp = kcpConfig.(*kcp.Config)
	}
	return config, nil
}

func readFileOrString(f string, s []string) ([]byte, error) {
	if len(f) > 0 {
		return filesystem.ReadFile(f)
	}
	if len(s) > 0 {
		return []byte(strings.Join(s, "\n")), nil
	}
	return nil, newError("both file and bytes are empty.")
}

type TLSCertConfig struct {
	CertFile string   `json:"certificateFile"`
	CertStr  []string `json:"certificate"`
	KeyFile  string   `json:"keyFile"`
	KeyStr   []string `json:"key"`
	Usage    string   `json:"usage"`
}

// Build implements Buildable.
func (c *TLSCertConfig) Build() (*tls.Certificate, error) {
	certificate := new(tls.Certificate)

	cert, err := readFileOrString(c.CertFile, c.CertStr)
	if err != nil {
		return nil, newError("failed to parse certificate").Base(err)
	}
	certificate.Certificate = cert

	if len(c.KeyFile) > 0 || len(c.KeyStr) > 0 {
		key, err := readFileOrString(c.KeyFile, c.KeyStr)
		if err != nil {
			return nil, newError("failed to parse key").Base(err)
		}
		certificate.Key = key
	}

	switch strings.ToLower(c.Usage) {
	case "encipherment":
		certificate.Usage = tls.Certificate_ENCIPHERMENT
	case "verify":
		certificate.Usage = tls.Certificate_AUTHORITY_VERIFY
	case "verifyclient":
		certificate.Usage = tls.Certificate_AUTHORITY_VERIFY_CLIENT
	case "issue":
		certificate.Usage = tls.Certificate_AUTHORITY_ISSUE
	default:
		certificate.Usage = tls.Certificate_ENCIPHERMENT
	}

	return certificate, nil
}

type TLSConfig struct {
	Insecure                             bool                  `json:"allowInsecure"`
	Certs                                []*TLSCertConfig      `json:"certificates"`
	ServerName                           string                `json:"serverName"`
	ALPN                                 *cfgcommon.StringList `json:"alpn"`
	EnableSessionResumption              bool                  `json:"enableSessionResumption"`
	DisableSystemRoot                    bool                  `json:"disableSystemRoot"`
	PinnedPeerCertificateChainSha256     *[]string             `json:"pinnedPeerCertificateChainSha256"`
	VerifyClientCertificate              bool                  `json:"verifyClientCertificate"`
	ECHConfig                            string                `json:"echConfig"`
	ECHDOHServer                         string                `json:"echDohServer"`
	ECHQueryDomain                       string                `json:"echQueryDomain"`
	MinVersion                           string                `json:"minVersion"`
	MaxVersion                           string                `json:"maxVersion"`
	AllowInsecureIfPinnedPeerCertificate bool                  `json:"allowInsecureIfPinnedPeerCertificate"`
	Ciphersuites                         []uint32              `json:"ciphersuites"`
	Fingerprint                          string                `json:"fingerprint"`
}

// Build implements Buildable.
func (c *TLSConfig) Build() (proto.Message, error) {
	config := new(tls.Config)
	config.Certificate = make([]*tls.Certificate, len(c.Certs))
	for idx, certConf := range c.Certs {
		cert, err := certConf.Build()
		if err != nil {
			return nil, err
		}
		config.Certificate[idx] = cert
	}
	serverName := c.ServerName
	config.AllowInsecure = c.Insecure
	config.VerifyClientCertificate = c.VerifyClientCertificate
	if len(c.ServerName) > 0 {
		config.ServerName = serverName
	}
	if c.ALPN != nil && len(*c.ALPN) > 0 {
		config.NextProtocol = []string(*c.ALPN)
	}
	config.EnableSessionResumption = c.EnableSessionResumption
	config.DisableSystemRoot = c.DisableSystemRoot

	if c.PinnedPeerCertificateChainSha256 != nil {
		config.PinnedPeerCertificateChainSha256 = [][]byte{}
		for _, v := range *c.PinnedPeerCertificateChainSha256 {
			hashValue, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return nil, err
			}
			config.PinnedPeerCertificateChainSha256 = append(config.PinnedPeerCertificateChainSha256, hashValue)
		}
	}

	if c.ECHConfig != "" {
		ECHConfig, err := base64.StdEncoding.DecodeString(c.ECHConfig)
		if err != nil {
			return nil, newError("invalid ECH Config", c.ECHConfig)
		}
		config.EchConfig = ECHConfig
	}
	config.Ech_DOHserver = c.ECHDOHServer
	config.EchQueryDomain = c.ECHQueryDomain

	switch strings.ToLower(c.MinVersion) {
	case "tls1_0", "tls1.0":
		config.MinVersion = tls.Config_TLS1_0
	case "tls1_1", "tls1.1":
		config.MinVersion = tls.Config_TLS1_1
	case "tls1_2", "tls1.2":
		config.MinVersion = tls.Config_TLS1_2
	case "tls1_3", "tls1.3":
		config.MinVersion = tls.Config_TLS1_3
	}

	switch strings.ToLower(c.MaxVersion) {
	case "tls1_0", "tls1.0":
		config.MaxVersion = tls.Config_TLS1_0
	case "tls1_1", "tls1.1":
		config.MaxVersion = tls.Config_TLS1_1
	case "tls1_2", "tls1.2":
		config.MaxVersion = tls.Config_TLS1_2
	case "tls1_3", "tls1.3":
		config.MaxVersion = tls.Config_TLS1_3
	}

	config.AllowInsecureIfPinnedPeerCertificate = c.AllowInsecureIfPinnedPeerCertificate

	config.Ciphersuites = c.Ciphersuites

	return config, nil
}

type UTLSConfig struct {
	TLSConfig *TLSConfig `json:"tlsConfig"`
	Imitate   string     `json:"imitate"`
	NoSNI     bool       `json:"noSNI"`
	ForceALPN string     `json:"forceALPN"`
}

// Build implements Buildable.
func (c *UTLSConfig) Build() (proto.Message, error) {
	config := new(utls.Config)
	if c.TLSConfig != nil {
		tlsConfig, err := c.TLSConfig.Build()
		if err != nil {
			return nil, err
		}
		config.TlsConfig = tlsConfig.(*tls.Config)
	}
	imitate := c.Imitate
	if len(c.Imitate) > 0 {
		config.Imitate = imitate
	}
	config.NoSNI = c.NoSNI
	switch strings.ToLower(c.ForceALPN) {
	case "transportpreferencetakepriority", "transport_preference_take_priority":
		config.ForceAlpn = utls.ForcedALPN_TRANSPORT_PREFERENCE_TAKE_PRIORITY
	case "noalpn", "no_alpn":
		config.ForceAlpn = utls.ForcedALPN_NO_ALPN
	case "utlspreset", "utls_preset":
		config.ForceAlpn = utls.ForcedALPN_UTLS_PRESET
	default:
		config.ForceAlpn = utls.ForcedALPN_TRANSPORT_PREFERENCE_TAKE_PRIORITY
	}
	return config, nil
}

type TransportProtocol string

// Build implements Buildable.
func (p TransportProtocol) Build() (string, error) {
	switch strings.ToLower(string(p)) {
	case "tcp":
		return "tcp", nil
	case "kcp", "mkcp":
		return "mkcp", nil
	case "ws", "websocket":
		return "websocket", nil
	case "h2", "http":
		return "http", nil
	case "ds", "domainsocket":
		return "domainsocket", nil
	case "quic":
		return "quic", nil
	case "gun", "grpc":
		return "gun", nil
	case "hy2", "hysteria2":
		return "hysteria2", nil
	case "meek":
		return "meek", nil
	case "httpupgrade":
		return "httpupgrade", nil
	case "mekya":
		return "mekya", nil
	default:
		return "", newError("Config: unknown transport protocol: ", p)
	}
}

type SocketConfig struct {
	Mark                 uint32 `json:"mark"`
	TFO                  *bool  `json:"tcpFastOpen"`
	TFOQueueLength       uint32 `json:"tcpFastOpenQueueLength"`
	TProxy               string `json:"tproxy"`
	AcceptProxyProtocol  bool   `json:"acceptProxyProtocol"`
	TCPKeepAliveInterval int32  `json:"tcpKeepAliveInterval"`
	TCPKeepAliveIdle     int32  `json:"tcpKeepAliveIdle"`
	BindToDevice         string `json:"bindToDevice"`
	RxBufSize            uint64 `json:"rxBufSize"`
	TxBufSize            uint64 `json:"txBufSize"`
	ForceBufSize         bool   `json:"forceBufSize"`
	MPTCP                *bool  `json:"mptcp"`
}

// Build implements Buildable.
func (c *SocketConfig) Build() (*internet.SocketConfig, error) {
	var tfoSettings internet.SocketConfig_TCPFastOpenState
	if c.TFO != nil {
		if *c.TFO {
			tfoSettings = internet.SocketConfig_Enable
		} else {
			tfoSettings = internet.SocketConfig_Disable
		}
	}

	tfoQueueLength := c.TFOQueueLength
	if tfoQueueLength == 0 {
		tfoQueueLength = 4096
	}

	var tproxy internet.SocketConfig_TProxyMode
	switch strings.ToLower(c.TProxy) {
	case "tproxy":
		tproxy = internet.SocketConfig_TProxy
	case "redirect":
		tproxy = internet.SocketConfig_Redirect
	default:
		tproxy = internet.SocketConfig_Off
	}

	var mptcpSettings internet.MPTCPState
	if c.MPTCP != nil {
		if *c.MPTCP {
			mptcpSettings = internet.MPTCPState_Enable
		} else {
			mptcpSettings = internet.MPTCPState_Disable
		}
	}

	return &internet.SocketConfig{
		Mark:                 c.Mark,
		Tfo:                  tfoSettings,
		TfoQueueLength:       tfoQueueLength,
		Tproxy:               tproxy,
		AcceptProxyProtocol:  c.AcceptProxyProtocol,
		TcpKeepAliveInterval: c.TCPKeepAliveInterval,
		TcpKeepAliveIdle:     c.TCPKeepAliveIdle,
		RxBufSize:            int64(c.RxBufSize),
		TxBufSize:            int64(c.TxBufSize),
		ForceBufSize:         c.ForceBufSize,
		BindToDevice:         c.BindToDevice,
		Mptcp:                mptcpSettings,
	}, nil
}

type StreamConfig struct {
	Network             *TransportProtocol  `json:"network"`
	Security            string              `json:"security"`
	TLSSettings         *TLSConfig          `json:"tlsSettings"`
	UTLSSettings        *UTLSConfig         `json:"utlsSettings"`
	TCPSettings         *TCPConfig          `json:"tcpSettings"`
	KCPSettings         *KCPConfig          `json:"kcpSettings"`
	WSSettings          *WebSocketConfig    `json:"wsSettings"`
	HTTPSettings        *HTTPConfig         `json:"httpSettings"`
	DSSettings          *DomainSocketConfig `json:"dsSettings"`
	QUICSettings        *QUICConfig         `json:"quicSettings"`
	GunSettings         *GunConfig          `json:"gunSettings"`
	GRPCSettings        *GunConfig          `json:"grpcSettings"`
	Hy2Settings         *Hy2Config          `json:"hy2Settings"`
	MeekSettings        *MeekConfig         `json:"meekSettings"`
	HTTPUpgradeSettings *HTTPUpgradeConfig  `json:"httpupgradeSettings"`
	MekyaSettings       *MekyaConfig        `json:"mekyaSettings"`
	SocketSettings      *SocketConfig       `json:"sockopt"`
}

// Build implements Buildable.
func (c *StreamConfig) Build() (*internet.StreamConfig, error) {
	config := &internet.StreamConfig{
		ProtocolName: "tcp",
	}
	if c.Network != nil {
		protocol, err := c.Network.Build()
		if err != nil {
			return nil, err
		}
		config.ProtocolName = protocol
	}
	if strings.EqualFold(c.Security, "tls") {
		tlsSettings := c.TLSSettings
		if tlsSettings == nil {
			tlsSettings = &TLSConfig{}
		}
		if tlsSettings.Fingerprint != "" {
			imitate := strings.ToLower(tlsSettings.Fingerprint)
			imitate = strings.TrimPrefix(imitate, "hello")
			switch imitate {
			case "chrome", "firefox", "safari", "ios", "edge", "360", "qq":
				imitate += "_auto"
			}
			utlsSettings := &UTLSConfig{
				TLSConfig: tlsSettings,
				Imitate:   imitate,
			}
			us, err := utlsSettings.Build()
			if err != nil {
				return nil, newError("Failed to build UTLS config.").Base(err)
			}
			tm := serial.ToTypedMessage(us)
			config.SecuritySettings = append(config.SecuritySettings, tm)
			config.SecurityType = tm.Type
		} else {
			ts, err := tlsSettings.Build()
			if err != nil {
				return nil, newError("Failed to build TLS config.").Base(err)
			}
			tm := serial.ToTypedMessage(ts)
			config.SecuritySettings = append(config.SecuritySettings, tm)
			config.SecurityType = tm.Type
		}
	} else if strings.EqualFold(c.Security, "utls") {
		utlsSettings := c.UTLSSettings
		if utlsSettings == nil {
			utlsSettings = &UTLSConfig{}
		}
		us, err := utlsSettings.Build()
		if err != nil {
			return nil, newError("Failed to build UTLS config.").Base(err)
		}
		tm := serial.ToTypedMessage(us)
		config.SecuritySettings = append(config.SecuritySettings, tm)
		config.SecurityType = tm.Type
	}
	if c.TCPSettings != nil {
		ts, err := c.TCPSettings.Build()
		if err != nil {
			return nil, newError("Failed to build TCP config.").Base(err)
		}
		config.TransportSettings = append(config.TransportSettings, &internet.TransportConfig{
			ProtocolName: "tcp",
			Settings:     serial.ToTypedMessage(ts),
		})
	}
	if c.KCPSettings != nil {
		ts, err := c.KCPSettings.Build()
		if err != nil {
			return nil, newError("Failed to build mKCP config.").Base(err)
		}
		config.TransportSettings = append(config.TransportSettings, &internet.TransportConfig{
			ProtocolName: "mkcp",
			Settings:     serial.ToTypedMessage(ts),
		})
	}
	if c.WSSettings != nil {
		ts, err := c.WSSettings.Build()
		if err != nil {
			return nil, newError("Failed to build WebSocket config.").Base(err)
		}
		config.TransportSettings = append(config.TransportSettings, &internet.TransportConfig{
			ProtocolName: "websocket",
			Settings:     serial.ToTypedMessage(ts),
		})
	}
	if c.HTTPSettings != nil {
		ts, err := c.HTTPSettings.Build()
		if err != nil {
			return nil, newError("Failed to build HTTP config.").Base(err)
		}
		config.TransportSettings = append(config.TransportSettings, &internet.TransportConfig{
			ProtocolName: "http",
			Settings:     serial.ToTypedMessage(ts),
		})
	}
	if c.DSSettings != nil {
		ds, err := c.DSSettings.Build()
		if err != nil {
			return nil, newError("Failed to build DomainSocket config.").Base(err)
		}
		config.TransportSettings = append(config.TransportSettings, &internet.TransportConfig{
			ProtocolName: "domainsocket",
			Settings:     serial.ToTypedMessage(ds),
		})
	}
	if c.QUICSettings != nil {
		qs, err := c.QUICSettings.Build()
		if err != nil {
			return nil, newError("Failed to build QUIC config.").Base(err)
		}
		config.TransportSettings = append(config.TransportSettings, &internet.TransportConfig{
			ProtocolName: "quic",
			Settings:     serial.ToTypedMessage(qs),
		})
	}
	if c.GunSettings == nil {
		c.GunSettings = c.GRPCSettings
	}
	if c.GunSettings != nil {
		gs, err := c.GunSettings.Build()
		if err != nil {
			return nil, newError("Failed to build Gun config.").Base(err)
		}
		config.TransportSettings = append(config.TransportSettings, &internet.TransportConfig{
			ProtocolName: "gun",
			Settings:     serial.ToTypedMessage(gs),
		})
	}
	if c.Hy2Settings != nil {
		hy2, err := c.Hy2Settings.Build()
		if err != nil {
			return nil, newError("Failed to build hy2 config.").Base(err)
		}
		config.TransportSettings = append(config.TransportSettings, &internet.TransportConfig{
			ProtocolName: "hysteria2",
			Settings:     serial.ToTypedMessage(hy2),
		})
	}
	if c.MeekSettings != nil {
		ms, err := c.MeekSettings.Build()
		if err != nil {
			return nil, newError("Failed to build Meek config.").Base(err)
		}
		config.TransportSettings = append(config.TransportSettings, &internet.TransportConfig{
			ProtocolName: "meek",
			Settings:     serial.ToTypedMessage(ms),
		})
	}
	if c.HTTPUpgradeSettings != nil {
		hs, err := c.HTTPUpgradeSettings.Build()
		if err != nil {
			return nil, newError("Failed to build HTTPUpgrade config.").Base(err)
		}
		config.TransportSettings = append(config.TransportSettings, &internet.TransportConfig{
			ProtocolName: "httpupgrade",
			Settings:     serial.ToTypedMessage(hs),
		})
	}
	if c.MekyaSettings != nil {
		ms, err := c.MekyaSettings.Build()
		if err != nil {
			return nil, newError("Failed to build Mekya config.").Base(err)
		}
		config.TransportSettings = append(config.TransportSettings, &internet.TransportConfig{
			ProtocolName: "mekya",
			Settings:     serial.ToTypedMessage(ms),
		})
	}
	if c.SocketSettings != nil {
		ss, err := c.SocketSettings.Build()
		if err != nil {
			return nil, newError("Failed to build sockopt.").Base(err)
		}
		config.SocketSettings = ss
	}
	return config, nil
}

type ProxyConfig struct {
	Tag                 string `json:"tag"`
	TransportLayerProxy bool   `json:"transportLayer"`
}

// Build implements Buildable.
func (v *ProxyConfig) Build() (*internet.ProxyConfig, error) {
	if v.Tag == "" {
		return nil, newError("Proxy tag is not set.")
	}
	return &internet.ProxyConfig{
		Tag:                 v.Tag,
		TransportLayerProxy: v.TransportLayerProxy,
	}, nil
}
