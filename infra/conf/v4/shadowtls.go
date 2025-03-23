package v4

import (
	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon"
	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon/tlscfg"
	"github.com/v2fly/v2ray-core/v5/proxy/shadowtls"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

type ShadowTLSClientConfig struct {
	Address     *cfgcommon.Address `json:"address"`
	Port        uint16             `json:"port"`
	Password    string             `json:"password"`
	Version     uint32             `json:"version"`
	TLSSettings *tlscfg.TLSConfig  `json:"tlsSettings"`
}

func (c *ShadowTLSClientConfig) Build() (proto.Message, error) {
	if c.Address == nil {
		return nil, newError("missing server address")
	}
	config := &shadowtls.ClientConfig{
		Address:  c.Address.Build(),
		Port:     uint32(c.Port),
		Password: c.Password,
		Version:  c.Version,
	}
	if c.TLSSettings != nil {
		tlsSettings, err := c.TLSSettings.Build()
		if err != nil {
			return nil, err
		}
		config.TlsSettings = tlsSettings.(*tls.Config)
	}
	return config, nil
}
