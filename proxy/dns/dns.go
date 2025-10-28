package dns

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/miekg/dns"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	dns_proto "github.com/v2fly/v2ray-core/v5/common/protocol/dns"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal"
	"github.com/v2fly/v2ray-core/v5/common/task"
	feature_dns "github.com/v2fly/v2ray-core/v5/features/dns"
	"github.com/v2fly/v2ray-core/v5/features/policy"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

func init() {
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		h := new(Handler)
		if err := core.RequireFeatures(ctx, func(dnsClient feature_dns.Client, policyManager policy.Manager) error {
			return h.Init(config.(*Config), dnsClient, policyManager)
		}); err != nil {
			return nil, err
		}
		return h, nil
	}))

	common.Must(common.RegisterConfig((*SimplifiedConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		simplifiedServer := config.(*SimplifiedConfig)
		_ = simplifiedServer
		fullConfig := &Config{}
		fullConfig.OverrideResponseTtl = simplifiedServer.OverrideResponseTtl
		fullConfig.ResponseTtl = simplifiedServer.ResponseTtl
		fullConfig.LookupAsExchange = simplifiedServer.LookupAsExchange
		return common.CreateObject(ctx, fullConfig)
	}))
}

type ownLinkVerifier interface {
	IsOwnLink(ctx context.Context) bool
}

type Handler struct {
	client          feature_dns.Client
	ipv4Lookup      feature_dns.IPv4Lookup
	ipv6Lookup      feature_dns.IPv6Lookup
	ownLinkVerifier ownLinkVerifier
	server          net.Destination
	timeout         time.Duration

	config *Config

	nonIPQuery string

	lookupAsExchange bool
}

func (h *Handler) Init(config *Config, dnsClient feature_dns.Client, policyManager policy.Manager) error {
	// Enable FakeDNS for DNS outbound
	if clientWithFakeDNS, ok := dnsClient.(feature_dns.ClientWithFakeDNS); ok {
		dnsClient = clientWithFakeDNS.AsFakeDNSClient()
	}
	h.client = dnsClient
	h.timeout = policyManager.ForLevel(config.UserLevel).Timeouts.ConnectionIdle

	if ipv4lookup, ok := dnsClient.(feature_dns.IPv4Lookup); ok {
		h.ipv4Lookup = ipv4lookup
	} else {
		return newError("dns.Client doesn't implement IPv4Lookup")
	}

	if ipv6lookup, ok := dnsClient.(feature_dns.IPv6Lookup); ok {
		h.ipv6Lookup = ipv6lookup
	} else {
		return newError("dns.Client doesn't implement IPv6Lookup")
	}

	if v, ok := dnsClient.(ownLinkVerifier); ok {
		h.ownLinkVerifier = v
	}

	if config.Server != nil {
		h.server = config.Server.AsDestination()
	}

	h.config = config

	h.nonIPQuery = config.Non_IPQuery

	h.lookupAsExchange = config.LookupAsExchange

	return nil
}

func (h *Handler) isOwnLink(ctx context.Context) bool {
	return h.ownLinkVerifier != nil && h.ownLinkVerifier.IsOwnLink(ctx)
}

func parseIPQuery(b []byte) (bool, string, uint16, uint16) {
	message := new(dns.Msg)
	if err := message.Unpack(b); err != nil {
		return false, "", 0, 0
	}
	id := message.Id
	if len(message.Question) != 1 || message.Response {
		return false, "", id, 0
	}
	qClass := message.Question[0].Qclass
	if qClass != dns.ClassINET {
		return false, "", id, 0
	}
	qType := message.Question[0].Qtype
	if qType != dns.TypeA && qType != dns.TypeAAAA {
		return false, "", id, qType
	}
	qName := message.Question[0].Name
	if dns_proto.IsDomainName(qName) {
		return true, qName, id, qType
	}
	return false, "", id, qType
}

// Process implements proxy.Outbound.
func (h *Handler) Process(ctx context.Context, link *transport.Link, d internet.Dialer) error {
	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("invalid outbound")
	}

	srcNetwork := outbound.Target.Network

	dest := outbound.Target
	if h.server.Network != net.Network_Unknown {
		dest.Network = h.server.Network
	}
	if h.server.Address != nil {
		dest.Address = h.server.Address
	}
	if h.server.Port != 0 {
		dest.Port = h.server.Port
	}

	newError("handling DNS traffic to ", dest).WriteToLog(session.ExportIDToError(ctx))

	conn := &outboundConn{
		dialer: func() (internet.Connection, error) {
			return d.Dial(ctx, dest)
		},
		connReady: make(chan struct{}, 1),
	}

	var reader dns_proto.MessageReader
	var writer dns_proto.MessageWriter
	if srcNetwork == net.Network_TCP {
		reader = &dns_proto.TCPReader{
			Reader: &buf.BufferedReader{
				Reader: link.Reader,
			},
		}
		writer = &dns_proto.TCPWriter{
			Writer: link.Writer,
		}
	} else {
		reader = &dns_proto.UDPReader{
			Reader: &buf.BufferedReader{
				Reader: link.Reader,
			},
		}
		writer = &dns_proto.UDPWriter{
			Writer: link.Writer,
		}
	}
	defer common.Close(reader)

	var connReader dns_proto.MessageReader
	var connWriter dns_proto.MessageWriter
	if dest.Network == net.Network_TCP {
		connReader = &dns_proto.TCPReader{
			Reader: conn,
		}
		connWriter = &dns_proto.TCPWriter{
			Writer: buf.NewWriter(conn),
		}
	} else {
		connReader = &dns_proto.UDPReader{
			Reader: conn,
		}
		connWriter = &dns_proto.UDPWriter{
			Writer: buf.NewWriter(conn),
		}
	}
	defer common.Close(connReader)

	ctx, cancel := context.WithCancel(ctx)
	timer := signal.CancelAfterInactivity(ctx, cancel, h.timeout)

	request := func() error {
		for {
			b, err := reader.ReadMessage()
			if err == io.EOF {
				return nil
			}

			if err != nil {
				return err
			}

			timer.Update()

			if !h.isOwnLink(ctx) {
				isIPQuery, domain, id, qType := parseIPQuery(b.Bytes())
				if isIPQuery || h.nonIPQuery != "drop" {
					if isIPQuery && !h.lookupAsExchange {
						go h.handleIPQuery(id, qType, domain, writer)
						b.Release()
						continue
					} else {
						go func() {
							h.handleRawQuery(b.Bytes(), writer)
							b.Release()
						}()
						continue
					}
				} else {
					h.handleDNSError(id, dns.RcodeNotImplemented, writer)
					b.Release()
				}
			} else if err := connWriter.WriteMessage(b); err != nil {
				return err
			}
		}
	}

	response := func() error {
		for {
			b, err := connReader.ReadMessage()
			if err == io.EOF {
				return nil
			}

			if err != nil {
				return err
			}

			timer.Update()

			if err := writer.WriteMessage(b); err != nil {
				return err
			}
		}
	}

	if err := task.Run(ctx, request, response); err != nil {
		return newError("connection ends").Base(err)
	}

	return nil
}

func (h *Handler) handleRawQuery(b []byte, writer dns_proto.MessageWriter) {
	if rawQuery, ok := h.client.(feature_dns.RawQuery); ok {
		resp, err := rawQuery.QueryRaw(b)
		if err != nil {
			newError(err).AtError().WriteToLog()
			return
		}
		if err := writer.WriteMessage(buf.FromBytes(resp)); err != nil {
			newError("write IP answer").Base(err).WriteToLog()
		}
	} else {
		newError("dns.RawQuery not implemented").AtError().WriteToLog()
	}
}

func (h *Handler) handleIPQuery(id uint16, qType uint16, domain string, writer dns_proto.MessageWriter) {
	var ips []net.IP
	var err error

	timeNow := time.Now()
	ttl := uint32(600)
	if h.config.OverrideResponseTtl {
		ttl = h.config.ResponseTtl
	}
	var expireAt time.Time

	switch qType {
	case dns.TypeA:
		if ipv4Lookup, ok := h.ipv4Lookup.(feature_dns.IPv4LookupWithTTL); ok {
			ips, expireAt, err = ipv4Lookup.LookupIPv4WithTTL(domain)
		} else {
			ips, err = h.ipv4Lookup.LookupIPv4(domain)
			expireAt = timeNow.Add(time.Duration(ttl) * time.Second)
		}
	case dns.TypeAAAA:
		if ipv6Lookup, ok := h.ipv6Lookup.(feature_dns.IPv6LookupWithTTL); ok {
			ips, expireAt, err = ipv6Lookup.LookupIPv6WithTTL(domain)
		} else {
			ips, err = h.ipv6Lookup.LookupIPv6(domain)
			expireAt = timeNow.Add(time.Duration(ttl) * time.Second)
		}
	}

	rcode := feature_dns.RCodeFromError(err)
	if rcode == 0 && len(ips) == 0 && err != feature_dns.ErrEmptyResponse {
		newError("ip query").Base(err).WriteToLog()
		return
	}

	message := new(dns.Msg)
	message.Compress = true
	message.Id = id
	message.Rcode = int(rcode)
	message.RecursionAvailable = true
	message.RecursionDesired = true
	message.Response = true
	message.Question = append(message.Question, dns.Question{Name: dns.Fqdn(domain), Qclass: dns.ClassINET, Qtype: qType})
	for _, ip := range ips {
		if len(ip) == net.IPv4len {
			message.Answer = append(message.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   dns.Fqdn(domain),
					Class:  dns.ClassINET,
					Rrtype: qType,
					Ttl:    max(uint32(expireAt.Sub(timeNow).Seconds()), 0),
				},
				A: ip,
			})
		} else {
			message.Answer = append(message.Answer, &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   dns.Fqdn(domain),
					Class:  dns.ClassINET,
					Rrtype: qType,
					Ttl:    max(uint32(expireAt.Sub(timeNow).Seconds()), 0),
				},
				AAAA: ip,
			})
		}
	}
	b := buf.New()
	rawBytes := b.Extend(buf.Size)
	msgBytes, err := message.PackBuffer(rawBytes)
	if err != nil {
		newError("pack message").Base(err).WriteToLog()
		b.Release()
		return
	}
	b.Resize(0, int32(len(msgBytes)))
	if err := writer.WriteMessage(b); err != nil {
		newError("write IP answer").Base(err).WriteToLog()
	}
}

func (h *Handler) handleDNSError(id uint16, rCode int, writer dns_proto.MessageWriter) {
	message := new(dns.Msg)
	message.Compress = true
	message.Id = id
	message.Rcode = rCode
	message.RecursionAvailable = true
	message.RecursionDesired = true
	message.Response = true
	b := buf.New()
	rawBytes := b.Extend(buf.Size)
	msgBytes, err := message.PackBuffer(rawBytes)
	if err != nil {
		newError("pack message").Base(err).WriteToLog()
		b.Release()
		return
	}
	b.Resize(0, int32(len(msgBytes)))
	if err := writer.WriteMessage(b); err != nil {
		newError("write IP answer").Base(err).WriteToLog()
	}
}

type outboundConn struct {
	access sync.Mutex
	dialer func() (internet.Connection, error)

	conn      net.Conn
	connReady chan struct{}
}

func (c *outboundConn) dial() error {
	conn, err := c.dialer()
	if err != nil {
		return err
	}
	c.conn = conn
	c.connReady <- struct{}{}
	return nil
}

func (c *outboundConn) Write(b []byte) (int, error) {
	c.access.Lock()

	if c.conn == nil {
		if err := c.dial(); err != nil {
			c.access.Unlock()
			newError("failed to dial outbound connection").Base(err).AtWarning().WriteToLog()
			return len(b), nil
		}
	}

	c.access.Unlock()

	return c.conn.Write(b)
}

func (c *outboundConn) Read(b []byte) (int, error) {
	var conn net.Conn
	c.access.Lock()
	conn = c.conn
	c.access.Unlock()

	if conn == nil {
		_, open := <-c.connReady
		if !open {
			return 0, io.EOF
		}
		conn = c.conn
	}

	return conn.Read(b)
}

func (c *outboundConn) Close() error {
	c.access.Lock()
	close(c.connReady)
	if c.conn != nil {
		c.conn.Close()
	}
	c.access.Unlock()
	return nil
}
