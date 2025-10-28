//go:build !confonly

package dns

import (
	"context"
	"strings"
	"time"

	"github.com/miekg/dns"

	core "github.com/v2fly/v2ray-core/v4"
	"github.com/v2fly/v2ray-core/v4/common/net"
	feature_dns "github.com/v2fly/v2ray-core/v4/features/dns"
)

type FakeDNSServer struct {
	fakeDNSEngine feature_dns.FakeDNSEngine
}

func NewFakeDNSServer(fakeDNSEngine feature_dns.FakeDNSEngine) *FakeDNSServer {
	return &FakeDNSServer{fakeDNSEngine: fakeDNSEngine}
}

func (FakeDNSServer) Name() string {
	return "fakedns"
}

func (f *FakeDNSServer) QueryIPWithTTL(ctx context.Context, domain string, _ net.IP, opt feature_dns.IPOption, _ bool) ([]net.IP, time.Time, error) {
	if !opt.FakeEnable {
		return nil, time.Time{}, nil // Returning empty ip record with no error will continue DNS lookup, effectively indicating that this server is disabled.
	}
	if f.fakeDNSEngine == nil {
		if err := core.RequireFeatures(ctx, func(fd feature_dns.FakeDNSEngine) {
			f.fakeDNSEngine = fd
		}); err != nil {
			return nil, time.Time{}, newError("Unable to locate a fake DNS Engine").Base(err).AtError()
		}
	}
	var ips []net.Address
	if fkr0, ok := f.fakeDNSEngine.(feature_dns.FakeDNSEngineRev0); ok {
		ips = fkr0.GetFakeIPForDomain3(domain, opt.IPv4Enable, opt.IPv6Enable)
	} else {
		ips = filterIP(f.fakeDNSEngine.GetFakeIPForDomain(domain), opt)
	}

	netIP, err := toNetIP(ips)
	if err != nil {
		return nil, time.Time{}, newError("Unable to convert IP to net ip").Base(err).AtError()
	}

	newError(f.Name(), " got answer: ", domain, " -> ", ips).AtInfo().WriteToLog()

	expireAt := time.Now().Add(time.Duration(1) * time.Second)
	if len(netIP) > 0 {
		return netIP, expireAt, nil
	}
	return nil, expireAt, feature_dns.ErrEmptyResponse
}

func (f *FakeDNSServer) QueryIP(ctx context.Context, domain string, _ net.IP, opt feature_dns.IPOption, _ bool) ([]net.IP, error) {
	ips, _, err := f.QueryIPWithTTL(ctx, domain, nil, opt, false)
	return ips, err
}

func (f *FakeDNSServer) NewReqID() uint16 {
	// placeholder
	return 0
}

func (f *FakeDNSServer) QueryRaw(ctx context.Context, request []byte) ([]byte, error) {
	requestMsg := new(dns.Msg)
	err := requestMsg.Unpack(request)
	if err != nil {
		return nil, newError("failed to parse dns request").Base(err)
	}
	if requestMsg.Response || len(requestMsg.Answer) > 0 {
		newError("failed to parse dns request: not query").AtError().WriteToLog()
	}
	if len(requestMsg.Question) == 0 {
		return nil, newError("failed to parse dns request: no question present")
	}
	if len(requestMsg.Question) > 1 {
		return nil, newError("failed to parse dns request: too many questions")
	}
	qName := requestMsg.Question[0].Name
	qType := requestMsg.Question[0].Qtype
	qClass := requestMsg.Question[0].Qclass
	var opt feature_dns.IPOption
	switch qType {
	case dns.TypeA:
		opt = feature_dns.IPOption{IPv4Enable: true, FakeEnable: true}
	case dns.TypeAAAA:
		opt = feature_dns.IPOption{IPv6Enable: true, FakeEnable: true}
	default:
		return nil, newError("failed to parse dns request: not A or AAAA")
	}
	ips, _, err := f.QueryIPWithTTL(ctx, strings.TrimSuffix(qName, "."), nil, opt, false)
	if err != nil && err != feature_dns.ErrEmptyResponse {
		return nil, err
	}
	responseMsg := new(dns.Msg)
	responseMsg.Compress = true
	responseMsg.Id = requestMsg.Id
	responseMsg.Rcode = dns.RcodeSuccess
	responseMsg.RecursionAvailable = true
	responseMsg.RecursionDesired = true
	responseMsg.Response = true
	responseMsg.Question = append(responseMsg.Question, dns.Question{
		Name:   qName,
		Qclass: qClass,
		Qtype:  qType,
	})
	for _, ip := range ips {
		switch qType {
		case dns.TypeA:
			responseMsg.Answer = append(responseMsg.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   qName,
					Class:  qClass,
					Rrtype: qType,
					Ttl:    1,
				},
				A: ip,
			})
		case dns.TypeAAAA:
			responseMsg.Answer = append(responseMsg.Answer, &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   qName,
					Class:  qClass,
					Rrtype: qType,
					Ttl:    1,
				},
				AAAA: ip,
			})
		}
	}
	return responseMsg.Pack()
}

func isFakeDNS(server Server) bool {
	_, ok := server.(*FakeDNSServer)
	return ok
}
