package muxcfg

import (
	"strings"

	"github.com/v2fly/v2ray-core/v5/app/proxyman"
	"github.com/v2fly/v2ray-core/v5/common/net/packetaddr"
)

type MuxConfig struct {
	Enabled        bool   `json:"enabled"`
	Concurrency    int16  `json:"concurrency"`
	PacketEncoding string `json:"packetEncoding"`
}

// Build creates MultiplexingConfig, Concurrency < 0 completely disables mux.
func (m *MuxConfig) Build() *proxyman.MultiplexingConfig {
	if m.Concurrency < 0 {
		return nil
	}

	var con uint32 = 8
	if m.Concurrency > 0 {
		con = uint32(m.Concurrency)
	}

	config := &proxyman.MultiplexingConfig{
		Enabled:     m.Enabled,
		Concurrency: con,
	}

	switch strings.ToLower(m.PacketEncoding) {
	case "packet":
		config.PacketEncoding = packetaddr.PacketAddrType_Packet
	case "xudp":
		config.PacketEncoding = packetaddr.PacketAddrType_XUDP
	case "", "none":
		config.PacketEncoding = packetaddr.PacketAddrType_None
	}

	return config
}
