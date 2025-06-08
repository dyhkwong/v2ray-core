package v4

import (
	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon"
	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon/tlscfg"
	"github.com/v2fly/v2ray-core/v5/proxy/tuic"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

type TuicClientConfig struct {
	Address           *cfgcommon.Address `json:"address"`
	Port              uint16             `json:"port"`
	UUID              string             `json:"uuid"`
	Password          string             `json:"password"`
	CongestionControl string             `json:"congestionControl"`
	UDPRelayMode      string             `json:"udpRelayMode"`
	Heartbeat         uint32             `json:"heartbeat"`
	ZeroRTTHandshake  bool               `json:"zeroRTTHandshake"`
	DisableSni        bool               `json:"disableSNI"`
	TLSSettings       *tlscfg.TLSConfig  `json:"tlsSettings"`
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
		Heartbeat:         c.Heartbeat,
		ZeroRttHandshake:  c.ZeroRTTHandshake,
		DisableSni:        c.DisableSni,
	}
	if c.TLSSettings != nil {
		tlsSettings, err := c.TLSSettings.Build()
		if err != nil {
			return nil, err
		}
		config.TlsSettings = tlsSettings.(*tls.Config)
	}
	return config, nil
}
