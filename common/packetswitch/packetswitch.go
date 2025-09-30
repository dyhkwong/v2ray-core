//go:build !confonly

package packetswitch

import "github.com/v2fly/v2ray-core/v4/common"

type NetworkLayerDevice interface {
	common.Closable
	NetworkLayerPacketWriter
	NetworkLayerPacketReader
}

type NetworkLayerPacketWriter interface {
	Write(packet []byte) (n int, err error)
}

type NetworkLayerPacketReader interface {
	OnAttach(writer NetworkLayerPacketWriter) error
}
