package v4

import (
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon"
	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon/tlscfg"
	"github.com/v2fly/v2ray-core/v5/proxy/http3"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

type HTTP3ClientConfig struct {
	Address        *cfgcommon.Address `json:"address"`
	Port           uint16             `json:"port"`
	Level          byte               `json:"level"`
	Username       *string            `json:"username"`
	Password       *string            `json:"password"`
	Headers        map[string]string  `json:"headers"`
	TLSSettings    *tlscfg.TLSConfig  `json:"tlsSettings"`
	TrustTunnelUDP bool               `json:"trustTunnelUDP"`
	DomainStrategy string             `json:"domainStrategy"`
}

func (c *HTTP3ClientConfig) Build() (proto.Message, error) {
	if c.Address == nil {
		return nil, newError("missing server address")
	}
	config := &http3.ClientConfig{
		Address:        c.Address.Build(),
		Port:           uint32(c.Port),
		Level:          uint32(c.Level),
		Username:       c.Username,
		Password:       c.Password,
		Headers:        c.Headers,
		TrustTunnelUdp: c.TrustTunnelUDP,
	}
	if c.TLSSettings != nil {
		tlsSettings, err := c.TLSSettings.Build()
		if err != nil {
			return nil, err
		}
		config.TlsSettings = tlsSettings.(*tls.Config)
	}

	config.TrustTunnelUdp = c.TrustTunnelUDP
	switch strings.ToLower(c.DomainStrategy) {
	case "useip", "":
		config.DomainStrategy = http3.ClientConfig_USE_IP
	case "useipv4":
		config.DomainStrategy = http3.ClientConfig_USE_IP4
	case "useipv6":
		config.DomainStrategy = http3.ClientConfig_USE_IP6
	case "preferipv4":
		config.DomainStrategy = http3.ClientConfig_PREFER_IP4
	case "preferipv6":
		config.DomainStrategy = http3.ClientConfig_PREFER_IP6
	default:
		return nil, newError("unsupported domain strategy: ", c.DomainStrategy)
	}

	return config, nil
}
