package v4

import (
	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon"
	"github.com/v2fly/v2ray-core/v5/proxy/tuic"
)

type TuicClientConfig struct {
	Address           *cfgcommon.Address `json:"address"`
	Port              uint16             `json:"port"`
	UUID              string             `json:"uuid"`
	Password          string             `json:"password"`
	CongestionControl string             `json:"congestionControl"`
	UDPRelayMode      string             `json:"udpRelayMode"`
	ZeroRTTHandshake  bool               `json:"zeroRTTHandshake"`
	ServerName        string             `json:"serverName"`
	ALPN              []string           `json:"alpn"`
	Certificate       []string           `json:"certificate"`
	AllowInsecure     bool               `json:"allowInsecure"`
	DisableSni        bool               `json:"disableSNI"`
}

func (c *TuicClientConfig) Build() (proto.Message, error) {
	if c.Address == nil {
		return nil, newError("missing server address")
	}
	config := &tuic.ClientConfig{
		Address:           c.Address.Build(),
		Port:              uint32(c.Port),
		Uuid:              c.UUID,
		Password:          c.Password,
		CongestionControl: c.CongestionControl,
		UdpRelayMode:      c.UDPRelayMode,
		ZeroRttHandshake:  c.ZeroRTTHandshake,
		ServerName:        c.ServerName,
		Alpn:              c.ALPN,
		Certificate:       c.Certificate,
		AllowInsecure:     c.AllowInsecure,
		DisableSni:        c.DisableSni,
	}
	return config, nil
}
