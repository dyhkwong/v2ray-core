//go:build !confonly && (!linux || (linux && !(amd64 || arm64)))

package gvisor

import "github.com/v2fly/v2ray-core/v4/app/tun/device"

func New(options device.Options) (device.Device, error) {
	return nil, newError("not supported").AtError()
}
