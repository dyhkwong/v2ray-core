//go:build tun && linux && (amd64 || arm64)

package all

import (
	_ "github.com/v2fly/v2ray-core/v4/app/tun"
)
