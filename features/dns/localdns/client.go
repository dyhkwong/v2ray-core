package localdns

import (
	"context"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/features/dns"
)

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

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

var rawQueryFunc = func(b []byte) ([]byte, error) {
	newError("localhost does not support raw query").AtError().WriteToLog()
	requestMsg := new(dnsmessage.Message)
	if err := requestMsg.Unpack(b); err != nil {
		return nil, newError("failed to parse dns request").Base(err)
	}
	responseMsg := new(dnsmessage.Message)
	responseMsg.ID = requestMsg.ID
	responseMsg.RCode = dnsmessage.RCodeNotImplemented
	responseMsg.RecursionAvailable = true
	responseMsg.RecursionDesired = true
	responseMsg.Response = true
	return responseMsg.Pack()
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

// QueryRaw implements RawQuery.
func (c *Client) QueryRaw(reqBytes []byte) ([]byte, error) {
	return rawQueryFunc(reqBytes)
}

// New create a new dns.Client that queries localhost for DNS.
func New() *Client {
	return &Client{}
}
