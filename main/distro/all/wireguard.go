//go:build !dragonfly

package all

import (
	// WireGuard Outbound is unreleased.
	_ "github.com/v2fly/v2ray-core/v4/proxy/wireguard/outbound"
)
