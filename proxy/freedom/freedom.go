package freedom

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

import (
	"context"
	"sync"
	"time"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/net/packetaddr"
	"github.com/v2fly/v2ray-core/v5/common/retry"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal"
	"github.com/v2fly/v2ray-core/v5/common/task"
	"github.com/v2fly/v2ray-core/v5/features/dns"
	"github.com/v2fly/v2ray-core/v5/features/dns/localdns"
	"github.com/v2fly/v2ray-core/v5/features/policy"
	"github.com/v2fly/v2ray-core/v5/features/stats"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

func init() {
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		h := new(Handler)
		if err := core.RequireFeatures(ctx, func(pm policy.Manager, d dns.Client) error {
			return h.Init(config.(*Config), pm, d)
		}); err != nil {
			return nil, err
		}
		return h, nil
	}))

	common.Must(common.RegisterConfig((*SimplifiedConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		simplifiedServer := config.(*SimplifiedConfig)
		_ = simplifiedServer
		fullConfig := &Config{
			DestinationOverride: simplifiedServer.DestinationOverride,
			ProtocolReplacement: simplifiedServer.ProtocolReplacement,
		}
		return common.CreateObject(ctx, fullConfig)
	}))
}

// Handler handles Freedom connections.
type Handler struct {
	policyManager policy.Manager
	dns           dns.Client
	config        *Config
}

// Init initializes the Handler with necessary parameters.
func (h *Handler) Init(config *Config, pm policy.Manager, d dns.Client) error {
	h.config = config
	h.policyManager = pm
	h.dns = d

	return nil
}

func (h *Handler) policy() policy.Session {
	p := h.policyManager.ForLevel(h.config.UserLevel)
	if h.config.Timeout > 0 && h.config.UserLevel == 0 {
		p.Timeouts.ConnectionIdle = time.Duration(h.config.Timeout) * time.Second
	}
	return p
}

func (h *Handler) resolveIP(ctx context.Context, domain string, localAddr net.Address) net.Address {
	ips, err := dns.LookupIPWithOption(h.dns, domain, dns.IPOption{
		IPv4Enable: h.config.DomainStrategy != Config_USE_IP6 || (localAddr != nil && localAddr.Family().IsIPv4()),
		IPv6Enable: h.config.DomainStrategy != Config_USE_IP4 || (localAddr != nil && localAddr.Family().IsIPv6()),
		FakeEnable: false,
	})
	if err != nil {
		newError("failed to get IP address for domain ", domain).Base(err).WriteToLog(session.ExportIDToError(ctx))
	}
	if len(ips) == 0 {
		return nil
	}
	if h.config.DomainStrategy == Config_PREFER_IP4 || h.config.DomainStrategy == Config_PREFER_IP6 {
		var addr net.Address
		for _, ip := range ips {
			addr = net.IPAddress(ip)
			if addr.Family().IsIPv4() == (h.config.DomainStrategy == Config_PREFER_IP4) {
				return addr
			}
		}
	}
	return net.IPAddress(ips[0])
}

func isValidAddress(addr *net.IPOrDomain) bool {
	if addr == nil {
		return false
	}

	a := addr.AsAddress()
	return a != net.AnyIP
}

// Process implements proxy.Outbound.
func (h *Handler) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified.")
	}
	destination := outbound.Target
	redirect := net.UDPDestination(nil, 0)
	if h.config.DestinationOverride != nil {
		server := h.config.DestinationOverride.Server
		if isValidAddress(server.Address) {
			destination.Address = server.Address.AsAddress()
			redirect.Address = destination.Address
		}
		if server.Port != 0 {
			destination.Port = net.Port(server.Port)
			redirect.Port = destination.Port
		}
	}
	if h.config.ProtocolReplacement != ProtocolReplacement_IDENTITY {
		if h.config.ProtocolReplacement == ProtocolReplacement_FORCE_TCP {
			destination.Network = net.Network_TCP
		}
		if h.config.ProtocolReplacement == ProtocolReplacement_FORCE_UDP {
			destination.Network = net.Network_UDP
		}
	}
	if h.config.useIP() {
		outbound.Resolver = func(ctx context.Context, domain string) net.Address {
			return h.resolveIP(ctx, domain, dialer.Address())
		}
	}
	newError("opening connection to ", destination).WriteToLog(session.ExportIDToError(ctx))

	input := link.Reader
	output := link.Writer

	var conn internet.Connection
	err := retry.ExponentialBackoff(5, 100).On(func() error {
		rawConn, err := dialer.Dial(ctx, destination)
		if err != nil {
			return err
		}
		conn = rawConn
		return nil
	})
	if err != nil {
		return newError("failed to open connection to ", destination).Base(err)
	}
	defer conn.Close()

	plcy := h.policy()
	ctx, cancel := context.WithCancel(ctx)
	timer := signal.CancelAfterInactivity(ctx, cancel, plcy.Timeouts.ConnectionIdle)

	isPacketAddr := false
	if _, err := packetaddr.ToPacketAddrConn(link, destination); err == nil {
		isPacketAddr = true
	}

	ipToDomain := new(sync.Map)

	requestDone := func() error {
		defer timer.SetTimeout(plcy.Timeouts.DownlinkOnly)

		var writer buf.Writer
		switch {
		case destination.Network == net.Network_TCP:
			writer = buf.NewWriter(conn)
		case redirect.Address != nil, redirect.Port != 0, isPacketAddr:
			writer = &buf.SequentialWriter{Writer: conn}
		default:
			writer = NewPacketWriter(ctx, h, conn, destination, ipToDomain)
		}

		if err := buf.Copy(input, writer, buf.UpdateActivity(timer)); err != nil {
			return newError("failed to process request").Base(err)
		}

		return nil
	}

	responseDone := func() error {
		defer timer.SetTimeout(plcy.Timeouts.UplinkOnly)

		var reader buf.Reader
		switch {
		case destination.Network == net.Network_TCP && h.config.ProtocolReplacement == ProtocolReplacement_IDENTITY:
			reader = buf.NewReader(conn)
		case redirect.Address != nil, redirect.Port != 0, isPacketAddr:
			reader = &buf.PacketReader{Reader: conn}
		default:
			reader = NewPacketReader(conn, ipToDomain)
		}
		if err := buf.Copy(reader, output, buf.UpdateActivity(timer)); err != nil {
			return newError("failed to process response").Base(err)
		}

		return nil
	}

	if err := task.Run(ctx, requestDone, task.OnSuccess(responseDone, task.Close(output))); err != nil {
		return newError("connection ends").Base(err)
	}

	return nil
}

func NewPacketReader(conn net.Conn, ipToDomain *sync.Map) buf.Reader {
	iConn := conn
	if trackedConn, ok := iConn.(*internet.TrackedConn); ok {
		iConn = trackedConn.NetConn()
	}
	statConn, ok := iConn.(*internet.StatCouterConnection)
	if ok {
		iConn = statConn.Connection
	}
	var counter stats.Counter
	if statConn != nil {
		counter = statConn.ReadCounter
	}
	if c, ok := iConn.(net.PacketConn); ok {
		return &PacketReader{
			packetConn: c,
			counter:    counter,
			ipToDomain: ipToDomain,
		}
	}
	return &buf.PacketReader{Reader: conn}
}

type PacketReader struct {
	packetConn net.PacketConn
	counter    stats.Counter
	ipToDomain *sync.Map
}

func (r *PacketReader) ReadMultiBuffer() (buf.MultiBuffer, error) {
	b := buf.New()
	b.Resize(0, buf.Size)
	n, d, err := r.packetConn.ReadFrom(b.Bytes())
	if err != nil {
		b.Release()
		return nil, err
	}
	b.Resize(0, int32(n))
	b.Endpoint = &net.Destination{
		Address: net.IPAddress(d.(*net.UDPAddr).IP),
		Port:    net.Port(d.(*net.UDPAddr).Port),
		Network: net.Network_UDP,
	}
	if domain, ok := r.ipToDomain.Load(d.(*net.UDPAddr).AddrPort().Addr()); ok {
		b.Endpoint.Address = domain.(net.Address)
	}
	if r.counter != nil {
		r.counter.Add(int64(n))
	}
	return buf.MultiBuffer{b}, nil
}

func NewPacketWriter(ctx context.Context, h *Handler, conn net.Conn, dest net.Destination, ipToDomain *sync.Map) buf.Writer {
	iConn := conn
	if trackedConn, ok := iConn.(*internet.TrackedConn); ok {
		iConn = trackedConn.NetConn()
	}
	statConn, ok := iConn.(*internet.StatCouterConnection)
	if ok {
		iConn = statConn.Connection
	}
	var counter stats.Counter
	if statConn != nil {
		counter = statConn.WriteCounter
	}
	if c, ok := iConn.(net.PacketConn); ok {
		return &PacketWriter{
			ctx:        ctx,
			handler:    h,
			packetConn: c,
			counter:    counter,
			dest:       dest,
			ipToDomain: ipToDomain,
		}
	}
	return &buf.SequentialWriter{Writer: conn}
}

type PacketWriter struct {
	ctx        context.Context
	handler    *Handler
	packetConn net.PacketConn
	counter    stats.Counter
	dest       net.Destination
	ipToDomain *sync.Map
}

func (w *PacketWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	defer buf.ReleaseMulti(mb)
	for _, b := range mb {
		if b == nil {
			continue
		}
		var dest net.Destination
		if b.Endpoint != nil {
			dest = *b.Endpoint
		} else {
			dest = w.dest
		}
		originalDest := dest
		if w.handler.config.useIP() && dest.Address.Family().IsDomain() {
			ip := w.handler.resolveIP(w.ctx, dest.Address.Domain(), nil)
			if ip != nil {
				dest.Address = ip
			}
		}
		if dest.Address.Family().IsDomain() {
			// SagerNet private
			ips, err := localdns.New().LookupIP(dest.Address.Domain())
			if err != nil {
				return err
			}
			if len(ips) == 0 {
				return dns.ErrEmptyResponse
			}
			dest.Address = net.IPAddress(ips[0])
		}
		destAddr, _ := net.ResolveUDPAddr("udp", dest.NetAddr())
		if destAddr == nil {
			continue
		}
		if originalDest.Address.Family().IsDomain() {
			w.ipToDomain.LoadOrStore(destAddr.AddrPort().Addr(), originalDest.Address)
		}
		n, err := w.packetConn.WriteTo(b.Bytes(), destAddr)
		if err != nil {
			return err
		}
		if w.counter != nil {
			w.counter.Add(int64(n))
		}
	}
	return nil
}
