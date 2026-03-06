package v4

import (
	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon"
	"github.com/v2fly/v2ray-core/v5/proxy/juicity"
)

type JuicityClientConfig struct {
	Address  *cfgcommon.Address `json:"address"`
	Port     uint16             `json:"port"`
	UUID     string             `json:"uuid"`
	Password string             `json:"password"`
}

func (c *JuicityClientConfig) Build() (proto.Message, error) {
	if c.Address == nil {
		return nil, newError("missing server address")
	}
	return &juicity.ClientConfig{
		Address:  c.Address.Build(),
		Port:     uint32(c.Port),
		Uuid:     c.UUID,
		Password: c.Password,
	}, nil
}
