package dns

import (
	"bytes"
	"context"
	"encoding/binary"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/app/dispatcher"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/protocol/dns"
	udp_proto "github.com/v2fly/v2ray-core/v5/common/protocol/udp"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal/pubsub"
	"github.com/v2fly/v2ray-core/v5/common/task"
	dns_feature "github.com/v2fly/v2ray-core/v5/features/dns"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/transport/internet/udp"
)

// ClassicNameServer implemented traditional UDP DNS.
type ClassicNameServer struct {
	sync.RWMutex
	name      string
	address   net.Destination
	ips       map[string]record
	requests  map[uint16]dnsRequest
	pub       *pubsub.Service
	udpServer udp.DispatcherI
	cleanup   *task.Periodic
	reqID     uint32
	tcpServer *TCPNameServer

	channel map[uint16]chan []byte
}

// NewUDPNameServer creates udp server object for remote resolving.
func NewUDPNameServer(u *url.URL, dispatcher routing.Dispatcher) (*ClassicNameServer, error) {
	s, err := baseUDPNameServer(u, "UDP", dispatcher)
	if err != nil {
		return nil, err
	}
	s.tcpServer, _ = NewTCPNameServer(&url.URL{
		Scheme: "tcp",
		Host:   u.Host,
	}, dispatcher)
	return s, nil
}

// NewUDPLocalNameServer creates udp server object for local resolving.
func NewUDPLocalNameServer(u *url.URL) (*ClassicNameServer, error) {
	s, err := baseUDPNameServer(u, "UDPL", dispatcher.SystemInstance)
	if err != nil {
		return nil, err
	}
	s.tcpServer, _ = NewTCPNameServer(&url.URL{
		Scheme: "tcp+local",
		Host:   u.Host,
	}, dispatcher.SystemInstance)
	return s, nil
}

func baseUDPNameServer(url *url.URL, prefix string, dispatcher routing.Dispatcher) (*ClassicNameServer, error) {
	var err error
	port := net.Port(53)
	if url.Port() != "" {
		port, err = net.PortFromString(url.Port())
		if err != nil {
			return nil, err
		}
	}
	dest := net.UDPDestination(net.ParseAddress(url.Hostname()), port)
	s := newClassicNameServer(dest, prefix+"//"+dest.NetAddr(), dispatcher)
	return s, nil
}

// NewClassicNameServer creates udp server object for remote resolving.
func NewClassicNameServer(address net.Destination, dispatcher routing.Dispatcher) *ClassicNameServer {
	// default to 53 if unspecific
	if address.Port == 0 {
		address.Port = net.Port(53)
	}
	newError("DNS: created UDP client initialized for ", address.NetAddr()).AtInfo().WriteToLog()
	s := newClassicNameServer(address, strings.ToUpper(address.String()), dispatcher)
	s.tcpServer, _ = NewTCPNameServer(&url.URL{
		Scheme: "tcp",
		Host:   address.NetAddr(),
	}, dispatcher)
	return s
}

func newClassicNameServer(address net.Destination, name string, dispatcher routing.Dispatcher) *ClassicNameServer {
	s := &ClassicNameServer{
		address:  address,
		ips:      make(map[string]record),
		requests: make(map[uint16]dnsRequest),
		pub:      pubsub.NewService(),
		name:     name,

		channel: make(map[uint16]chan []byte),
	}
	s.cleanup = &task.Periodic{
		Interval: time.Minute,
		Execute:  s.Cleanup,
	}
	s.udpServer = udp.NewSplitDispatcher(dispatcher, s.HandleResponse)
	return s
}

// Name implements Server.
func (s *ClassicNameServer) Name() string {
	return s.name
}

// Cleanup clears expired items from cache
func (s *ClassicNameServer) Cleanup() error {
	now := time.Now()
	s.Lock()
	defer s.Unlock()

	if len(s.ips) == 0 && len(s.requests) == 0 {
		return newError(s.name, " nothing to do. stopping...")
	}

	for domain, record := range s.ips {
		if record.A != nil && record.A.Expire.Before(now) {
			record.A = nil
		}
		if record.AAAA != nil && record.AAAA.Expire.Before(now) {
			record.AAAA = nil
		}

		if record.A == nil && record.AAAA == nil {
			delete(s.ips, domain)
		} else {
			s.ips[domain] = record
		}
	}

	if len(s.ips) == 0 {
		s.ips = make(map[string]record)
	}

	for id, req := range s.requests {
		if req.expire.Before(now) {
			delete(s.requests, id)
		}
	}

	if len(s.requests) == 0 {
		s.requests = make(map[uint16]dnsRequest)
	}

	return nil
}

// HandleResponse handles udp response packet from remote DNS server.
func (s *ClassicNameServer) HandleResponse(ctx context.Context, packet *udp_proto.Packet) {
	defer packet.Payload.Release()
	payload := packet.Payload.Bytes()
	if len(payload) < 2 {
		return
	}

	id := binary.BigEndian.Uint16(payload[:2])
	s.Lock()
	if ch, found := s.channel[id]; found {
		ch <- bytes.Clone(payload)
		s.Unlock()
		return
	}
	s.Unlock()

	ipRec, err := parseResponse(payload)
	if err != nil && err != errTruncated {
		newError(s.name, " fail to parse responded DNS udp").AtError().WriteToLog()
		return
	}

	s.Lock()
	req, ok := s.requests[id]
	if ok {
		// remove the pending request
		delete(s.requests, id)
	}
	s.Unlock()
	if !ok {
		newError(s.name, " cannot find the pending request").AtError().WriteToLog()
		return
	}

	if err == errTruncated {
		newError("truncated, retry over TCP").AtError().WriteToLog()
		b, packErr := dns.PackMessage(req.msg)
		if packErr != nil {
			newError(packErr).AtError().WriteToLog()
			return
		}
		defer b.Release()
		response, tcpErr := s.tcpServer.QueryRaw(context.WithoutCancel(ctx), b.Bytes())
		if tcpErr != nil {
			newError("failed to send DNS query over TCP").Base(tcpErr).AtError().WriteToLog()
			return
		}
		ipRec, err = parseResponse(response)
		if err != nil {
			newError(s.name, " fail to parse responded DNS tcp").AtError().WriteToLog()
			return
		}
		ipRec.ReqID = id
	}

	var rec record
	switch req.reqType {
	case dnsmessage.TypeA:
		rec.A = ipRec
	case dnsmessage.TypeAAAA:
		rec.AAAA = ipRec
	}

	elapsed := time.Since(req.start)
	newError(s.name, " got answer: ", req.domain, " ", req.reqType, " -> ", ipRec.IP, " ", elapsed).AtInfo().WriteToLog()
	if len(req.domain) > 0 && (rec.A != nil || rec.AAAA != nil) {
		s.updateIP(req.domain, rec)
	}
}

func (s *ClassicNameServer) updateIP(domain string, newRec record) {
	s.Lock()

	newError(s.name, " updating IP records for domain:", domain).AtDebug().WriteToLog()
	rec := s.ips[domain]

	updated := false
	if isNewer(rec.A, newRec.A) {
		rec.A = newRec.A
		updated = true
	}
	if isNewer(rec.AAAA, newRec.AAAA) {
		rec.AAAA = newRec.AAAA
		updated = true
	}

	if updated {
		s.ips[domain] = rec
	}
	if newRec.A != nil {
		s.pub.Publish(domain+"4", nil)
	}
	if newRec.AAAA != nil {
		s.pub.Publish(domain+"6", nil)
	}
	s.Unlock()
	common.Must(s.cleanup.Start())
}

func (s *ClassicNameServer) NewReqID() uint16 {
	return uint16(atomic.AddUint32(&s.reqID, 1))
}

func (s *ClassicNameServer) addPendingRequest(req *dnsRequest) {
	s.Lock()
	defer s.Unlock()

	id := req.msg.ID
	req.expire = time.Now().Add(time.Second * 8)
	s.requests[id] = *req
}

func (s *ClassicNameServer) sendQuery(ctx context.Context, domain string, clientIP net.IP, option dns_feature.IPOption) {
	newError(s.name, " querying DNS for: ", domain).AtDebug().WriteToLog(session.ExportIDToError(ctx))

	reqs := buildReqMsgs(domain, option, s.NewReqID, genEDNS0Options(clientIP))

	var deadline time.Time
	if d, ok := ctx.Deadline(); ok {
		deadline = d
	} else {
		deadline = time.Now().Add(time.Second * 5)
	}

	for _, req := range reqs {
		s.addPendingRequest(req)
		b, err := dns.PackMessage(req.msg)
		if err != nil {
			newError("failed to pack dns query").Base(err).AtError().WriteToLog()
			return
		}
		udpCtx := ctx
		if inbound := session.InboundFromContext(ctx); inbound != nil {
			udpCtx = session.ContextWithInbound(udpCtx, inbound)
		}
		udpCtx = session.ContextWithContent(udpCtx, &session.Content{
			Protocol:       "dns",
			SkipDNSResolve: true,
		})
		var cancel context.CancelFunc
		udpCtx, cancel = context.WithDeadline(udpCtx, deadline)
		defer cancel()
		s.udpServer.Dispatch(core.ToBackgroundDetachedContext(udpCtx), s.address, b)
	}
}

func (s *ClassicNameServer) QueryRaw(ctx context.Context, request []byte) ([]byte, error) {
	if len(request) < 2 {
		return nil, newError("too short")
	}
	id := binary.BigEndian.Uint16(request[:2])
	ch := make(chan []byte)
	s.Lock()
	s.channel[id] = ch
	s.Unlock()

	var deadline time.Time
	if d, ok := ctx.Deadline(); ok {
		deadline = d
	} else {
		deadline = time.Now().Add(time.Second * 5)
	}

	udpCtx := ctx
	if inbound := session.InboundFromContext(ctx); inbound != nil {
		udpCtx = session.ContextWithInbound(udpCtx, inbound)
	}
	udpCtx = session.ContextWithContent(udpCtx, &session.Content{
		Protocol:       "dns",
		SkipDNSResolve: true,
	})
	var cancel context.CancelFunc
	udpCtx, cancel = context.WithDeadline(udpCtx, deadline)
	defer cancel()
	s.udpServer.Dispatch(core.ToBackgroundDetachedContext(udpCtx), s.address, buf.FromBytes(request))

	select {
	case <-udpCtx.Done():
		s.Lock()
		delete(s.channel, id)
		s.Unlock()
		return nil, udpCtx.Err()
	case response := <-ch:
		s.Lock()
		delete(s.channel, id)
		s.Unlock()
		if len(response) >= 4 && response[3]&0x02 >= 0x02 {
			newError("truncated, retry over TCP").AtError().WriteToLog()
			return s.tcpServer.QueryRaw(context.WithoutCancel(ctx), request)
		}
		return response, nil
	}
}

func (s *ClassicNameServer) findIPsForDomain(domain string, option dns_feature.IPOption) ([]net.IP, time.Time, error) {
	s.RLock()
	record, found := s.ips[domain]
	s.RUnlock()

	if !found {
		return nil, time.Time{}, errRecordNotFound
	}

	var ips, a, aaaa []net.Address
	var expireAt time.Time
	var err, lastErr error
	updated := false
	if option.IPv4Enable {
		a, expireAt, err = record.A.getIPs()
		if record.A != nil && record.A.TTL == 0 {
			record.A = nil
			updated = true
		}
		if err != nil {
			lastErr = err
		}
		ips = append(ips, a...)
	}

	if option.IPv6Enable {
		aaaa, expireAt, err = record.AAAA.getIPs()
		if record.AAAA != nil && record.AAAA.TTL == 0 {
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
		return ips, expireAt, err
	}

	if lastErr != nil {
		return nil, expireAt, lastErr
	}

	return nil, expireAt, dns_feature.ErrEmptyResponse
}

// QueryIPWithTTL implements ServerWithTTL.
func (s *ClassicNameServer) QueryIPWithTTL(ctx context.Context, domain string, clientIP net.IP, option dns_feature.IPOption, disableCache bool) ([]net.IP, time.Time, error) {
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
		select {
		case <-ctx.Done():
			return nil, time.Time{}, ctx.Err()
		case <-done:
		}

		ips, expireAt, err := s.findIPsForDomain(fqdn, option)
		if err != errRecordNotFound {
			return ips, expireAt, err
		}
	}
}

// QueryIP implements Server.
func (s *ClassicNameServer) QueryIP(ctx context.Context, domain string, clientIP net.IP, option dns_feature.IPOption, disableCache bool) ([]net.IP, error) {
	ips, _, err := s.QueryIPWithTTL(ctx, domain, clientIP, option, disableCache)
	return ips, err
}
