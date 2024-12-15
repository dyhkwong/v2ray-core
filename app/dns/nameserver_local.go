package dns

import (
	"context"
	"math/rand"
	"time"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/features/dns"
	"github.com/v2fly/v2ray-core/v5/features/dns/localdns"
)

// LocalNameServer is an wrapper over local DNS feature.
type LocalNameServer struct {
	client *localdns.Client
}

// QueryIPWithTTL implements ServerWithTTL.
func (s *LocalNameServer) QueryIPWithTTL(ctx context.Context, domain string, _ net.IP, option dns.IPOption, _ bool) ([]net.IP, uint32, time.Time, error) {
	newError("localhost querying: ", domain).AtInfo().WriteToLog(session.ExportIDToError(ctx))
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
		newError("localhost got answer: ", domain, " -> ", ips).AtInfo().WriteToLog()
	}

	return ips, ttl, expireAt, err
}

// QueryIP implements Server.
func (s *LocalNameServer) QueryIP(ctx context.Context, domain string, _ net.IP, option dns.IPOption, _ bool) ([]net.IP, error) {
	ips, _, _, err := s.QueryIPWithTTL(ctx, domain, nil, option, false)
	return ips, err
}

// NewReqID implements ServerRaw.
func (s *LocalNameServer) NewReqID() uint16 {
	return uint16(rand.Intn(65536))
}

// QueryRaw implements ServerRaw.
func (s *LocalNameServer) QueryRaw(ctx context.Context, request []byte) ([]byte, error) {
	return s.client.QueryRaw(request)
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
