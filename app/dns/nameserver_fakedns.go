package dns

import (
	"context"
	"time"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/features/dns"
)

type FakeDNSServer struct {
	fakeDNSEngine dns.FakeDNSEngine
}

func NewFakeDNSServer(fakeDNSEngine dns.FakeDNSEngine) *FakeDNSServer {
	return &FakeDNSServer{fakeDNSEngine: fakeDNSEngine}
}

func (FakeDNSServer) Name() string {
	return "fakedns"
}

func (f *FakeDNSServer) QueryIPWithTTL(ctx context.Context, domain string, _ net.IP, opt dns.IPOption, _ bool) ([]net.IP, uint32, time.Time, error) {
	if !opt.FakeEnable {
		return nil, 0, time.Time{}, nil // Returning empty ip record with no error will continue DNS lookup, effectively indicating that this server is disabled.
	}
	if f.fakeDNSEngine == nil {
		if err := core.RequireFeatures(ctx, func(fd dns.FakeDNSEngine) {
			f.fakeDNSEngine = fd
		}); err != nil {
			return nil, 0, time.Time{}, newError("Unable to locate a fake DNS Engine").Base(err).AtError()
		}
	}
	var ips []net.Address
	if fkr0, ok := f.fakeDNSEngine.(dns.FakeDNSEngineRev0); ok {
		ips = fkr0.GetFakeIPForDomain3(domain, opt.IPv4Enable, opt.IPv6Enable)
	} else {
		ips = filterIP(f.fakeDNSEngine.GetFakeIPForDomain(domain), opt)
	}

	netIP, err := toNetIP(ips)
	if err != nil {
		return nil, 0, time.Time{}, newError("Unable to convert IP to net ip").Base(err).AtError()
	}

	newError(f.Name(), " got answer: ", domain, " -> ", ips).AtInfo().WriteToLog()

	ttl := uint32(1)
	expireAt := time.Now().Add(time.Duration(ttl) * time.Second)
	if len(netIP) > 0 {
		return netIP, ttl, expireAt, nil
	}
	return nil, ttl, expireAt, dns.ErrEmptyResponse
}

func (f *FakeDNSServer) QueryIP(ctx context.Context, domain string, _ net.IP, opt dns.IPOption, _ bool) ([]net.IP, error) {
	ips, _, _, err := f.QueryIPWithTTL(ctx, domain, nil, opt, false)
	return ips, err
}

func isFakeDNS(server Server) bool {
	_, ok := server.(*FakeDNSServer)
	return ok
}
