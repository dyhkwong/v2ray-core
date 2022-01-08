package v4

import (
	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon"
	"github.com/v2fly/v2ray-core/v5/proxy/shadowsocks_2022"
)

type Shadowsocks2022ServerConfig struct {
	Method     string                       `json:"method"`
	Key        string                       `json:"key"`
	Level      byte                         `json:"level"`
	Email      string                       `json:"email"`
	Network    *cfgcommon.NetworkList       `json:"network"`
	Users      []*Shadowsocks2022UserConfig `json:"users"`
	Plugin     string                       `json:"plugin"`
	PluginOpts string                       `json:"pluginOpts"`
	PluginArgs *cfgcommon.StringList        `json:"pluginArgs"`
}

func (v *Shadowsocks2022ServerConfig) Build() (proto.Message, error) {
	var network []net.Network
	if v.Network != nil {
		network = v.Network.Build()
	}
	config := &shadowsocks_2022.ServerConfig{
		Method:  v.Method,
		Key:     v.Key,
		Level:   int32(v.Level),
		Email:   v.Email,
		Network: network,
	}
	config.Plugin = v.Plugin
	config.PluginOpts = v.PluginOpts
	if v.PluginArgs != nil && len(*v.PluginArgs) > 0 {
		config.PluginArgs = *v.PluginArgs
	}
	return config, nil
}

type Shadowsocks2022MultiUserServerConfig struct {
	Method     string                       `json:"method"`
	Key        string                       `json:"key"`
	Network    *cfgcommon.NetworkList       `json:"network"`
	Users      []*Shadowsocks2022UserConfig `json:"users"`
	Plugin     string                       `json:"plugin"`
	PluginOpts string                       `json:"pluginOpts"`
	PluginArgs *cfgcommon.StringList        `json:"pluginArgs"`
}

type Shadowsocks2022UserConfig struct {
	Key   string `json:"key"`
	Level byte   `json:"level"`
	Email string `json:"email"`
}

func (v *Shadowsocks2022MultiUserServerConfig) Build() (proto.Message, error) {
	var network []net.Network
	if v.Network != nil {
		network = v.Network.Build()
	}
	var users []*shadowsocks_2022.User
	for _, user := range v.Users {
		users = append(users, &shadowsocks_2022.User{
			Key:   user.Key,
			Level: int32(user.Level),
			Email: user.Email,
		})
	}
	config := &shadowsocks_2022.MultiUserServerConfig{
		Method:  v.Method,
		Key:     v.Key,
		Network: network,
		Users:   users,
	}
	config.Plugin = v.Plugin
	config.PluginOpts = v.PluginOpts
	if v.PluginArgs != nil && len(*v.PluginArgs) > 0 {
		config.PluginArgs = *v.PluginArgs
	}
	return config, nil
}

type Shadowsocks2022RelayDestinationConfig struct {
	Key     string             `json:"key"`
	Address *cfgcommon.Address `json:"address"`
	Port    uint16             `json:"port"`
	Email   string             `json:"email"`
	Level   byte               `json:"level"`
}

type Shadowsocks2022RelayServerConfig struct {
	Method       string                                   `json:"method"`
	Key          string                                   `json:"key"`
	Network      *cfgcommon.NetworkList                   `json:"network"`
	Destinations []*Shadowsocks2022RelayDestinationConfig `json:"destinations"`
	Plugin       string                                   `json:"plugin"`
	PluginOpts   string                                   `json:"pluginOpts"`
	PluginArgs   *cfgcommon.StringList                    `json:"pluginArgs"`
}

func (v *Shadowsocks2022RelayServerConfig) Build() (proto.Message, error) {
	var network []net.Network
	if v.Network != nil {
		network = v.Network.Build()
	}
	if v.Destinations == nil {
		return nil, newError("shadowsocks 2022: missing relay destinations")
	}
	var destinations []*shadowsocks_2022.RelayDestination
	for _, destination := range v.Destinations {
		destinations = append(destinations, &shadowsocks_2022.RelayDestination{
			Key:     destination.Key,
			Address: destination.Address.Build(),
			Port:    uint32(destination.Port),
			Email:   destination.Email,
			Level:   int32(destination.Level),
		})
	}
	config := &shadowsocks_2022.RelayServerConfig{
		Method:       v.Method,
		Key:          v.Key,
		Network:      network,
		Destinations: destinations,
	}
	config.Plugin = v.Plugin
	config.PluginOpts = v.PluginOpts
	if v.PluginArgs != nil && len(*v.PluginArgs) > 0 {
		config.PluginArgs = *v.PluginArgs
	}
	return config, nil
}

type Shadowsocks2022ClientConfig struct {
	Address    *cfgcommon.Address    `json:"address"`
	Port       uint16                `json:"port"`
	Method     string                `json:"method"`
	Key        string                `json:"key"`
	Plugin     string                `json:"plugin"`
	PluginOpts string                `json:"pluginOpts"`
	PluginArgs *cfgcommon.StringList `json:"pluginArgs"`
}

func (v *Shadowsocks2022ClientConfig) Build() (proto.Message, error) {
	if v.Address == nil {
		return nil, newError("shadowsocks 2022: missing server address")
	}
	config := &shadowsocks_2022.ClientConfig{
		Address: v.Address.Build(),
		Port:    uint32(v.Port),
		Method:  v.Method,
		Key:     v.Key,
	}
	config.Plugin = v.Plugin
	config.PluginOpts = v.PluginOpts
	if v.PluginArgs != nil && len(*v.PluginArgs) > 0 {
		config.PluginArgs = *v.PluginArgs
	}
	return config, nil
}
