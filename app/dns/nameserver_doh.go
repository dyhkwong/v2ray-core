//go:build !confonly
// +build !confonly

package dns

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/v2fly/v2ray-core/v4/common"
	"github.com/v2fly/v2ray-core/v4/common/net"
	"github.com/v2fly/v2ray-core/v4/common/net/cnc"
	"github.com/v2fly/v2ray-core/v4/common/protocol/dns"
	"github.com/v2fly/v2ray-core/v4/common/session"
	"github.com/v2fly/v2ray-core/v4/common/signal/pubsub"
	"github.com/v2fly/v2ray-core/v4/common/task"
	dns_feature "github.com/v2fly/v2ray-core/v4/features/dns"
	"github.com/v2fly/v2ray-core/v4/features/routing"
	"github.com/v2fly/v2ray-core/v4/transport/internet"
)

// DoHNameServer implemented DNS over HTTPS (RFC8484) Wire Format,
// which is compatible with traditional dns over udp(RFC1035),
// thus most of the DOH implementation is copied from udpns.go
type DoHNameServer struct {
	sync.RWMutex
	ip4        map[string]*IPRecord
	ip6        map[string]*IPRecord
	pub        *pubsub.Service
	cleanup    *task.Periodic
	httpClient *http.Client
	dohURL     string
	name       string
	protocol   string
}

// NewDoHNameServer creates DOH server object for remote resolving.
func NewDoHNameServer(url *url.URL, dispatcher routing.Dispatcher) (*DoHNameServer, error) {
	newError("DNS: created Remote DOH client for ", url.String()).AtInfo().WriteToLog()
	s := baseDOHNameServer(url, "DOH", "tls")

	tr := &http.Transport{
		MaxIdleConns:        30,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 30 * time.Second,
		ForceAttemptHTTP2:   true,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dest, err := net.ParseDestination(network + ":" + addr)
			if err != nil {
				return nil, err
			}

			link, err := dispatcher.Dispatch(ctx, dest)
			if err != nil {
				return nil, err
			}
			return cnc.NewConnection(
				cnc.ConnectionInputMulti(link.Writer),
				cnc.ConnectionOutputMulti(link.Reader),
			), nil
		},
	}

	dispatchedClient := &http.Client{
		Transport: tr,
		Timeout:   60 * time.Second,
	}

	s.httpClient = dispatchedClient
	return s, nil
}

// NewDoHLocalNameServer creates DOH client object for local resolving
func NewDoHLocalNameServer(url *url.URL) *DoHNameServer {
	url.Scheme = "https"
	s := baseDOHNameServer(url, "DOHL", "tls")
	tr := &http.Transport{
		IdleConnTimeout:   90 * time.Second,
		ForceAttemptHTTP2: true,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dest, err := net.ParseDestination(network + ":" + addr)
			if err != nil {
				return nil, err
			}
			conn, err := internet.DialSystem(ctx, dest, nil)
			if err != nil {
				return nil, err
			}
			return conn, nil
		},
	}
	s.httpClient = &http.Client{
		Timeout:   time.Second * 180,
		Transport: tr,
	}
	newError("DNS: created Local DOH client for ", url.String()).AtInfo().WriteToLog()
	return s
}

func baseDOHNameServer(url *url.URL, prefix, protocol string) *DoHNameServer {
	s := &DoHNameServer{
		ip4:      make(map[string]*IPRecord),
		ip6:      make(map[string]*IPRecord),
		pub:      pubsub.NewService(),
		name:     prefix + "//" + url.Host,
		dohURL:   url.String(),
		protocol: protocol,
	}
	s.cleanup = &task.Periodic{
		Interval: time.Minute,
		Execute:  s.Cleanup,
	}
	return s
}

// Name implements Server.
func (s *DoHNameServer) Name() string {
	return s.name
}

// Cleanup clears expired items from cache
func (s *DoHNameServer) Cleanup() error {
	now := time.Now()
	s.Lock()
	defer s.Unlock()
	if len(s.ip4) == 0 && len(s.ip6) == 0 {
		return newError("nothing to do. stopping...")
	}
	for domain, record := range s.ip4 {
		if record.Expire.Before(now) {
			delete(s.ip4, domain)
			newError(s.name, " cleanup ", domain).AtDebug().WriteToLog()
		}
	}
	for domain, record := range s.ip6 {
		if record.Expire.Before(now) {
			delete(s.ip6, domain)
			newError(s.name, " cleanup ", domain).AtDebug().WriteToLog()
		}
	}
	if len(s.ip4) == 0 {
		s.ip4 = make(map[string]*IPRecord)
	}
	if len(s.ip6) == 0 {
		s.ip6 = make(map[string]*IPRecord)
	}
	return nil
}

func (s *DoHNameServer) updateIP(req *dnsRequest, ipRec *IPRecord) {
	var ipRecords map[string]*IPRecord
	if req.reqType == dnsmessage.TypeAAAA {
		ipRecords = s.ip6
	} else {
		ipRecords = s.ip4
	}
	elapsed := time.Since(req.start)
	s.Lock()
	rec := ipRecords[req.domain]
	if isNewer(rec, ipRec) {
		ipRecords[req.domain] = ipRec
		newError(s.name, " got answer: ", req.domain, " ", req.reqType, " -> ", ipRec.IP, " ", elapsed).AtInfo().WriteToLog()
	}
	switch req.reqType {
	case dnsmessage.TypeA:
		s.pub.Publish(req.domain+"4", nil)
	case dnsmessage.TypeAAAA:
		s.pub.Publish(req.domain+"6", nil)
	}
	s.Unlock()
	common.Must(s.cleanup.Start())
}

func (s *DoHNameServer) NewReqID() uint16 {
	return 0
}

func (s *DoHNameServer) sendQuery(ctx context.Context, domain string, clientIP net.IP, option dns_feature.IPOption) {
	newError(s.name, " querying: ", domain).AtInfo().WriteToLog(session.ExportIDToError(ctx))

	reqs := buildReqMsgs(domain, option, s.NewReqID, genEDNS0Options(clientIP))

	var deadline time.Time
	if d, ok := ctx.Deadline(); ok {
		deadline = d
	} else {
		deadline = time.Now().Add(time.Second * 5)
	}

	for _, req := range reqs {
		go func(r *dnsRequest) {
			// generate new context for each req, using same context
			// may cause reqs all aborted if any one encounter an error
			dnsCtx := ctx

			// reserve internal dns server requested Inbound
			if inbound := session.InboundFromContext(ctx); inbound != nil {
				dnsCtx = session.ContextWithInbound(dnsCtx, inbound)
			}

			dnsCtx = session.ContextWithContent(dnsCtx, &session.Content{
				Protocol:       s.protocol,
				SkipDNSResolve: true,
			})

			var cancel context.CancelFunc
			dnsCtx, cancel = context.WithDeadline(dnsCtx, deadline)
			defer cancel()

			b, err := dns.PackMessage(r.msg)
			if err != nil {
				newError("failed to pack dns query").Base(err).AtError().WriteToLog()
				return
			}
			resp, err := s.dohHTTPSContext(dnsCtx, b.Bytes())
			b.Release()
			if err != nil {
				newError("failed to retrieve response").Base(err).AtError().WriteToLog()
				return
			}
			rec, err := parseResponse(resp)
			if err != nil {
				newError("failed to handle DOH response").Base(err).AtError().WriteToLog()
				return
			}
			s.updateIP(r, rec)
		}(req)
	}
}

func (s *DoHNameServer) QueryRaw(ctx context.Context, request []byte) ([]byte, error) {
	var deadline time.Time
	if d, ok := ctx.Deadline(); ok {
		deadline = d
	} else {
		deadline = time.Now().Add(time.Second * 5)
	}

	// generate new context for each req, using same context
	// may cause reqs all aborted if any one encounter an error
	dnsCtx := ctx

	// reserve internal dns server requested Inbound
	if inbound := session.InboundFromContext(ctx); inbound != nil {
		dnsCtx = session.ContextWithInbound(dnsCtx, inbound)
	}

	dnsCtx = session.ContextWithContent(dnsCtx, &session.Content{
		Protocol:       s.protocol,
		SkipDNSResolve: true,
	})

	var cancel context.CancelFunc
	dnsCtx, cancel = context.WithDeadline(dnsCtx, deadline)
	defer cancel()

	resp, err := s.dohHTTPSContext(dnsCtx, request)
	if err != nil {
		return nil, newError("failed to retrieve response").Base(err)
	}
	return resp, nil
}

func (s *DoHNameServer) dohHTTPSContext(ctx context.Context, b []byte) ([]byte, error) {
	body := bytes.NewBuffer(b)
	req, err := http.NewRequest("POST", s.dohURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/dns-message")
	req.Header.Add("Content-Type", "application/dns-message")

	resp, err := s.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body) // flush resp.Body so that the conn is reusable
		return nil, fmt.Errorf("DOH server returned code %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (s *DoHNameServer) findSingleStackIPsForDomain(domain string, isIPv6 bool) ([]net.IP, time.Time, error) {
	var ipRecords map[string]*IPRecord
	if isIPv6 {
		ipRecords = s.ip6
	} else {
		ipRecords = s.ip4
	}
	s.RLock()
	record, found := ipRecords[domain]
	s.RUnlock()
	if !found {
		return nil, time.Time{}, errRecordNotFound
	}
	ips, expireAt, err := record.getIPs()
	if record.RCode != dnsmessage.RCodeSuccess && record.RCode != dnsmessage.RCodeNameError || record.TTL == 0 {
		s.Lock()
		delete(ipRecords, domain)
		s.Unlock()
	}
	if err != nil {
		return nil, expireAt, err
	}
	if len(ips) > 0 {
		netIP, err := toNetIP(ips)
		return netIP, expireAt, err
	}
	return nil, expireAt, dns_feature.ErrEmptyResponse
}

func (s *DoHNameServer) findDualStackIPsForDomain(domain string) ([]net.IP, time.Time, error) {
	// WTF
	ip4, expireAt4, err4 := s.findSingleStackIPsForDomain(domain, false)
	ip6, expireAt6, err6 := s.findSingleStackIPsForDomain(domain, true)
	minExpireAt := expireAt4
	if expireAt4.After(expireAt6) {
		minExpireAt = expireAt6
	}
	if err4 == nil && err6 == nil {
		return append(ip4, ip6...), minExpireAt, nil
	}
	if err4 == nil && err6 != errRecordNotFound {
		return ip4, expireAt4, nil
	}
	if err4 != errRecordNotFound && err6 == nil {
		return ip6, expireAt6, nil
	}
	if err4 == dns_feature.ErrEmptyResponse && err6 == dns_feature.ErrEmptyResponse {
		return nil, minExpireAt, dns_feature.ErrEmptyResponse
	}
	if err4 == errRecordNotFound || err6 == errRecordNotFound {
		return nil, minExpireAt, errRecordNotFound
	}
	return nil, minExpireAt, err6
}

func (s *DoHNameServer) findIPsForDomain(domain string, option dns_feature.IPOption) ([]net.IP, time.Time, error) {
	if !option.IPv4Enable && !option.IPv6Enable {
		return nil, time.Time{}, dns_feature.ErrEmptyResponse
	}
	if option.IPv4Enable && !option.IPv6Enable {
		return s.findSingleStackIPsForDomain(domain, false)
	}
	if !option.IPv4Enable && option.IPv6Enable {
		return s.findSingleStackIPsForDomain(domain, true)
	}
	return s.findDualStackIPsForDomain(domain)
}

// QueryIPWithTTL implements ServerWithTTL.
func (s *DoHNameServer) QueryIPWithTTL(ctx context.Context, domain string, clientIP net.IP, option dns_feature.IPOption, disableCache bool) ([]net.IP, time.Time, error) { // nolint: dupl
	fqdn := Fqdn(domain)

	if disableCache {
		newError("DNS cache is disabled. Querying IP for ", domain, " at ", s.name).AtDebug().WriteToLog()
	} else {
		ips, expireAt, err := s.findIPsForDomain(fqdn, option)
		if err != errRecordNotFound {
			newError(s.name, " cache HIT ", domain, " -> ", ips).Base(err).AtDebug().WriteToLog()
			return ips, expireAt, err
		}
	}

	// ipv4 and ipv6 belong to different subscription groups
	var sub4, sub6 *pubsub.Subscriber
	if option.IPv4Enable {
		sub4 = s.pub.Subscribe(fqdn + "4")
		defer sub4.Close()
	}
	if option.IPv6Enable {
		sub6 = s.pub.Subscribe(fqdn + "6")
		defer sub6.Close()
	}
	done := make(chan interface{})
	go func() {
		if sub4 != nil {
			select {
			case <-sub4.Wait():
			case <-ctx.Done():
			}
		}
		if sub6 != nil {
			select {
			case <-sub6.Wait():
			case <-ctx.Done():
			}
		}
		close(done)
	}()
	s.sendQuery(ctx, fqdn, clientIP, option)

	for {
		ips, expireAt, err := s.findIPsForDomain(fqdn, option)
		if err != errRecordNotFound {
			return ips, expireAt, err
		}

		select {
		case <-ctx.Done():
			return nil, time.Time{}, ctx.Err()
		case <-done:
		}
	}
}

// QueryIP implements Server.
func (s *DoHNameServer) QueryIP(ctx context.Context, domain string, clientIP net.IP, option dns_feature.IPOption, disableCache bool) ([]net.IP, error) { // nolint: dupl
	ips, _, err := s.QueryIPWithTTL(ctx, domain, clientIP, option, disableCache)
	return ips, err
}
