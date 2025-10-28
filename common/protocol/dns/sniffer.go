package dns

import (
	"encoding/binary"
	_ "unsafe"

	"github.com/miekg/dns"
)

//go:linkname IsDomainName net.isDomainName
func IsDomainName(domain string) bool

var errNotDNS = newError("not dns")

type SniffHeader struct{}

func (s *SniffHeader) Protocol() string {
	return "dns"
}

func (s *SniffHeader) Domain() string {
	return ""
}

func SniffTCPDNS(b []byte) (*SniffHeader, error) {
	if len(b)-2 != int(binary.BigEndian.Uint16(b[:2])) {
		return nil, errNotDNS
	}
	return SniffDNS(b[2:])
}

func SniffDNS(b []byte) (*SniffHeader, error) {
	message := new(dns.Msg)
	if err := message.Unpack(b); err != nil || len(message.Answer) > 0 || message.Response {
		return nil, errNotDNS
	}
	return &SniffHeader{}, nil
}
