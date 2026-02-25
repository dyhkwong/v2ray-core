package trusttunnel

import (
	"runtime"
)

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

var (
	defaultH2AppName    = "Go-http-client/2.0" // http2.Transport
	defaultH3AppName    = "quic-go HTTP/3"     // http3.Transport
	defaultH2UserAgent  = runtime.GOOS + " " + defaultH2AppName
	defaultH3UserAgent  = runtime.GOOS + " " + defaultH3AppName
	defaultUoTUserAgent = runtime.GOOS + " " + uotMagicAddress
)
