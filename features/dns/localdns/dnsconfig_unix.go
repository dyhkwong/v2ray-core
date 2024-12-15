//go:build !windows && !android

package localdns

import (
	"bufio"
	"net/netip"
	"os"
	"strings"
)

// See resolv.conf(5) on a Linux machine.
func dnsReadConfig() (ips []netip.Addr) {
	file, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return []netip.Addr{netip.MustParseAddr("127.0.0.1"), netip.MustParseAddr("::1")}
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 && (line[0] == ';' || line[0] == '#') {
			// comment.
			continue
		}
		f := strings.Fields(line)
		if len(f) < 1 {
			continue
		}
		if f[0] == "nameserver" { // add one name server
			if len(f) > 1 {
				// One more check: make sure server name is
				// just an IP address. Otherwise we need DNS
				// to look it up.
				if ip, err := netip.ParseAddr(f[1]); err == nil {
					ips = append(ips, ip)
				}
			}
		}
	}
	if len(ips) == 0 {
		return []netip.Addr{netip.MustParseAddr("127.0.0.1"), netip.MustParseAddr("::1")}
	}
	return ips
}
