package tlscfg

import (
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls/utls"
)

type UTLSConfig struct {
	TLSConfig *TLSConfig `json:"tlsConfig"`
	Imitate   string     `json:"imitate"`
	NoSNI     bool       `json:"noSNI"`
	ForceALPN string     `json:"forceALPN"`
}

// Build implements Buildable.
func (c *UTLSConfig) Build() (proto.Message, error) {
	config := new(utls.Config)
	if c.TLSConfig != nil {
		tlsConfig, err := c.TLSConfig.Build()
		if err != nil {
			return nil, err
		}
		config.TlsConfig = tlsConfig.(*tls.Config)
	}
	imitate := c.Imitate
	if len(c.Imitate) > 0 {
		config.Imitate = imitate
	}
	config.NoSNI = c.NoSNI
	switch strings.ToLower(c.ForceALPN) {
	case "transportpreferencetakepriority", "transport_preference_take_priority":
		config.ForceAlpn = utls.ForcedALPN_TRANSPORT_PREFERENCE_TAKE_PRIORITY
	case "noalpn", "no_alpn":
		config.ForceAlpn = utls.ForcedALPN_NO_ALPN
	case "utlspreset", "utls_preset":
		config.ForceAlpn = utls.ForcedALPN_UTLS_PRESET
	default:
		config.ForceAlpn = utls.ForcedALPN_TRANSPORT_PREFERENCE_TAKE_PRIORITY
	}
	return config, nil
}
