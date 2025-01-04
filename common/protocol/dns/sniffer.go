package dns

import (
	"encoding/binary"
	_ "unsafe"

	"golang.org/x/net/dns/dnsmessage"
)

//go:linkname isDomainName net.isDomainName
func isDomainName(domain string) bool

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
	message := new(dnsmessage.Message)
	if err := message.Unpack(b); err != nil || len(message.Questions) == 0 {
		return nil, errNotDNS
	}
	for _, question := range message.Questions {
		if !isDomainName(question.Name.String()) {
			return nil, errNotDNS
		}
	}
	return &SniffHeader{}, nil
}
