package v4

import (
	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon"
	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon/tlscfg"
	"github.com/v2fly/v2ray-core/v5/proxy/http3"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

type HTTP3ClientConfig struct {
	Address     *cfgcommon.Address `json:"address"`
	Port        uint16             `json:"port"`
	Level       byte               `json:"level"`
	Username    string             `json:"username"`
	Password    string             `json:"password"`
	Headers     map[string]string  `json:"headers"`
	TLSSettings *tlscfg.TLSConfig  `json:"tlsSettings"`
}

func (c *HTTP3ClientConfig) Build() (proto.Message, error) {
	config := &http3.ClientConfig{
		Address:  c.Address.Build(),
		Port:     uint32(c.Port),
		Level:    uint32(c.Level),
		Username: c.Username,
		Password: c.Password,
		Headers:  c.Headers,
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
