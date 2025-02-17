package v4

import (
	"encoding/base64"

	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon"
	shadowsocks2022 "github.com/v2fly/v2ray-core/v5/proxy/shadowsocks2022"
)

type Shadowsocks2022Config struct {
	Method           string             `json:"method"`
	PSK              string             `json:"psk"`
	IPSK             []string           `json:"iPSK"`
	Address          *cfgcommon.Address `json:"address"`
	Port             uint16             `json:"port"`
	Plugin           string             `json:"plugin"`
	PluginOpts       string             `json:"pluginOpts"`
	PluginArgs       []string           `json:"pluginArgs"`
	PluginWorkingDir string             `json:"pluginWorkingDir"`
	UoT              bool               `json:"uot"`
}

func (c *Shadowsocks2022Config) Build() (proto.Message, error) {
	config := new(shadowsocks2022.ClientConfig)
	config.Method = c.Method
	psk, err := base64.StdEncoding.DecodeString(c.PSK)
	if err != nil {
		return nil, err
	}
	config.Psk = psk
	for _, ipsk := range c.IPSK {
		k, err := base64.StdEncoding.DecodeString(ipsk)
		if err != nil {
			return nil, err
		}
		config.Ipsk = append(config.Ipsk, k)
	}
	config.Address = c.Address.Build()
	config.Port = uint32(c.Port)
	config.Plugin = c.Plugin
	config.PluginOpts = c.PluginOpts
	config.PluginArgs = c.PluginArgs
	config.PluginWorkingDir = c.PluginWorkingDir
	config.Uot = c.UoT
	return config, nil
}
