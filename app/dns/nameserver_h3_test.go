package dns_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	. "github.com/v2fly/v2ray-core/v4/app/dns"
	"github.com/v2fly/v2ray-core/v4/common"
	"github.com/v2fly/v2ray-core/v4/common/net"
	dns_feature "github.com/v2fly/v2ray-core/v4/features/dns"
)

func TestH3LocalNameServer(t *testing.T) {
	url, err := url.Parse("h3+local://1.1.1.1/dns-query")
	common.Must(err)

	s := NewH3LocalNameServer(url)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	ips, err := s.QueryIP(ctx, "google.com", net.IP(nil), dns_feature.IPOption{
		IPv4Enable: true,
		IPv6Enable: true,
	}, false)
	cancel()
	common.Must(err)
	if len(ips) == 0 {
		t.Error("expect some ips, but got 0")
	}
}

func TestH3LocalNameServerWithCache(t *testing.T) {
	url, err := url.Parse("h3+local://1.1.1.1/dns-query")
	common.Must(err)

	s := NewH3LocalNameServer(url)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	ips, err := s.QueryIP(ctx, "google.com", net.IP(nil), dns_feature.IPOption{
		IPv4Enable: true,
		IPv6Enable: true,
	}, false)
	cancel()
	common.Must(err)
	if len(ips) == 0 {
		t.Error("expect some ips, but got 0")
	}

	ctx2, cancel := context.WithTimeout(context.Background(), time.Second*5)
	ips2, err := s.QueryIP(ctx2, "google.com", net.IP(nil), dns_feature.IPOption{
		IPv4Enable: true,
		IPv6Enable: true,
	}, false)
	cancel()
	common.Must(err)
	if r := cmp.Diff(ips2, ips); r != "" {
		t.Fatal(r)
	}
}
