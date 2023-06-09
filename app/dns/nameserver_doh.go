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

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/net/cnc"
	"github.com/v2fly/v2ray-core/v5/common/protocol/dns"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal/pubsub"
	"github.com/v2fly/v2ray-core/v5/common/task"
	dns_feature "github.com/v2fly/v2ray-core/v5/features/dns"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

// DoHNameServer implemented DNS over HTTPS (RFC8484) Wire Format,
// which is compatible with traditional dns over udp(RFC1035),
// thus most of the DOH implementation is copied from udpns.go
type DoHNameServer struct {
	sync.RWMutex
	ips        map[string]record
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
			dispatcherCtx := context.Background()

			dest, err := net.ParseDestination(network + ":" + addr)
			if err != nil {
				return nil, err
			}

			dispatcherCtx = session.ContextWithContent(dispatcherCtx, session.ContentFromContext(ctx))
			dispatcherCtx = session.ContextWithInbound(dispatcherCtx, session.InboundFromContext(ctx))
			dispatcherCtx = core.WithContext(dispatcherCtx, core.FromContext(ctx))

			link, err := dispatcher.Dispatch(dispatcherCtx, dest)
			if err != nil {
				return nil, err
			}

			cc := common.ChainedClosable{}
			if cw, ok := link.Writer.(common.Closable); ok {
				cc = append(cc, cw)
			}
			if cr, ok := link.Reader.(common.Closable); ok {
				cc = append(cc, cr)
			}
			return cnc.NewConnection(
				cnc.ConnectionInputMulti(link.Writer),
				cnc.ConnectionOutputMulti(link.Reader),
				cnc.ConnectionOnClose(cc),
			), nil
		},
	}
	dispatchedClient := &http.Client{
		Transport: tr,
		Timeout:   180 * time.Second,
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
		ips:      make(map[string]record),
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

	if len(s.ips) == 0 {
		return newError("nothing to do. stopping...")
	}

	for domain, record := range s.ips {
		if record.A != nil && record.A.Expire.Before(now) {
			record.A = nil
		}
		if record.AAAA != nil && record.AAAA.Expire.Before(now) {
			record.AAAA = nil
		}

		if record.A == nil && record.AAAA == nil {
			newError(s.name, " cleanup ", domain).AtDebug().WriteToLog()
			delete(s.ips, domain)
		} else {
			s.ips[domain] = record
		}
	}

	if len(s.ips) == 0 {
		s.ips = make(map[string]record)
	}

	return nil
}

func (s *DoHNameServer) updateIP(req *dnsRequest, ipRec *IPRecord) {
	elapsed := time.Since(req.start)

	s.Lock()
	rec := s.ips[req.domain]
	updated := false

	switch req.reqType {
	case dnsmessage.TypeA:
		if isNewer(rec.A, ipRec) {
			rec.A = ipRec
			updated = true
		}
	case dnsmessage.TypeAAAA:
		addr := make([]net.Address, 0)
		for _, ip := range ipRec.IP {
			if len(ip.IP()) == net.IPv6len {
				addr = append(addr, ip)
			}
		}
		ipRec.IP = addr
		if isNewer(rec.AAAA, ipRec) {
			rec.AAAA = ipRec
			updated = true
		}
	}
	newError(s.name, " got answer: ", req.domain, " ", req.reqType, " -> ", ipRec.IP, " ", elapsed).AtInfo().WriteToLog()

	if updated {
		s.ips[req.domain] = rec
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

func (s *DoHNameServer) newReqID() uint16 {
	return 0
}

func (s *DoHNameServer) sendQuery(ctx context.Context, domain string, clientIP net.IP, option dns_feature.IPOption) {
	newError(s.name, " querying: ", domain).AtInfo().WriteToLog(session.ExportIDToError(ctx))

	reqs := buildReqMsgs(domain, option, s.newReqID, genEDNS0Options(clientIP))

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

func (s *DoHNameServer) findIPsForDomain(domain string, option dns_feature.IPOption) ([]net.IP, uint32, time.Time, error) {
	s.RLock()
	record, found := s.ips[domain]
	s.RUnlock()

	if !found {
		return nil, 0, time.Time{}, errRecordNotFound
	}

	var ips, a, aaaa []net.Address
	var ttl uint32
	var expireAt time.Time
	var err, lastErr error
	updated := false
	if option.IPv4Enable {
		a, ttl, expireAt, err = record.A.getIPsAndTTL()
		if ttl == 0 {
			record.A = nil
			updated = true
		}
		if err != nil {
			lastErr = err
		}
		ips = append(ips, a...)
	}

	if option.IPv6Enable {
		aaaa, ttl, expireAt, err = record.AAAA.getIPsAndTTL()
		if ttl == 0 {
			record.AAAA = nil
			updated = true
		}
		if err != nil {
			lastErr = err
		}
		ips = append(ips, aaaa...)
	}

	if updated {
		s.Lock()
		s.ips[domain] = record
		s.Unlock()
	}

	if len(ips) > 0 {
		ips, err := toNetIP(ips)
		return ips, ttl, expireAt, err
	}

	if lastErr != nil {
		return nil, ttl, expireAt, lastErr
	}

	return nil, ttl, expireAt, dns_feature.ErrEmptyResponse
}

// QueryIPWithTTL implements ServerWithTTL.
func (s *DoHNameServer) QueryIPWithTTL(ctx context.Context, domain string, clientIP net.IP, option dns_feature.IPOption, disableCache bool) ([]net.IP, uint32, time.Time, error) { // nolint: dupl
	fqdn := Fqdn(domain)

	if disableCache {
		newError("DNS cache is disabled. Querying IP for ", domain, " at ", s.name).AtDebug().WriteToLog()
	} else {
		ips, ttl, expireAt, err := s.findIPsForDomain(fqdn, option)
		if err != errRecordNotFound {
			newError(s.name, " cache HIT ", domain, " -> ", ips).Base(err).AtDebug().WriteToLog()
			return ips, ttl, expireAt, err
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
		ips, ttl, expireAt, err := s.findIPsForDomain(fqdn, option)
		if err != errRecordNotFound {
			return ips, ttl, expireAt, err
		}

		select {
		case <-ctx.Done():
			return nil, ttl, expireAt, ctx.Err()
		case <-done:
		}
	}
}

// QueryIP implements Server.
func (s *DoHNameServer) QueryIP(ctx context.Context, domain string, clientIP net.IP, option dns_feature.IPOption, disableCache bool) ([]net.IP, error) { // nolint: dupl
	ips, _, _, err := s.QueryIPWithTTL(ctx, domain, clientIP, option, disableCache)
	return ips, err
}
