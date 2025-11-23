package v4

import (
	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon"
	"github.com/v2fly/v2ray-core/v5/proxy/mieru"
)

type MieruClientConfig struct {
	Address        *cfgcommon.Address `json:"address"`
	Port           uint16             `json:"port"`
	PortRange      []string           `json:"portRange"`
	Username       string             `json:"username"`
	Password       string             `json:"password"`
	Protocol       string             `json:"protocol"`
	Multiplexing   string             `json:"multiplexing"`
	HandshakeMode  string             `json:"handshakeMode"`
	TrafficPattern string             `json:"trafficPattern"`
}

type MieruPortBindingConfig struct {
	Port      uint16 `json:"port"`
	PortRange string `json:"portRange"`
	Protocol  string `json:"protocol"`
}

func (c *MieruClientConfig) Build() (proto.Message, error) {
	if c.Address == nil {
		return nil, newError("missing server address")
	}
	config := &mieru.ClientConfig{
		Address:        c.Address.Build(),
		Port:           uint32(c.Port),
		PortRange:      c.PortRange,
		Username:       c.Username,
		Password:       c.Password,
		Protocol:       c.Protocol,
		Multiplexing:   c.Multiplexing,
		HandshakeMode:  c.HandshakeMode,
		TrafficPattern: c.TrafficPattern,
	}
	return config, nil
}
