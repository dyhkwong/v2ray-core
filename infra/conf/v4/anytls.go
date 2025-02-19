package v4

import (
	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon"
	"github.com/v2fly/v2ray-core/v5/proxy/anytls"
)

type AnyTLSClientConfig struct {
	Address                  *cfgcommon.Address `json:"address"`
	Port                     uint16             `json:"port"`
	Password                 string             `json:"password"`
	IdleSessionCheckInterval int64              `json:"idleSessionCheckInterval"`
	IdleSessionTimeout       int64              `json:"idleSessionTimeout"`
	MinIdleSession           int64              `json:"minIdleSession"`
}

func (c *AnyTLSClientConfig) Build() (proto.Message, error) {
	if c.Address == nil {
		return nil, newError("missing server address")
	}
	config := &anytls.ClientConfig{
		Address:                  c.Address.Build(),
		Port:                     uint32(c.Port),
		Password:                 c.Password,
		IdleSessionCheckInterval: c.IdleSessionCheckInterval,
		IdleSessionTimeout:       c.IdleSessionTimeout,
		MinIdleSession:           c.MinIdleSession,
	}
	return config, nil
}

type AnyTLSUserConfig struct {
	Password string `json:"password"`
	Email    string `json:"email"`
	Level    byte   `json:"level"`
}

type AnyTLSServerConfig struct {
	Users         []AnyTLSUserConfig `json:"users"`
	PaddingScheme []string           `json:"paddingScheme"`
}

func (c *AnyTLSServerConfig) Build() (proto.Message, error) {
	config := &anytls.ServerConfig{
		PaddingScheme: c.PaddingScheme,
	}
	for _, user := range c.Users {
		config.Users = append(config.Users, &anytls.User{
			Password: user.Password,
			Level:    int32(user.Level),
			Email:    user.Email,
		})
	}
	return config, nil
}
