//go:build dragonfly

package conf

import (
	"github.com/golang/protobuf/proto"
)

type WireGuardOutboundConfig struct{}

func (c *WireGuardOutboundConfig) Build() (proto.Message, error) { // nolint:staticcheck
	return nil, newError("wireguard unsupported")
}
