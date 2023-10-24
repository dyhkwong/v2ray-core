//go:build tun && linux && (amd64 || arm64)

package conf

import (
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v4/app/tun"
	"github.com/v2fly/v2ray-core/v4/common/net/packetaddr"
	"github.com/v2fly/v2ray-core/v4/infra/conf/rule"
)

type TUNConfig struct {
	Name                  string          `json:"name"`
	MTU                   uint32          `json:"mtu"`
	Level                 uint32          `json:"level"`
	PacketEncoding        string          `json:"packetEncoding"`
	Tag                   string          `json:"tag"`
	IPs                   []string        `json:"ips"`
	Routes                []string        `json:"routes"`
	EnablePromiscuousMode bool            `json:"enablePromiscuousMode"`
	EnableSpoofing        bool            `json:"enableSpoofing"`
	SocketSettings        *SocketConfig   `json:"sockopt"`
	SniffingConfig        *SniffingConfig `json:"sniffing"`
}

func (t *TUNConfig) Build() (proto.Message, error) {
	config := new(tun.Config)
	for _, ip := range t.IPs {
		parsedIP, err := rule.ParseIP(ip)
		if err != nil {
			return nil, err
		}
		config.Ips = append(config.Ips, parsedIP)
	}
	for _, route := range t.Routes {
		parsedRoute, err := rule.ParseIP(route)
		if err != nil {
			return nil, err
		}
		config.Routes = append(config.Routes, parsedRoute)
	}
	if t.SocketSettings != nil {
		ss, err := t.SocketSettings.Build()
		if err != nil {
			return nil, newError("Failed to build sockopt.").Base(err)
		}
		config.SocketSettings = ss
	}
	if t.SniffingConfig != nil {
		sc, err := t.SniffingConfig.Build()
		if err != nil {
			return nil, newError("failed to build sniffing config").Base(err)
		}
		config.SniffingSettings = sc
	}
	config.Name = t.Name
	config.Mtu = t.MTU
	config.UserLevel = t.Level
	switch strings.ToLower(t.PacketEncoding) {
	case "packet":
		config.PacketEncoding = packetaddr.PacketAddrType_Packet
	case "", "none":
		config.PacketEncoding = packetaddr.PacketAddrType_None
	}
	config.Tag = t.Tag
	config.EnablePromiscuousMode = t.EnablePromiscuousMode
	config.EnableSpoofing = t.EnableSpoofing
	return config, nil
}
