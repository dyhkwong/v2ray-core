package reality

import (
	"net"

	"github.com/xtls/reality"

	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

func (c *Config) GetREALITYConfig() *reality.Config {
	var dialer net.Dialer
	config := &reality.Config{
		DialContext:            dialer.DialContext,
		Type:                   c.Type,
		Dest:                   c.Dest,
		Xver:                   byte(c.Xver),
		PrivateKey:             c.PrivateKey,
		NextProtos:             nil, // should be nil
		SessionTicketsDisabled: true,
	}
	config.ServerNames = make(map[string]bool)
	for _, serverName := range c.ServerNames {
		config.ServerNames[serverName] = true
	}
	config.ShortIds = make(map[[8]byte]bool)
	for _, shortId := range c.ShortIds {
		config.ShortIds[*(*[8]byte)(shortId)] = true
	}
	return config
}

func ConfigFromStreamSettings(settings *internet.MemoryStreamConfig) *Config {
	if settings == nil {
		return nil
	}
	config, ok := settings.SecuritySettings.(*Config)
	if !ok {
		return nil
	}
	return config
}
