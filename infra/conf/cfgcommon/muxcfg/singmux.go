package muxcfg

import (
	"github.com/v2fly/v2ray-core/v5/app/proxyman"
)

type SingMuxConfig struct {
	Enabled        bool   `json:"enabled"`
	Protocol       string `json:"protocol"`
	MaxConnections uint32 `json:"maxConnections"`
	MinStreams     uint32 `json:"minStreams"`
	MaxStreams     uint32 `json:"maxStreams"`
	Padding        bool   `json:"padding"`
}

func (c *SingMuxConfig) Build() *proxyman.SingMultiplexConfig {
	return &proxyman.SingMultiplexConfig{
		Enabled:        c.Enabled,
		Protocol:       c.Protocol,
		MaxConnections: c.MaxConnections,
		MinStreams:     c.MinStreams,
		MaxStreams:     c.MaxStreams,
		Padding:        c.Padding,
	}
}
