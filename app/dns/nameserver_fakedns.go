package dns

import (
	"context"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
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

func (f *FakeDNSServer) QueryIPWithTTL(ctx context.Context, domain string, _ net.IP, opt dns.IPOption, _ bool) ([]net.IP, time.Time, error) {
	if !opt.FakeEnable {
		return nil, time.Time{}, nil // Returning empty ip record with no error will continue DNS lookup, effectively indicating that this server is disabled.
	}
	if f.fakeDNSEngine == nil {
		if err := core.RequireFeatures(ctx, func(fd dns.FakeDNSEngine) {
			f.fakeDNSEngine = fd
		}); err != nil {
			return nil, time.Time{}, newError("Unable to locate a fake DNS Engine").Base(err).AtError()
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
		return nil, time.Time{}, newError("Unable to convert IP to net ip").Base(err).AtError()
	}

	newError(f.Name(), " got answer: ", domain, " -> ", ips).AtInfo().WriteToLog()

	expireAt := time.Now().Add(time.Duration(1) * time.Second)
	if len(netIP) > 0 {
		return netIP, expireAt, nil
	}
	return nil, expireAt, dns.ErrEmptyResponse
}

func (f *FakeDNSServer) QueryIP(ctx context.Context, domain string, _ net.IP, opt dns.IPOption, _ bool) ([]net.IP, error) {
	ips, _, err := f.QueryIPWithTTL(ctx, domain, nil, opt, false)
	return ips, err
}

func (f *FakeDNSServer) NewReqID() uint16 {
	// placeholder
	return 0
}

func (f *FakeDNSServer) QueryRaw(ctx context.Context, request []byte) ([]byte, error) {
	requestMsg := new(dnsmessage.Message)
	err := requestMsg.Unpack(request)
	if err != nil {
		return nil, err
	}
	if len(requestMsg.Questions) != 1 {
		return nil, newError("failed to parse dns request: too many questions")
	}
	qName := requestMsg.Questions[0].Name
	qType := requestMsg.Questions[0].Type
	qClass := requestMsg.Questions[0].Class
	var opt dns.IPOption
	switch qType {
	case dnsmessage.TypeA:
		opt = dns.IPOption{IPv4Enable: true, FakeEnable: true}
	case dnsmessage.TypeAAAA:
		opt = dns.IPOption{IPv6Enable: true, FakeEnable: true}
	default:
		return nil, newError("failed to parse dns request: not A or AAAA")
	}
	ips, _, err := f.QueryIPWithTTL(ctx, strings.TrimSuffix(strings.ToLower(qName.String()), "."), nil, opt, false)
	if err != nil && err != dns.ErrEmptyResponse {
		return nil, err
	}
	builder := dnsmessage.NewBuilder(nil, dnsmessage.Header{
		ID:                 requestMsg.ID,
		RCode:              dnsmessage.RCodeSuccess,
		RecursionAvailable: true,
		RecursionDesired:   true,
		Response:           true,
	})
	builder.EnableCompression()
	common.Must(builder.StartQuestions())
	common.Must(builder.Question(dnsmessage.Question{Name: qName, Class: qClass, Type: qType}))
	common.Must(builder.StartAnswers())
	h := dnsmessage.ResourceHeader{Name: qName, Type: qType, Class: qClass, TTL: 1}
	for _, ip := range ips {
		switch qType {
		case dnsmessage.TypeA:
			common.Must(builder.AResource(h, dnsmessage.AResource{A: [4]byte(ip)}))
		case dnsmessage.TypeAAAA:
			common.Must(builder.AAAAResource(h, dnsmessage.AAAAResource{AAAA: [16]byte(ip)}))
		}
	}
	return builder.Finish()
}

func isFakeDNS(server Server) bool {
	_, ok := server.(*FakeDNSServer)
	return ok
}
