package v4

import (
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/common/net/packetaddr"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
	"github.com/v2fly/v2ray-core/v5/common/serial"
	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon"
	"github.com/v2fly/v2ray-core/v5/proxy/shadowsocks"
	"github.com/v2fly/v2ray-core/v5/proxy/shadowsocks_2022"
)

type ShadowsocksUserConfig struct {
	Password string             `json:"password"`
	Level    byte               `json:"level"`
	Email    string             `json:"email"`
	Address  *cfgcommon.Address `json:"address"`
	Port     uint16             `json:"port"`
}

type ShadowsocksServerConfig struct {
	Cipher         string                   `json:"method"`
	Password       string                   `json:"password"`
	UDP            bool                     `json:"udp"`
	Level          byte                     `json:"level"`
	Email          string                   `json:"email"`
	NetworkList    *cfgcommon.NetworkList   `json:"network"`
	IVCheck        bool                     `json:"ivCheck"`
	PacketEncoding string                   `json:"packetEncoding"`
	Clients        []*ShadowsocksUserConfig `json:"clients"`
	Users          []*ShadowsocksUserConfig `json:"users"`
}

func (v *ShadowsocksServerConfig) Build() (proto.Message, error) {
	if strings.HasPrefix(v.Cipher, strings.ToLower("2022-blake3-")) {
		if v.Users == nil {
			v.Users = v.Clients
		}
		if len(v.Users) > 0 {
			if !strings.HasPrefix(v.Cipher, strings.ToLower("2022-blake3-aes-")) {
				return nil, newError("shadowsocks 2022 (multi-user): only 2022-blake3-aes-*-gcm methods are supported")
			}
			if v.Users[0].Address == nil {
				config := new(shadowsocks_2022.MultiUserServerConfig)
				config.Method = strings.ToLower(v.Cipher)
				config.Key = v.Password
				config.Network = v.NetworkList.Build()
				for _, user := range v.Users {
					config.Users = append(config.Users, &shadowsocks_2022.User{
						Key:   user.Password,
						Level: int32(user.Level),
						Email: user.Email,
					})
				}
				return config, nil
			} else {
				config := new(shadowsocks_2022.RelayServerConfig)
				config.Method = v.Cipher
				config.Key = v.Password
				config.Network = v.NetworkList.Build()
				for _, user := range v.Users {
					config.Destinations = append(config.Destinations, &shadowsocks_2022.RelayDestination{
						Key:     user.Password,
						Level:   int32(user.Level),
						Email:   user.Email,
						Address: user.Address.Build(),
						Port:    uint32(user.Port),
					})
				}
				return config, nil
			}
		}
		config := new(shadowsocks_2022.ServerConfig)
		config.Method = strings.ToLower(v.Cipher)
		config.Key = v.Password
		config.Level = int32(v.Level)
		config.Email = v.Email
		config.Network = v.NetworkList.Build()
		return config, nil
	}

	config := new(shadowsocks.ServerConfig)
	config.UdpEnabled = v.UDP
	config.Network = v.NetworkList.Build()

	if v.Password == "" {
		return nil, newError("Shadowsocks password is not specified.")
	}
	account := &shadowsocks.Account{
		Password: v.Password,
		IvCheck:  v.IVCheck,
	}
	account.CipherType = shadowsocks.CipherFromString(v.Cipher)
	if account.CipherType == shadowsocks.CipherType_UNKNOWN {
		return nil, newError("unknown cipher method: ", v.Cipher)
	}

	config.User = &protocol.User{
		Email:   v.Email,
		Level:   uint32(v.Level),
		Account: serial.ToTypedMessage(account),
	}

	switch strings.ToLower(v.PacketEncoding) {
	case "packet":
		config.PacketEncoding = packetaddr.PacketAddrType_Packet
	case "", "none":
		config.PacketEncoding = packetaddr.PacketAddrType_None
	}

	return config, nil
}

type ShadowsocksServerTarget struct {
	Address                        *cfgcommon.Address `json:"address"`
	Port                           uint16             `json:"port"`
	Cipher                         string             `json:"method"`
	Password                       string             `json:"password"`
	Email                          string             `json:"email"`
	Ota                            bool               `json:"ota"`
	Level                          byte               `json:"level"`
	IVCheck                        bool               `json:"ivCheck"`
	ExperimentReducedIvHeadEntropy bool               `json:"experimentReducedIvHeadEntropy"`
}

type ShadowsocksClientConfig struct {
	Servers []*ShadowsocksServerTarget `json:"servers"`
}

func (v *ShadowsocksClientConfig) Build() (proto.Message, error) {
	if len(v.Servers) == 0 {
		return nil, newError("0 Shadowsocks server configured.")
	}

	if len(v.Servers) == 1 {
		server := v.Servers[0]

		if server.Address == nil {
			return nil, newError("Shadowsocks server address is not set.")
		}
		if server.Port == 0 {
			return nil, newError("Invalid Shadowsocks port.")
		}
		if server.Password == "" {
			return nil, newError("Shadowsocks password is not specified.")
		}

		if strings.HasPrefix(server.Cipher, strings.ToLower("2022-blake3-")) {
			config := new(shadowsocks_2022.ClientConfig)
			config.Address = server.Address.Build()
			config.Port = uint32(server.Port)
			config.Method = strings.ToLower(server.Cipher)
			config.Key = server.Password
			return config, nil
		}
	}

	config := new(shadowsocks.ClientConfig)

	serverSpecs := make([]*protocol.ServerEndpoint, len(v.Servers))
	for idx, server := range v.Servers {
		if strings.HasPrefix(server.Cipher, strings.ToLower("2022-blake3-")) {
			return nil, newError("Shadowsocks 2022 accept no multi servers")
		}
		if server.Address == nil {
			return nil, newError("Shadowsocks server address is not set.")
		}
		if server.Port == 0 {
			return nil, newError("Invalid Shadowsocks port.")
		}
		if server.Password == "" {
			return nil, newError("Shadowsocks password is not specified.")
		}
		account := &shadowsocks.Account{
			Password: server.Password,
		}
		account.CipherType = shadowsocks.CipherFromString(server.Cipher)
		if account.CipherType == shadowsocks.CipherType_UNKNOWN {
			return nil, newError("unknown cipher method: ", server.Cipher)
		}

		account.IvCheck = server.IVCheck
		account.ExperimentReducedIvHeadEntropy = server.ExperimentReducedIvHeadEntropy

		ss := &protocol.ServerEndpoint{
			Address: server.Address.Build(),
			Port:    uint32(server.Port),
			User: []*protocol.User{
				{
					Level:   uint32(server.Level),
					Email:   server.Email,
					Account: serial.ToTypedMessage(account),
				},
			},
		}

		serverSpecs[idx] = ss
	}

	config.Server = serverSpecs

	return config, nil
}
