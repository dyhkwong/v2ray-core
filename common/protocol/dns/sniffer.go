package dns

import (
	"encoding/binary"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/v2fly/v2ray-core/v5/common/errors"
)

var errNotDNS = errors.New("not dns")

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
	return &SniffHeader{}, nil
}
