package localdns

import (
	"context"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/features/dns"
)

var lookupFunc = func(network, host string) ([]net.IP, error) {
	resolver := &net.Resolver{PreferGo: false}
	ips, err := resolver.LookupIP(context.Background(), network, host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, dns.ErrEmptyResponse
	}
	return ips, nil
}

// Client is an implementation of dns.Client, which queries localhost for DNS.
type Client struct{}

// Type implements common.HasType.
func (*Client) Type() interface{} {
	return dns.ClientType()
}

// Start implements common.Runnable.
func (*Client) Start() error { return nil }

// Close implements common.Closable.
func (*Client) Close() error { return nil }

// LookupIP implements Client.
func (*Client) LookupIP(host string) ([]net.IP, error) {
	return lookupFunc("ip", host)
}

// LookupIPv4 implements IPv4Lookup.
func (*Client) LookupIPv4(host string) ([]net.IP, error) {
	return lookupFunc("ip4", host)
}

// LookupIPv6 implements IPv6Lookup.
func (*Client) LookupIPv6(host string) ([]net.IP, error) {
	return lookupFunc("ip6", host)
}

// New create a new dns.Client that queries localhost for DNS.
func New() *Client {
	return &Client{}
}
