//go:build !confonly
// +build !confonly

package device

import (
	"github.com/v2fly/v2ray-core/v4/common"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

//go:generate go run github.com/v2fly/v2ray-core/v4/common/errors/errorgen

type Device interface {
	stack.LinkEndpoint

	common.Closable
}

type Options struct {
	Name string
	MTU  uint32
}

type DeviceConstructor func(Options) (Device, error)
