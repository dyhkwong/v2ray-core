//go:build android && !cgo

package localdns

import (
	"net/netip"
)

// transport/internet/system_dns_android.go
const SystemDNS = "8.8.8.8"

func dnsReadConfig() []netip.Addr {
	return []netip.Addr{netip.MustParseAddr(SystemDNS)}
}
