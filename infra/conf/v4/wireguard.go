package v4

import (
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/proxy/wireguard"
)

type WireGuardPeerConfig struct {
	PublicKey    string   `json:"publicKey"`
	PreSharedKey string   `json:"preSharedKey"`
	Endpoint     string   `json:"endpoint"`
	KeepAlive    uint32   `json:"keepAlive"`
	AllowedIPs   []string `json:"allowedIPs,omitempty"`
}

func (c *WireGuardPeerConfig) Build() (proto.Message, error) {
	var err error
	config := new(wireguard.PeerConfig)
	config.PublicKey, err = parseWireGuardKey(c.PublicKey)
	if err != nil {
		return nil, err
	}
	if len(c.PreSharedKey) > 0 {
		config.PreSharedKey, err = parseWireGuardKey(c.PreSharedKey)
		if err != nil {
			return nil, err
		}
	}
	config.Endpoint = c.Endpoint
	// default 0
	config.KeepAlive = c.KeepAlive
	if len(c.AllowedIPs) == 0 {
		config.AllowedIps = []string{"0.0.0.0/0", "::0/0"}
	} else {
		config.AllowedIps = c.AllowedIPs
	}
	return config, nil
}

type WireGuardClientConfig struct {
	SecretKey      string                `json:"secretKey"`
	Address        []string              `json:"address"`
	Peers          []WireGuardPeerConfig `json:"peers"`
	MTU            int32                 `json:"mtu"`
	NumWorkers     int32                 `json:"workers"`
	Reserved       []byte                `json:"reserved"`
	DomainStrategy string                `json:"domainStrategy"`
}

func (c *WireGuardClientConfig) Build() (proto.Message, error) {
	config := new(wireguard.ClientConfig)
	var err error
	config.SecretKey, err = parseWireGuardKey(c.SecretKey)
	if err != nil {
		return nil, err
	}
	if len(c.Address) == 0 {
		return nil, newError("empty address")
	}
	config.Address = c.Address
	for _, peer := range c.Peers {
		msg, err := peer.Build()
		if err != nil {
			return nil, err
		}
		config.Peers = append(config.Peers, msg.(*wireguard.PeerConfig))
	}
	if c.MTU == 0 {
		config.Mtu = 1420
	} else {
		config.Mtu = c.MTU
	}
	// these a fallback code in wireguard-go code,
	// we don't need to process fallback manually
	config.NumWorkers = c.NumWorkers
	if len(c.Reserved) != 0 && len(c.Reserved) != 3 {
		return nil, newError(`"reserved" should be empty or 3 bytes`)
	}
	config.Reserved = c.Reserved
	switch strings.ToLower(c.DomainStrategy) {
	case "useip", "":
		config.DomainStrategy = wireguard.ClientConfig_USE_IP
	case "useipv4":
		config.DomainStrategy = wireguard.ClientConfig_USE_IP4
	case "useipv6":
		config.DomainStrategy = wireguard.ClientConfig_USE_IP6
	case "preferipv4":
		config.DomainStrategy = wireguard.ClientConfig_PREFER_IP4
	case "preferipv6":
		config.DomainStrategy = wireguard.ClientConfig_PREFER_IP6
	default:
		return nil, newError("unsupported domain strategy: ", c.DomainStrategy)
	}
	return config, nil
}

func parseWireGuardKey(key string) (string, error) {
	if key == "" {
		return "", newError("key must not be empty")
	}
	str, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", newError("invalid key")
	}
	return hex.EncodeToString(str), nil
}
