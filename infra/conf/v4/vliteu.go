package v4

import (
	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon"
	"github.com/v2fly/v2ray-core/v5/proxy/vlite/inbound"
	"github.com/v2fly/v2ray-core/v5/proxy/vlite/outbound"
)

type VLiteUDPInboundConfig struct {
	Password                    string `json:"password"`
	ScramblePacket              bool   `json:"scramble_packet"`
	EnableFEC                   bool   `json:"enable_fec"`
	EnableStabilization         bool   `json:"enable_stabilization"`
	EnableRenegotiation         bool   `json:"enable_renegotiation"`
	HandshakeMaskingPaddingSize uint32 `json:"handshake_masking_padding_size"`
}

func (c *VLiteUDPInboundConfig) Build() (proto.Message, error) {
	return &inbound.UDPProtocolConfig{
		Password:                    c.Password,
		ScramblePacket:              c.ScramblePacket,
		EnableFec:                   c.EnableFEC,
		EnableStabilization:         c.EnableStabilization,
		EnableRenegotiation:         c.EnableRenegotiation,
		HandshakeMaskingPaddingSize: c.HandshakeMaskingPaddingSize,
	}, nil
}

type VLiteUDPOutboundConfig struct {
	Address                     *cfgcommon.Address `json:"address"`
	Port                        uint16             `json:"port"`
	Password                    string             `json:"password"`
	ScramblePacket              bool               `json:"scramble_packet"`
	EnableFEC                   bool               `json:"enable_fec"`
	EnableStabilization         bool               `json:"enable_stabilization"`
	EnableRenegotiation         bool               `json:"enable_renegotiation"`
	HandshakeMaskingPaddingSize uint32             `json:"handshake_masking_padding_size"`
}

func (c *VLiteUDPOutboundConfig) Build() (proto.Message, error) {
	return &outbound.UDPProtocolConfig{
		Address:                     c.Address.Build(),
		Port:                        uint32(c.Port),
		Password:                    c.Password,
		ScramblePacket:              c.ScramblePacket,
		EnableFec:                   c.EnableFEC,
		EnableStabilization:         c.EnableStabilization,
		EnableRenegotiation:         c.EnableRenegotiation,
		HandshakeMaskingPaddingSize: c.HandshakeMaskingPaddingSize,
	}, nil
}
