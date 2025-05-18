package v4

import (
	"net"
	"strings"

	"github.com/golang/protobuf/proto"

	v2net "github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon/socketcfg"
	"github.com/v2fly/v2ray-core/v5/proxy/freedom"
)

type FreedomConfig struct {
	DomainStrategy      string              `json:"domainStrategy"`
	Timeout             *uint32             `json:"timeout"`
	Redirect            string              `json:"redirect"`
	UserLevel           uint32              `json:"userLevel"`
	ProtocolReplacement string              `json:"protocolReplacement"`
	Fragment            *socketcfg.Fragment `json:"fragment"`
	Noises              []*socketcfg.Noise  `json:"noises"`
	NoiseKeepalive      uint32              `json:"noiseKeepAlive"`
}

// Build implements Buildable
func (c *FreedomConfig) Build() (proto.Message, error) {
	config := new(freedom.Config)
	config.DomainStrategy = freedom.Config_AS_IS
	switch strings.ToLower(c.DomainStrategy) {
	case "useip", "use_ip", "use-ip":
		config.DomainStrategy = freedom.Config_USE_IP
	case "useip4", "useipv4", "use_ip4", "use_ipv4", "use_ip_v4", "use-ip4", "use-ipv4", "use-ip-v4":
		config.DomainStrategy = freedom.Config_USE_IP4
	case "useip6", "useipv6", "use_ip6", "use_ipv6", "use_ip_v6", "use-ip6", "use-ipv6", "use-ip-v6":
		config.DomainStrategy = freedom.Config_USE_IP6
	case "preferip4", "preferipv4", "prefer_ip4", "prefer_ipv4", "prefer_ip_v4", "prefer-ip4", "prefer-ipv4", "prefer-ip-v4":
		config.DomainStrategy = freedom.Config_PREFER_IP4
	case "preferip6", "preferipv6", "prefer_ip6", "prefer_ipv6", "prefer_ip_v6", "prefer-ip6", "prefer-ipv6", "prefer-ip-v6":
		config.DomainStrategy = freedom.Config_PREFER_IP6
	}

	if c.Timeout != nil {
		config.Timeout = *c.Timeout
	}
	config.UserLevel = c.UserLevel
	if len(c.Redirect) > 0 {
		host, portStr, err := net.SplitHostPort(c.Redirect)
		if err != nil {
			return nil, newError("invalid redirect address: ", c.Redirect, ": ", err).Base(err)
		}
		port, err := v2net.PortFromString(portStr)
		if err != nil {
			return nil, newError("invalid redirect port: ", c.Redirect, ": ", err).Base(err)
		}
		config.DestinationOverride = &freedom.DestinationOverride{
			Server: &protocol.ServerEndpoint{
				Port: uint32(port),
			},
		}

		if len(host) > 0 {
			config.DestinationOverride.Server.Address = v2net.NewIPOrDomain(v2net.ParseAddress(host))
		}
	}
	switch strings.ToLower(c.ProtocolReplacement) {
	case "forcetcp", "force_tcp", "force-tcp":
		config.ProtocolReplacement = freedom.ProtocolReplacement_FORCE_TCP
	case "forceudp", "force_udp", "force-udp":
		config.ProtocolReplacement = freedom.ProtocolReplacement_FORCE_UDP
	case "identity", "":
		config.ProtocolReplacement = freedom.ProtocolReplacement_IDENTITY
	default:
		return nil, newError("invalid protocol replacement: ", c.ProtocolReplacement)
	}

	if c.Fragment != nil {
		config.Fragment = c.Fragment.Build()
	}
	if c.Noises != nil {
		for _, noise := range c.Noises {
			config.Noises = append(config.Noises, noise.Build())
		}
	}
	config.NoiseKeepAlive = uint64(c.NoiseKeepalive)

	return config, nil
}
