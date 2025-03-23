package v4

import (
	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon"
	"github.com/v2fly/v2ray-core/v5/proxy/shadowtls"
)

type ShadowTLSClientConfig struct {
	Address  *cfgcommon.Address `json:"address"`
	Port     uint16             `json:"port"`
	Password string             `json:"password"`
	Version  uint32             `json:"version"`
}

func (c *ShadowTLSClientConfig) Build() (proto.Message, error) {
	if c.Address == nil {
		return nil, newError("missing server address")
	}
	return &shadowtls.ClientConfig{
		Address:  c.Address.Build(),
		Port:     uint32(c.Port),
		Password: c.Password,
		Version:  c.Version,
	}, nil
}
