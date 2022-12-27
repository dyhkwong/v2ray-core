//go:build !confonly
// +build !confonly

package assembly

import (
	"github.com/v2fly/v2ray-core/v4/common"
	"github.com/v2fly/v2ray-core/v4/transport/internet"
)

//go:generate go run github.com/v2fly/v2ray-core/v4/common/errors/errorgen

const protocolName = "request"

func init() {
	common.Must(internet.RegisterProtocolConfigCreator(protocolName, func() interface{} {
		return new(Config)
	}))
}
