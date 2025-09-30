//go:build !confonly

package httpupgrade

import (
	"github.com/v2fly/v2ray-core/v4/common"
	"github.com/v2fly/v2ray-core/v4/transport/internet"
)

//go:generate go run github.com/v2fly/v2ray-core/v4/common/errors/errorgen

const protocolName = "httpupgrade"

func init() {
	common.Must(internet.RegisterProtocolConfigCreator(protocolName, func() interface{} {
		return new(Config)
	}))
}
