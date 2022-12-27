//go:build dragonfly

package v4

import (
	"github.com/golang/protobuf/proto"
)

type WireGuardOutboundConfig struct{}

func (c *WireGuardOutboundConfig) Build() (proto.Message, error) { // nolint:staticcheck
	return nil, newError("wireguard unsupported")
}
