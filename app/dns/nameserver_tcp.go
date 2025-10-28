package dns

import (
	"context"
	"encoding/binary"
	"io"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
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

// TCPNameServer implemented DNS over TCP (RFC7766).
type TCPNameServer struct {
	sync.RWMutex
	name        string
	destination net.Destination
	ip4         map[string]*IPRecord
	ip6         map[string]*IPRecord
	pub         *pubsub.Service
	cleanup     *task.Periodic
	reqID       uint32
	dial        func(context.Context) (net.Conn, error)
	protocol    string
}

// NewTCPNameServer creates DNS over TCP server object for remote resolving.
func NewTCPNameServer(url *url.URL, dispatcher routing.Dispatcher) (*TCPNameServer, error) {
	s, err := baseTCPNameServer(url, "TCP", net.Port(53), "v2ray.dns")
	if err != nil {
		return nil, err
	}

	s.dial = func(ctx context.Context) (net.Conn, error) {
		link, err := dispatcher.Dispatch(ctx, s.destination)
		if err != nil {
			return nil, err
		}

		return cnc.NewConnection(
			cnc.ConnectionInputMulti(link.Writer),
			cnc.ConnectionOutputMulti(link.Reader),
		), nil
	}

	return s, nil
}

// NewTCPLocalNameServer creates DNS over TCP client object for local resolving
func NewTCPLocalNameServer(url *url.URL) (*TCPNameServer, error) {
	s, err := baseTCPNameServer(url, "TCPL", net.Port(53), "v2ray.dns")
	if err != nil {
		return nil, err
	}

	s.dial = func(ctx context.Context) (net.Conn, error) {
		return internet.DialSystem(ctx, s.destination, nil)
	}

	return s, nil
}

func baseTCPNameServer(url *url.URL, prefix string, port net.Port, protocol string) (*TCPNameServer, error) {
	var err error
	if url.Port() != "" {
		port, err = net.PortFromString(url.Port())
		if err != nil {
			return nil, err
		}
	}
	dest := net.TCPDestination(net.ParseAddress(url.Hostname()), port)

	s := &TCPNameServer{
		destination: dest,
		ip4:         make(map[string]*IPRecord),
		ip6:         make(map[string]*IPRecord),
		pub:         pubsub.NewService(),
		name:        prefix + "//" + dest.NetAddr(),
		protocol:    protocol,
	}
	s.cleanup = &task.Periodic{
		Interval: time.Minute,
		Execute:  s.Cleanup,
	}

	return s, nil
}

// Name implements Server.
func (s *TCPNameServer) Name() string {
	return s.name
}

// Cleanup clears expired items from cache
func (s *TCPNameServer) Cleanup() error {
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

func (s *TCPNameServer) updateIP(req *dnsRequest, ipRec *IPRecord) {
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

func (s *TCPNameServer) NewReqID() uint16 {
	return uint16(atomic.AddUint32(&s.reqID, 1))
}

func (s *TCPNameServer) sendQuery(ctx context.Context, domain string, clientIP net.IP, option dns_feature.IPOption) {
	newError(s.name, " querying DNS for: ", domain).AtDebug().WriteToLog(session.ExportIDToError(ctx))

	reqs := buildReqMsgs(domain, option, s.NewReqID, genEDNS0Options(clientIP))

	var deadline time.Time
	if d, ok := ctx.Deadline(); ok {
		deadline = d
	} else {
		deadline = time.Now().Add(time.Second * 5)
	}

	for _, req := range reqs {
		go func(r *dnsRequest) {
			dnsCtx := ctx

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

			dnsReqBuf := buf.NewWithSize(2 + b.Len())
			defer dnsReqBuf.Release()
			binary.Write(dnsReqBuf, binary.BigEndian, uint16(b.Len()))
			dnsReqBuf.Write(b.Bytes())
			b.Release()

			conn, err := s.dial(dnsCtx)
			if err != nil {
				newError("failed to dial namesever").Base(err).AtError().WriteToLog()
				return
			}
			defer conn.Close()

			_, err = conn.Write(dnsReqBuf.Bytes())
			if err != nil {
				newError("failed to send query").Base(err).AtError().WriteToLog()
				return
			}

			var length uint16
			err = binary.Read(conn, binary.BigEndian, &length)
			if err != nil {
				newError("failed to parse response length").Base(err).AtError().WriteToLog()
				return
			}
			respBuf := buf.NewWithSize(int32(length))
			defer respBuf.Release()
			_, err = respBuf.ReadFullFrom(conn, int32(length))
			if err != nil {
				newError("failed to read response length").Base(err).AtError().WriteToLog()
				return
			}

			rec, err := parseResponse(respBuf.Bytes())
			if err != nil {
				newError("failed to parse DNS over TCP response").Base(err).AtError().WriteToLog()
				return
			}

			s.updateIP(r, rec)
		}(req)
	}
}

func (s *TCPNameServer) QueryRaw(ctx context.Context, request []byte) ([]byte, error) {
	var deadline time.Time
	if d, ok := ctx.Deadline(); ok {
		deadline = d
	} else {
		deadline = time.Now().Add(time.Second * 5)
	}

	dnsCtx := ctx

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

	dnsReqBuf := buf.NewWithSize(2 + int32(len(request)))
	defer dnsReqBuf.Release()

	binary.Write(dnsReqBuf, binary.BigEndian, uint16(len(request)))
	dnsReqBuf.Write(request)

	conn, err := s.dial(dnsCtx)
	if err != nil {
		return nil, newError("failed to dial namesever").Base(err)
	}
	defer conn.Close()

	_, err = conn.Write(dnsReqBuf.Bytes())
	if err != nil {
		return nil, newError("failed to send query").Base(err)
	}

	var length uint16
	err = binary.Read(conn, binary.BigEndian, &length)
	if err != nil {
		return nil, newError("failed to parse response length").Base(err)
	}
	response := make([]byte, length)
	_, err = io.ReadFull(conn, response)
	if err != nil {
		return nil, newError("failed to read response length").Base(err)
	}
	return response, nil
}

func (s *TCPNameServer) findSingleStackIPsForDomain(domain string, isIPv6 bool) ([]net.IP, time.Time, error) {
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

func (s *TCPNameServer) findDualStackIPsForDomain(domain string) ([]net.IP, time.Time, error) {
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

func (s *TCPNameServer) findIPsForDomain(domain string, option dns_feature.IPOption) ([]net.IP, time.Time, error) {
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
func (s *TCPNameServer) QueryIPWithTTL(ctx context.Context, domain string, clientIP net.IP, option dns_feature.IPOption, disableCache bool) ([]net.IP, time.Time, error) {
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
func (s *TCPNameServer) QueryIP(ctx context.Context, domain string, clientIP net.IP, option dns_feature.IPOption, disableCache bool) ([]net.IP, error) {
	ips, _, err := s.QueryIPWithTTL(ctx, domain, clientIP, option, disableCache)
	return ips, err
}
