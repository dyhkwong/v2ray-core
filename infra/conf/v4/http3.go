package v4

import (
	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon"
	"github.com/v2fly/v2ray-core/v5/proxy/http3"
)

type HTTP3ClientConfig struct {
	Address  *cfgcommon.Address `json:"address"`
	Port     uint16             `json:"port"`
	Level    byte               `json:"level"`
	Username string             `json:"username"`
	Password string             `json:"password"`
	Headers  map[string]string  `json:"headers"`
}

func (c *HTTP3ClientConfig) Build() (proto.Message, error) {
	if c.Address == nil {
		return nil, newError("missing server address")
	}
	return &http3.ClientConfig{
		Address:  c.Address.Build(),
		Port:     uint32(c.Port),
		Level:    uint32(c.Level),
		Username: c.Username,
		Password: c.Password,
		Headers:  c.Headers,
	}, nil
}
