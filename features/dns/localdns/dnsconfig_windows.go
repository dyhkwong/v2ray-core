package localdns

import (
	"net/netip"
	"os"
	"strconv"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func dnsReadConfig() (ips []netip.Addr) {
	aas, err := adapterAddresses()
	if err != nil {
		return []netip.Addr{netip.MustParseAddr("127.0.0.1"), netip.MustParseAddr("::1")}
	}
	for _, aa := range aas {
		// Only take interfaces whose OperStatus is IfOperStatusUp(0x01) into DNS configs.
		if aa.OperStatus != windows.IfOperStatusUp {
			continue
		}
		// Only take interfaces which have at least one gateway
		if aa.FirstGatewayAddress == nil {
			continue
		}
		for dns := aa.FirstDnsServerAddress; dns != nil; dns = dns.Next {
			sa, err := dns.Address.Sockaddr.Sockaddr()
			if err != nil {
				continue
			}
			var ip netip.Addr
			switch sa := sa.(type) {
			case *syscall.SockaddrInet4:
				ip = netip.AddrFrom4(sa.Addr)
			case *syscall.SockaddrInet6:
				if sa.Addr[0] == 0xfe && sa.Addr[1] == 0xc0 {
					// fec0/10 IPv6 addresses are site local anycast DNS
					// addresses Microsoft sets by default if no other
					// IPv6 DNS address is set. Site local anycast is
					// deprecated since 2004, see
					// https://datatracker.ietf.org/doc/html/rfc3879
					continue
				}
				ip = netip.AddrFrom16(sa.Addr)
				if sa.ZoneId != 0 {
					ip = ip.WithZone(strconv.FormatInt(int64(sa.ZoneId), 10))
				}
			default:
				// Unexpected type.
				continue
			}
			ips = append(ips, ip)
		}
	}
	if len(ips) == 0 {
		return []netip.Addr{netip.MustParseAddr("127.0.0.1"), netip.MustParseAddr("::1")}
	}
	return ips
}

// adapterAddresses returns a list of IP adapter and address
// structures. The structure contains an IP adapter and flattened
// multiple IP addresses including unicast, anycast and multicast
// addresses.
func adapterAddresses() ([]*windows.IpAdapterAddresses, error) {
	var b []byte
	l := uint32(15000) // recommended initial size
	for {
		b = make([]byte, l)
		const flags = windows.GAA_FLAG_INCLUDE_PREFIX | windows.GAA_FLAG_INCLUDE_GATEWAYS
		err := windows.GetAdaptersAddresses(syscall.AF_UNSPEC, flags, 0, (*windows.IpAdapterAddresses)(unsafe.Pointer(&b[0])), &l)
		if err == nil {
			if l == 0 {
				return nil, nil
			}
			break
		}
		if err.(syscall.Errno) != syscall.ERROR_BUFFER_OVERFLOW {
			return nil, os.NewSyscallError("getadaptersaddresses", err)
		}
		if l <= uint32(len(b)) {
			return nil, os.NewSyscallError("getadaptersaddresses", err)
		}
	}
	var aas []*windows.IpAdapterAddresses
	for aa := (*windows.IpAdapterAddresses)(unsafe.Pointer(&b[0])); aa != nil; aa = aa.Next {
		aas = append(aas, aa)
	}
	return aas, nil
}
