/*

Some of codes are copied from https://github.com/octeep/wireproxy, license below.

Copyright (c) 2022 Wind T.F. Wong <octeep@pm.me>

Permission to use, copy, modify, and distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

*/

package wireguard

import (
	"context"
	"net/netip"
	"sync"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal"
	"github.com/v2fly/v2ray-core/v5/common/task"
	"github.com/v2fly/v2ray-core/v5/features/dns"
	"github.com/v2fly/v2ray-core/v5/features/policy"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

type Client struct {
	conf          *ClientConfig
	net           Tunnel
	bind          *netBindClient
	policyManager policy.Manager
	dns           dns.Client
	// cached configuration
	addresses        []netip.Addr
	hasIPv4, hasIPv6 bool
	wgLock           sync.Mutex
}

func NewClient(ctx context.Context, conf *ClientConfig) (*Client, error) {
	v := core.MustFromContext(ctx)

	addresses, hasIPv4, hasIPv6, err := parseEndpoints(conf.Address)
	if err != nil {
		return nil, err
	}

	d := v.GetFeature(dns.ClientType()).(dns.Client)
	return &Client{
		conf:          conf,
		policyManager: v.GetFeature(policy.ManagerType()).(policy.Manager),
		dns:           d,
		addresses:     addresses,
		hasIPv4:       hasIPv4,
		hasIPv6:       hasIPv6,
	}, nil
}

func (c *Client) InterfaceUpdate() {
	_ = c.Close()
}

func (c *Client) Close() error {
	c.wgLock.Lock()
	defer c.wgLock.Unlock()
	if c.net != nil {
		net := c.net
		go func() {
			_ = net.Close()
		}()
		c.net = nil
	}
	if c.bind != nil {
		_ = c.bind.Close()
		c.bind = nil
	}
	return nil
}

func (c *Client) processWireGuard(ctx context.Context, dialer internet.Dialer, resolver func(ctx context.Context, domain string) net.Address) error {
	c.wgLock.Lock()
	defer c.wgLock.Unlock()

	if c.bind != nil && c.net != nil {
		return nil
	}

	newError("switching dialer").AtInfo().WriteToLog()

	// bind := conn.NewStdNetBind() // TODO: conn.Bind wrapper for dialer
	c.bind = &netBindClient{
		netBind: netBind{
			workers:   int(c.conf.NumWorkers),
			readQueue: make(chan *netReadInfo),
			resolver:  resolver,
		},
		ctx:      core.ToBackgroundDetachedContext(ctx),
		dialer:   dialer,
		reserved: c.conf.Reserved,
	}

	net, err := c.makeVirtualTun()
	if err != nil {
		_ = c.bind.Close()
		c.bind = nil
		return newError("failed to create virtual tun interface").Base(err)
	}
	c.net = net
	return nil
}

// Process implements OutboundHandler.Dispatch().
func (c *Client) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified")
	}

	if err := c.processWireGuard(ctx, dialer, outbound.Resolver); err != nil {
		return err
	}

	// Destination of the inner request.
	dest := outbound.Target

	newError("tunneling request to ", dest).WriteToLog(session.ExportIDToError(ctx))

	originalDest := dest
	// resolve dns
	if dest.Address.Family().IsDomain() {
		ips, err := dns.LookupIPWithOption(c.dns, dest.Address.Domain(), dns.IPOption{
			IPv4Enable: c.hasIPv4 && c.conf.DomainStrategy != ClientConfig_USE_IP6,
			IPv6Enable: c.hasIPv6 && c.conf.DomainStrategy != ClientConfig_USE_IP4,
		})
		if err != nil {
			return newError("failed to lookup DNS").Base(err)
		} else if len(ips) == 0 {
			return dns.ErrEmptyResponse
		}
		if c.conf.DomainStrategy == ClientConfig_PREFER_IP4 || c.conf.DomainStrategy == ClientConfig_PREFER_IP6 {
			for _, ip := range ips {
				if ip.To4() != nil == (c.conf.DomainStrategy == ClientConfig_PREFER_IP4) {
					dest.Address = net.IPAddress(ip)
				}
			}
		} else {
			dest.Address = net.IPAddress(ips[0])
		}
	}

	p := c.policyManager.ForLevel(0)

	ctx, cancel := context.WithCancel(ctx)
	timer := signal.CancelAfterInactivity(ctx, cancel, p.Timeouts.ConnectionIdle)

	var requestFunc func() error
	var responseFunc func() error

	addrPort := netip.AddrPortFrom(toNetIpAddr(dest.Address), dest.Port.Value())
	if dest.Network == net.Network_TCP {
		conn, err := c.net.DialContextTCPAddrPort(ctx, addrPort)
		if err != nil {
			return newError("failed to create TCP connection").Base(err)
		}
		defer conn.Close()

		requestFunc = func() error {
			defer timer.SetTimeout(p.Timeouts.DownlinkOnly)
			return buf.Copy(link.Reader, buf.NewWriter(conn), buf.UpdateActivity(timer))
		}
		responseFunc = func() error {
			defer timer.SetTimeout(p.Timeouts.UplinkOnly)
			return buf.Copy(buf.NewReader(conn), link.Writer, buf.UpdateActivity(timer))
		}
	} else {
		conn, err := c.net.DialUDPAddrPort(netip.AddrPort{}, addrPort)
		if err != nil {
			return newError("failed to create UDP connection").Base(err)
		}
		defer conn.Close()

		ipToDomain := new(sync.Map)

		requestFunc = func() error {
			defer timer.SetTimeout(p.Timeouts.DownlinkOnly)
			// return buf.Copy(link.Reader, buf.NewWriter(conn), buf.UpdateActivity(timer))
			return buf.Copy(link.Reader, newPacketWriter(conn, originalDest, dest, c.dns, c.conf.DomainStrategy, c.hasIPv4, c.hasIPv6, ipToDomain), buf.UpdateActivity(timer))
		}
		responseFunc = func() error {
			defer timer.SetTimeout(p.Timeouts.UplinkOnly)
			// return buf.Copy(buf.NewReader(conn), link.Writer, buf.UpdateActivity(timer))
			return buf.Copy(newPacketReader(conn, ipToDomain), link.Writer, buf.UpdateActivity(timer))
		}
	}

	responseDonePost := task.OnSuccess(responseFunc, task.Close(link.Writer))
	if err := task.Run(ctx, requestFunc, responseDonePost); err != nil {
		common.Interrupt(link.Reader)
		common.Interrupt(link.Writer)
		return newError("connection ends").Base(err)
	}

	return nil
}

// creates a tun interface on netstack given a configuration
func (c *Client) makeVirtualTun() (Tunnel, error) {
	t, err := createTun(c.addresses, int(c.conf.Mtu), nil)
	if err != nil {
		return nil, err
	}

	if err = t.BuildDevice(createIPCRequest(c.conf.SecretKey, c.conf.Peers, false), c.bind); err != nil {
		_ = t.Close()
		return nil, err
	}
	return t, nil
}

func newPacketReader(conn net.Conn, ipToDomain *sync.Map) buf.Reader {
	if c, ok := conn.(net.PacketConn); ok {
		return &packetReader{
			packetConn: c,
			ipToDomain: ipToDomain,
		}
	}
	return buf.NewReader(conn)
}

type packetReader struct {
	packetConn net.PacketConn
	ipToDomain *sync.Map
}

func (r *packetReader) ReadMultiBuffer() (buf.MultiBuffer, error) {
	b := buf.New()
	b.Resize(0, buf.Size)
	n, src, err := r.packetConn.ReadFrom(b.Bytes())
	if err != nil {
		b.Release()
		return nil, err
	}
	b.Resize(0, int32(n))
	b.Endpoint = &net.Destination{
		Address: net.IPAddress(src.(*net.UDPAddr).IP),
		Port:    net.Port(src.(*net.UDPAddr).Port),
		Network: net.Network_UDP,
	}
	if domain, ok := r.ipToDomain.Load(src.(*net.UDPAddr).AddrPort().Addr()); ok {
		b.Endpoint.Address = domain.(net.Address)
	}
	return buf.MultiBuffer{b}, nil
}

func newPacketWriter(conn net.Conn, originalDest, dest net.Destination, dns dns.Client, domainStrategy ClientConfig_DomainStrategy, hasIPv4, hasIPv6 bool, ipToDomain *sync.Map) buf.Writer {
	if c, ok := conn.(net.PacketConn); ok {
		return &packetWriter{
			packetConn:     c,
			originalDest:   originalDest,
			dest:           dest,
			dns:            dns,
			domainStrategy: domainStrategy,
			hasIPv4:        hasIPv4,
			hasIPv6:        hasIPv6,
			ipToDomain:     ipToDomain,
		}
	}
	return buf.NewWriter(conn)
}

type packetWriter struct {
	packetConn     net.PacketConn
	originalDest   net.Destination
	dest           net.Destination
	dns            dns.Client
	domainStrategy ClientConfig_DomainStrategy
	hasIPv4        bool
	hasIPv6        bool
	ipToDomain     *sync.Map
}

func (w *packetWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	defer buf.ReleaseMulti(mb)
	for _, b := range mb {
		if b == nil {
			continue
		}
		originalDest, dest := w.originalDest, w.dest
		if b.Endpoint != nil {
			originalDest, dest = *b.Endpoint, *b.Endpoint
		}
		if dest.Address.Family().IsDomain() {
			if w.originalDest.Address.Family().IsDomain() && dest.Address.Domain() == w.originalDest.Address.Domain() {
				dest.Address = w.dest.Address
			} else {
				ips, err := dns.LookupIPWithOption(w.dns, dest.Address.Domain(), dns.IPOption{
					IPv4Enable: w.hasIPv4 && w.domainStrategy != ClientConfig_USE_IP6,
					IPv6Enable: w.hasIPv6 && w.domainStrategy != ClientConfig_USE_IP4,
				})
				if err != nil {
					return newError("failed to lookup DNS").Base(err)
				} else if len(ips) == 0 {
					return dns.ErrEmptyResponse
				}
				if w.domainStrategy == ClientConfig_PREFER_IP4 || w.domainStrategy == ClientConfig_PREFER_IP6 {
					for _, ip := range ips {
						if ip.To4() != nil == (w.domainStrategy == ClientConfig_PREFER_IP4) {
							dest.Address = net.IPAddress(ip)
						}
					}
				} else {
					dest.Address = net.IPAddress(ips[0])
				}
			}
		}
		udpAddr := &net.UDPAddr{
			IP:   dest.Address.IP(),
			Port: int(dest.Port),
		}
		if originalDest.Address.Family().IsDomain() {
			w.ipToDomain.LoadOrStore(udpAddr.AddrPort().Addr(), originalDest.Address)
		}
		_, err := w.packetConn.WriteTo(b.Bytes(), udpAddr)
		if err != nil {
			return err
		}
	}
	return nil
}

func init() {
	common.Must(common.RegisterConfig((*ClientConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewClient(ctx, config.(*ClientConfig))
	}))
}
