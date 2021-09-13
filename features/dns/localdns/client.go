package localdns

import (
	"context"
	"time"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/features/dns"
)

var (
	defaultLookupFunc = func(network, host string) ([]net.IP, error) {
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
	lookupFunc = defaultLookupFunc
)

// SagerNet private
func SetLookupFunc(fn func(network, host string) ([]net.IP, error)) {
	if fn == nil {
		lookupFunc = defaultLookupFunc
	} else {
		lookupFunc = fn
	}
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

// LookupIPv4WithTTL implements IPv4LookupWithTTL.
func (c *Client) LookupIPv4WithTTL(host string) ([]net.IP, uint32, time.Time, error) {
	ttl := uint32(600)
	expireAt := time.Now().Add(time.Duration(ttl) * time.Second)
	ips, err := c.LookupIPv4(host)
	return ips, ttl, expireAt, err
}

// LookupIPv6WithTTL implements IPv6LookupWithTTL.
func (c *Client) LookupIPv6WithTTL(host string) ([]net.IP, uint32, time.Time, error) {
	ttl := uint32(600)
	expireAt := time.Now().Add(time.Duration(ttl) * time.Second)
	ips, err := c.LookupIPv6(host)
	return ips, ttl, expireAt, err
}

// New create a new dns.Client that queries localhost for DNS.
func New() *Client {
	return &Client{}
}
