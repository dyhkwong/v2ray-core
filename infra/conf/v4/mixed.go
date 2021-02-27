package v4

import (
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/common/net/packetaddr"
	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon"
	"github.com/v2fly/v2ray-core/v5/proxy/mixed"
)

type MixedAccount struct {
	Username string `json:"user"`
	Password string `json:"pass"`
}

func (v *MixedAccount) Build() *mixed.Account {
	return &mixed.Account{
		Username: v.Username,
		Password: v.Password,
	}
}

type MixedServerConfig struct {
	AuthMethod     string             `json:"auth"`
	Accounts       []*MixedAccount    `json:"accounts"`
	UDP            bool               `json:"udp"`
	Host           *cfgcommon.Address `json:"ip"`
	Timeout        uint32             `json:"timeout"`
	UserLevel      uint32             `json:"userLevel"`
	Transparent    bool               `json:"allowTransparent"`
	PacketEncoding string             `json:"packetEncoding"`
}

func (v *MixedServerConfig) Build() (proto.Message, error) {
	config := new(mixed.ServerConfig)
	switch v.AuthMethod {
	case AuthMethodNoAuth:
		config.AuthType = mixed.AuthType_NO_AUTH
	case AuthMethodUserPass:
		config.AuthType = mixed.AuthType_PASSWORD
	default:
		// newError("unknown socks auth method: ", v.AuthMethod, ". Default to noauth.").AtWarning().WriteToLog()
		config.AuthType = mixed.AuthType_NO_AUTH
	}

	if len(v.Accounts) > 0 {
		config.Accounts = make(map[string]string, len(v.Accounts))
		for _, account := range v.Accounts {
			config.Accounts[account.Username] = account.Password
		}
	}
	config.UdpEnabled = v.UDP
	if v.Host != nil {
		config.Address = v.Host.Build()
	}
	config.Timeout = v.Timeout
	config.UserLevel = v.UserLevel
	config.AllowTransparent = v.Transparent

	switch strings.ToLower(v.PacketEncoding) {
	case "packet":
		config.PacketEncoding = packetaddr.PacketAddrType_Packet
	case "", "none":
		config.PacketEncoding = packetaddr.PacketAddrType_None
	}

	return config, nil
}
