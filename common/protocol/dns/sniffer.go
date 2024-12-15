package dns

import (
	"golang.org/x/net/dns/dnsmessage"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/errors"
)

var (
	errNotDNS    = errors.New("not dns")
	errNotWanted = errors.New("not wanted")
)

type SniffHeader struct {
}

func (s *SniffHeader) Protocol() string {
	return "dns"
}

func (s *SniffHeader) Domain() string {
	return ""
}

func SniffTCPDNS(b []byte) (*SniffHeader, error) {
	if len(b) < 2 {
		return nil, common.ErrNoClue
	}
	return SniffDNS(b[2:])
}

func SniffDNS(b []byte) (*SniffHeader, error) {
	var parser dnsmessage.Parser
	if common.Error2(parser.Start(b)) != nil {
		return nil, errNotDNS
	}
	_, err := parser.Question()
	if err != nil {
		return nil, errNotDNS
	}
	return &SniffHeader{}, nil
}
