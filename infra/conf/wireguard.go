//go:build !dragonfly

package conf

import (
	"encoding/base64"
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v4/common/packetswitch/gvisorstack"
	"github.com/v2fly/v2ray-core/v4/infra/conf/rule"
	"github.com/v2fly/v2ray-core/v4/proxy/wireguard/outbound"
	"github.com/v2fly/v2ray-core/v4/proxy/wireguard/wgcommon"
)

type WireGuardOutboundConfig struct {
	WGDevice              *WireGuardDeviceConfig `json:"wgDevice"`
	Stack                 *WireGuardStackConfig  `json:"stack"`
	ListenOnSystemNetwork bool                   `json:"listenOnSystemNetwork"`
	DomainStrategy        string                 `json:"domainStrategy"`
}

func (c *WireGuardOutboundConfig) Build() (proto.Message, error) {
	config := &outbound.Config{
		ListenOnSystemNetwork: c.ListenOnSystemNetwork,
	}
	config.DomainStrategy = outbound.Config_AS_IS
	switch strings.ToLower(c.DomainStrategy) {
	case "useip", "use_ip", "use-ip":
		config.DomainStrategy = outbound.Config_USE_IP
	case "useip4", "useipv4", "use_ip4", "use_ipv4", "use_ip_v4", "use-ip4", "use-ipv4", "use-ip-v4":
		config.DomainStrategy = outbound.Config_USE_IP4
	case "useip6", "useipv6", "use_ip6", "use_ipv6", "use_ip_v6", "use-ip6", "use-ipv6", "use-ip-v6":
		config.DomainStrategy = outbound.Config_USE_IP6
	}
	if c.WGDevice != nil {
		wgDevice, err := c.WGDevice.Build()
		if err != nil {
			return nil, err
		}
		config.WgDevice = wgDevice
	}
	if c.Stack != nil {
		stack, err := c.Stack.Build()
		if err != nil {
			return nil, err
		}
		config.Stack = stack
	}
	return config, nil
}

type WireGuardStackConfig struct {
	MTU                   uint32                            `json:"mtu"`
	UserLevel             uint32                            `json:"userLevel"`
	IPs                   []string                          `json:"ips"`
	Routes                []string                          `json:"routes"`
	EnablePromiscuousMode bool                              `json:"enablePromiscuousMode"`
	EnableSpoofing        bool                              `json:"enableSpoofing"`
	SocketSettings        *SocketConfig                     `json:"socketSettings"`
	PreferIPv6ForUDP      bool                              `json:"preferIPv6ForUDP"`
	DualStackUDP          bool                              `json:"dualStackUDP"`
	TCPListener           []WireGuardStackTCPListenerConfig `json:"tcpListener"`
}

type WireGuardStackTCPListenerConfig struct {
	Port    uint32 `json:"port"`
	Address string `json:"address"`
	Tag     string `json:"tag"`
}

func (c *WireGuardStackConfig) Build() (*gvisorstack.Config, error) {
	config := &gvisorstack.Config{
		Mtu:                   c.MTU,
		UserLevel:             c.UserLevel,
		EnablePromiscuousMode: c.EnablePromiscuousMode,
		EnableSpoofing:        c.EnableSpoofing,
		PreferIpv6ForUdp:      c.PreferIPv6ForUDP,
		DualStackUdp:          c.DualStackUDP,
	}
	if c.IPs != nil {
		for _, ip := range c.IPs {
			parsedIP, err := rule.ParseIP(ip)
			if err != nil {
				return nil, err
			}
			config.Ips = append(config.Ips, parsedIP)
		}
	}
	if c.Routes != nil {
		for _, route := range c.Routes {
			parsedRoute, err := rule.ParseIP(route)
			if err != nil {
				return nil, err
			}
			config.Routes = append(config.Routes, parsedRoute)
		}
	}
	if c.SocketSettings != nil {
		socketSettings, err := c.SocketSettings.Build()
		if err != nil {
			return nil, err
		}
		config.SocketSettings = socketSettings
	}
	if c.TCPListener != nil {
		for _, tcpListener := range c.TCPListener {
			address, err := rule.ParseIP(tcpListener.Address)
			if err != nil {
				return nil, err
			}
			config.TcpListener = append(config.TcpListener, &gvisorstack.TCPListener{
				Port:    tcpListener.Port,
				Address: address,
				Tag:     tcpListener.Tag,
			})
		}
	}
	return config, nil
}

type WireGuardDeviceConfig struct {
	PrivateKey string                `json:"privateKey"`
	ListenPort uint32                `json:"listenPort"`
	Peers      []WireGuardPeerConfig `json:"peers"`
	MTU        uint32                `json:"mtu"`
}

func (c *WireGuardDeviceConfig) Build() (*wgcommon.DeviceConfig, error) {
	config := &wgcommon.DeviceConfig{
		ListenPort: c.ListenPort,
		Mtu:        c.MTU,
	}
	privateKey, err := base64.StdEncoding.DecodeString(c.PrivateKey)
	if err != nil {
		return nil, err
	}
	config.PrivateKey = privateKey
	if c.Peers != nil {
		config.Peers = make([]*wgcommon.PeerConfig, len(c.Peers))
		for i, peer := range c.Peers {
			p, err := peer.Build()
			if err != nil {
				return nil, err
			}
			config.Peers[i] = p
		}
	}
	return config, nil
}

type WireGuardPeerConfig struct {
	PublicKey                   string   `json:"publicKey"`
	PresharedKey                string   `json:"presharedKey"`
	AllowedIPs                  []string `json:"allowedIPs"`
	Endpoint                    string   `json:"endpoint"`
	PersistentKeepaliveInterval int64    `json:"persistentKeepaliveInterval"`
}

func (c *WireGuardPeerConfig) Build() (*wgcommon.PeerConfig, error) {
	config := &wgcommon.PeerConfig{
		AllowedIps:                  c.AllowedIPs,
		Endpoint:                    c.Endpoint,
		PersistentKeepaliveInterval: c.PersistentKeepaliveInterval,
	}
	publicKey, err := base64.StdEncoding.DecodeString(c.PublicKey)
	if err != nil {
		return nil, err
	}
	config.PublicKey = publicKey
	presharedKey, err := base64.StdEncoding.DecodeString(c.PresharedKey)
	if err != nil {
		return nil, err
	}
	config.PresharedKey = presharedKey
	return config, nil
}
