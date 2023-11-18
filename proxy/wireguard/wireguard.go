package wireguard

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/v2fly/v2ray-core/v5/common/net"
)

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

func toNetIpAddr(addr net.Address) netip.Addr {
	if addr.Family().IsIPv4() {
		return netip.AddrFrom4([4]byte(addr.IP()))
	} else {
		return netip.AddrFrom16([16]byte(addr.IP()))
	}
}

// convert endpoint string to netip.Addr
func parseEndpoints(ep []string) ([]netip.Addr, bool, bool, error) {
	var hasIPv4, hasIPv6 bool

	endpoints := make([]netip.Addr, len(ep))
	for i, str := range ep {
		var addr netip.Addr
		if strings.Contains(str, "/") {
			prefix, err := netip.ParsePrefix(str)
			if err != nil {
				return nil, false, false, err
			}
			addr = prefix.Addr()
		} else {
			var err error
			addr, err = netip.ParseAddr(str)
			if err != nil {
				return nil, false, false, err
			}
		}
		endpoints[i] = addr

		if addr.Is4() {
			hasIPv4 = true
		} else if addr.Is6() {
			hasIPv6 = true
		}
	}

	return endpoints, hasIPv4, hasIPv6, nil
}

// serialize the config into an IPC request
func createIPCRequest(secretKey string, peers []*PeerConfig, isServer bool) string {
	var request strings.Builder

	fmt.Fprintf(&request, "private_key=%s\n", secretKey)

	if isServer {
		// placeholder, we'll handle actual port listening on Xray
		request.WriteString("listen_port=1337\n")
	}

	for _, peer := range peers {
		if peer.PublicKey != "" {
			fmt.Fprintf(&request, "public_key=%s\n", peer.PublicKey)
		}

		if peer.PreSharedKey != "" {
			fmt.Fprintf(&request, "preshared_key=%s\n", peer.PreSharedKey)
		}

		if peer.Endpoint != "" {
			fmt.Fprintf(&request, "endpoint=%s\n", peer.Endpoint)
		}

		for _, ip := range peer.AllowedIps {
			fmt.Fprintf(&request, "allowed_ip=%s\n", ip)
		}

		if peer.KeepAlive != 0 {
			fmt.Fprintf(&request, "persistent_keepalive_interval=%d\n", peer.KeepAlive)
		}
	}

	return request.String()
}
