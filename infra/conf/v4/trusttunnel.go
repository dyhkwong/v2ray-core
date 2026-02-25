package v4

import (
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon"
	"github.com/v2fly/v2ray-core/v5/proxy/trusttunnel"
)

type TrustTunnelClientConfig struct {
	Address        *cfgcommon.Address `json:"address"`
	Port           uint16             `json:"port"`
	Level          byte               `json:"level"`
	Username       string             `json:"username"`
	Password       string             `json:"password"`
	HTTP3          bool               `json:"http3"`
	DomainStrategy string             `json:"domainStrategy"`
}

func (c *TrustTunnelClientConfig) Build() (proto.Message, error) {
	if c.Address == nil {
		return nil, newError("missing server address")
	}
	config := &trusttunnel.ClientConfig{
		Address:  c.Address.Build(),
		Port:     uint32(c.Port),
		Level:    uint32(c.Level),
		Username: c.Username,
		Password: c.Password,
		Http3:    c.HTTP3,
	}
	switch strings.ToLower(c.DomainStrategy) {
	case "useip", "":
		config.DomainStrategy = trusttunnel.ClientConfig_USE_IP
	case "useipv4":
		config.DomainStrategy = trusttunnel.ClientConfig_USE_IP4
	case "useipv6":
		config.DomainStrategy = trusttunnel.ClientConfig_USE_IP6
	case "preferipv4":
		config.DomainStrategy = trusttunnel.ClientConfig_PREFER_IP4
	case "preferipv6":
		config.DomainStrategy = trusttunnel.ClientConfig_PREFER_IP6
	default:
		return nil, newError("unsupported domain strategy: ", c.DomainStrategy)
	}
	return config, nil
}
