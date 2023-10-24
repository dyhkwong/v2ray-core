//go:build !confonly
// +build !confonly

package device

import (
	"github.com/v2fly/v2ray-core/v4/common"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

type Device interface {
	stack.LinkEndpoint

	common.Closable
}

type Options struct {
	Name string
	MTU  uint32
}

type DeviceCreator func(Options) (Device, error)
