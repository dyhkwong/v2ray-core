package dns

import (
	"context"
	"time"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/features/dns"
	"github.com/v2fly/v2ray-core/v5/features/dns/localdns"
)

// LocalNameServer is an wrapper over local DNS feature.
type LocalNameServer struct {
	client *localdns.Client
}

// QueryIPWithTTL implements ServerWithTTL.
func (s *LocalNameServer) QueryIPWithTTL(_ context.Context, domain string, _ net.IP, option dns.IPOption, _ bool) ([]net.IP, uint32, time.Time, error) {
	var ips []net.IP
	ttl := uint32(600)
	expireAt := time.Now().Add(time.Duration(ttl) * time.Second)
	var err error

	switch {
	case option.IPv4Enable && option.IPv6Enable:
		ips, err = s.client.LookupIP(domain)
	case option.IPv4Enable:
		ips, err = s.client.LookupIPv4(domain)
	case option.IPv6Enable:
		ips, err = s.client.LookupIPv6(domain)
	}

	if len(ips) > 0 {
		newError("Localhost got answer: ", domain, " -> ", ips).AtInfo().WriteToLog()
	}

	return ips, ttl, expireAt, err
}

// QueryIP implements Server.
func (s *LocalNameServer) QueryIP(_ context.Context, domain string, _ net.IP, option dns.IPOption, _ bool) ([]net.IP, error) {
	ips, _, _, err := s.QueryIPWithTTL(context.TODO(), domain, nil, option, false)
	return ips, err
}

// Name implements Server.
func (s *LocalNameServer) Name() string {
	return "localhost"
}

// NewLocalNameServer creates localdns server object for directly lookup in system DNS.
func NewLocalNameServer() *LocalNameServer {
	newError("DNS: created localhost client").AtInfo().WriteToLog()
	return &LocalNameServer{
		client: localdns.New(),
	}
}

// NewLocalDNSClient creates localdns client object for directly lookup in system DNS.
func NewLocalDNSClient() *Client {
	return &Client{server: NewLocalNameServer()}
}
