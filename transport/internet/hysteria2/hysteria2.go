//go:build !confonly
// +build !confonly

package hysteria2

import (
	"github.com/v2fly/v2ray-core/v4/common"
	"github.com/v2fly/v2ray-core/v4/transport/internet"
)

//go:generate go run github.com/v2fly/v2ray-core/v4/common/errors/errorgen

const (
	protocolName = "hysteria2"
)

func init() {
	common.Must(internet.RegisterProtocolConfigCreator(protocolName, func() interface{} {
		return new(Config)
	}))
}
