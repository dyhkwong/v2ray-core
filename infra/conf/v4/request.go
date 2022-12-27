package v4

import (
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/common/serial"
	"github.com/v2fly/v2ray-core/v5/transport/internet/request/assembler/packetconn"
	"github.com/v2fly/v2ray-core/v5/transport/internet/request/assembler/simple"
	"github.com/v2fly/v2ray-core/v5/transport/internet/request/assembly"
	"github.com/v2fly/v2ray-core/v5/transport/internet/request/roundtripper/httprt"
)

type RequestConfig struct {
	Assembler    *AssemblerConfig    `json:"assembler"`
	RoundTripper *RoundTripperConfig `json:"roundTripper"`
}

// Build implements Buildable.
func (c *RequestConfig) Build() (proto.Message, error) {
	config := new(assembly.Config)
	if c.Assembler != nil {
		assembler, err := c.Assembler.Build()
		if err != nil {
			return nil, err
		}
		config.Assembler = serial.ToTypedMessage(assembler)
	}
	if c.RoundTripper != nil {
		roundTripper, err := c.RoundTripper.Build()
		if err != nil {
			return nil, err
		}
		config.Roundtripper = serial.ToTypedMessage(roundTripper)
	}
	return config, nil
}

type AssemblerConfig struct {
	Type                     string                 `json:"type"`
	PacketConnClientSettings PacketConnClientConfig `json:"packetconnClientSettings"`
	PacketConnServerSettings PacketConnServerConfig `json:"packetconnServerSettings"`
	SimpleClientSettings     SimpleClientConfig     `json:"simpleClientSettings"`
	SimpleServerSettings     SimpleServerConfig     `json:"simpleServerSettings"`
}

func (c *AssemblerConfig) Build() (proto.Message, error) {
	switch strings.ToLower(c.Type) {
	case "packetconn.client":
		return c.PacketConnClientSettings.Build()
	case "packetconn.server":
		return c.PacketConnServerSettings.Build()
	case "simple.client":
		return c.SimpleClientSettings.Build()
	case "simple.server":
		return c.SimpleServerSettings.Build()
	default:
		return nil, newError("unknown assembler type: ", c.Type)
	}
}

type RoundTripperConfig struct {
	Type                 string             `json:"type"`
	HttprtClientSettings HTTPRTClientConfig `json:"httprtClientSettings"`
	HttprtServerSettings HTTPRTServerConfig `json:"httprtServerSettings"`
}

func (c *RoundTripperConfig) Build() (proto.Message, error) {
	switch strings.ToLower(c.Type) {
	case "httprt.client":
		return c.HttprtClientSettings.Build()
	case "httprt.server":
		return c.HttprtServerSettings.Build()
	default:
		return nil, newError("unknown roundTripper type: ", c.Type)
	}
}

type PacketConnClientConfig struct {
	UnderlyingNetwork      string `json:"underlyingNetwork"`
	MaxWriteDelay          int32  `json:"maxWriteDelay"`
	MaxRequestSize         int32  `json:"maxRequestSize"`
	PollingIntervalInitial int32  `json:"pollingIntervalInitial"`

	KCPSettings  *KCPConfig  `json:"kcpSettings"`
	DTLSSettings *DTLSConfig `json:"dtlsSettings"`
}

func (c *PacketConnClientConfig) Build() (proto.Message, error) {
	config := &packetconn.ClientConfig{
		UnderlyingTransportName: c.UnderlyingNetwork,
		MaxWriteDelay:           c.MaxWriteDelay,
		MaxRequestSize:          c.MaxRequestSize,
		PollingIntervalInitial:  c.PollingIntervalInitial,
	}
	switch strings.ToLower(c.UnderlyingNetwork) {
	case "kcp", "mkcp":
		if c.KCPSettings != nil {
			underlyingTransportSettings, err := c.KCPSettings.Build()
			if err != nil {
				return nil, err
			}
			config.UnderlyingTransportSetting = serial.ToTypedMessage(underlyingTransportSettings)
		}
	case "dtls":
		if c.DTLSSettings != nil {
			underlyingTransportSettings, err := c.DTLSSettings.Build()
			if err != nil {
				return nil, err
			}
			config.UnderlyingTransportSetting = serial.ToTypedMessage(underlyingTransportSettings)
		}
	default:
		return nil, newError("unknown underlyingNetwork: ", c.UnderlyingNetwork)
	}
	return config, nil
}

type PacketConnServerConfig struct {
	UnderlyingNetwork              string `json:"underlyingNetwork"`
	MaxWriteSize                   int32  `json:"maxWriteSize"`
	MaxWriteDurationMs             int32  `json:"maxWriteDurationMs"`
	MaxSimultaneousWriteConnection int32  `json:"maxSimultaneousWriteConnection"`
	PacketWritingBuffer            int32  `json:"packetWritingBuffer"`

	KCPSettings  *KCPConfig  `json:"kcpSettings"`
	DTLSSettings *DTLSConfig `json:"dtlsSettings"`
}

func (c *PacketConnServerConfig) Build() (proto.Message, error) {
	config := &packetconn.ServerConfig{
		UnderlyingTransportName:        c.UnderlyingNetwork,
		MaxWriteSize:                   c.MaxWriteSize,
		MaxWriteDurationMs:             c.MaxWriteDurationMs,
		MaxSimultaneousWriteConnection: c.MaxSimultaneousWriteConnection,
		PacketWritingBuffer:            c.PacketWritingBuffer,
	}
	switch strings.ToLower(c.UnderlyingNetwork) {
	case "kcp", "mkcp":
		if c.KCPSettings != nil {
			underlyingTransportSettings, err := c.KCPSettings.Build()
			if err != nil {
				return nil, err
			}
			config.UnderlyingTransportSetting = serial.ToTypedMessage(underlyingTransportSettings)
		}
	case "dtls":
		if c.DTLSSettings != nil {
			underlyingTransportSettings, err := c.DTLSSettings.Build()
			if err != nil {
				return nil, err
			}
			config.UnderlyingTransportSetting = serial.ToTypedMessage(underlyingTransportSettings)
		}
	default:
		return nil, newError("unknown underlyingNetwork: ", c.UnderlyingNetwork)
	}
	return config, nil
}

type SimpleClientConfig struct {
	MaxWriteSize             int32   `json:"maxWriteSize"`
	WaitSubsequentWriteMs    int32   `json:"waitSubsequentWriteMs"`
	InitialPollingIntervalMs int32   `json:"initialPollingIntervalMs"`
	MaxPollingIntervalMs     int32   `json:"maxPollingIntervalMs"`
	MinPollingIntervalMs     int32   `json:"minPollingIntervalMs"`
	BackoffFactor            float64 `json:"backoffFactor"`
	FailedRetryIntervalMs    int32   `json:"failedRetryIntervalMs"`
}

func (c *SimpleClientConfig) Build() (proto.Message, error) {
	config := &simple.ClientConfig{
		MaxWriteSize:             c.MaxWriteSize,
		WaitSubsequentWriteMs:    c.WaitSubsequentWriteMs,
		InitialPollingIntervalMs: c.InitialPollingIntervalMs,
		MaxPollingIntervalMs:     c.MaxPollingIntervalMs,
		MinPollingIntervalMs:     c.MinPollingIntervalMs,
		BackoffFactor:            float32(c.BackoffFactor),
		FailedRetryIntervalMs:    c.FailedRetryIntervalMs,
	}
	return config, nil
}

type SimpleServerConfig struct {
	MaxWriteSize int32 `json:"maxWriteSize"`
}

func (c *SimpleServerConfig) Build() (proto.Message, error) {
	config := &simple.ServerConfig{
		MaxWriteSize: c.MaxWriteSize,
	}
	return config, nil
}

type HTTPRTClientConfig struct {
	HTTP       *HTTPRTConfig `json:"http"`
	AllowHTTP  bool          `json:"allowHTTP"`
	H2PoolSize int32         `json:"h2PoolSize"`
}

func (c *HTTPRTClientConfig) Build() (proto.Message, error) {
	config := &httprt.ClientConfig{
		AllowHttp:  c.AllowHTTP,
		H2PoolSize: c.H2PoolSize,
	}
	if c.HTTP != nil {
		config.Http = &httprt.HTTPConfig{
			Path:      c.HTTP.Path,
			UrlPrefix: c.HTTP.URLPrefix,
		}
	}
	return config, nil
}

type HTTPRTServerConfig struct {
	HTTP                 *HTTPRTConfig `json:"http"`
	NoDecodingSessionTag bool          `json:"noDecodingSessionTag"`
}

func (c *HTTPRTServerConfig) Build() (proto.Message, error) {
	config := &httprt.ServerConfig{
		NoDecodingSessionTag: c.NoDecodingSessionTag,
	}
	if c.HTTP != nil {
		config.Http = &httprt.HTTPConfig{
			Path:      c.HTTP.Path,
			UrlPrefix: c.HTTP.URLPrefix,
		}
	}
	return config, nil
}

type HTTPRTConfig struct {
	Path      string `json:"path"`
	URLPrefix string `json:"urlPrefix"`
}
