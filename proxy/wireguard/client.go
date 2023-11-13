package wireguard

import (
	"context"
	"net/netip"
	"sync"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/log"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
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
	wgLock           *sync.Mutex
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
		wgLock:        &sync.Mutex{},
	}, nil
}

func (h *Client) processWireGuard(ctx context.Context, dialer internet.Dialer) (err error) {
	h.wgLock.Lock()
	defer h.wgLock.Unlock()

	if h.bind != nil && h.bind.dialer == dialer && h.net != nil {
		return nil
	}

	log.Record(&log.GeneralMessage{
		Severity: log.Severity_Info,
		Content:  "switching dialer",
	})

	if h.net != nil {
		_ = h.net.Close()
		h.net = nil
	}
	if h.bind != nil {
		_ = h.bind.Close()
		h.bind = nil
	}

	// bind := conn.NewStdNetBind() // TODO: conn.Bind wrapper for dialer
	h.bind = &netBindClient{
		netBind: netBind{
			dns: h.dns,
			dnsOption: dns.IPOption{
				IPv4Enable: h.hasIPv4,
				IPv6Enable: h.hasIPv6,
			},
			workers: int(h.conf.NumWorkers),
		},
		ctx:      core.ToBackgroundDetachedContext(ctx),
		dialer:   dialer,
		reserved: h.conf.Reserved,
	}
	defer func() {
		if err != nil {
			_ = h.bind.Close()
		}
	}()

	h.net, err = h.makeVirtualTun()
	if err != nil {
		return newError("failed to create virtual tun interface").Base(err)
	}
	return nil
}

// Process implements OutboundHandler.Dispatch().
func (h *Client) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified")
	}

	if err := h.processWireGuard(ctx, dialer); err != nil {
		return err
	}

	// Destination of the inner request.
	destination := outbound.Target
	command := protocol.RequestCommandTCP
	if destination.Network == net.Network_UDP {
		command = protocol.RequestCommandUDP
	}

	// resolve dns
	addr := destination.Address
	if addr.Family().IsDomain() {
		ips, err := dns.LookupIPWithOption(h.dns, addr.Domain(), dns.IPOption{
			IPv4Enable: h.hasIPv4 && h.conf.DomainStrategy != ClientConfig_USE_IP6,
			IPv6Enable: h.hasIPv6 && h.conf.DomainStrategy != ClientConfig_USE_IP4,
		})
		if err != nil {
			return newError("failed to lookup DNS").Base(err)
		} else if len(ips) == 0 {
			return dns.ErrEmptyResponse
		}
		if h.conf.DomainStrategy == ClientConfig_PREFER_IP4 || h.conf.DomainStrategy == ClientConfig_PREFER_IP6 {
			for _, ip := range ips {
				if ip.To4() != nil == (h.conf.DomainStrategy == ClientConfig_PREFER_IP4) {
					addr = net.IPAddress(ip)
				}
			}
		} else {
			addr = net.IPAddress(ips[0])
		}
	}

	p := h.policyManager.ForLevel(0)

	ctx, cancel := context.WithCancel(ctx)
	timer := signal.CancelAfterInactivity(ctx, cancel, p.Timeouts.ConnectionIdle)
	addrPort := netip.AddrPortFrom(toNetIpAddr(addr), destination.Port.Value())

	var requestFunc func() error
	var responseFunc func() error

	if command == protocol.RequestCommandTCP {
		conn, err := h.net.DialContextTCPAddrPort(ctx, addrPort)
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
	} else if command == protocol.RequestCommandUDP {
		conn, err := h.net.DialUDPAddrPort(netip.AddrPort{}, addrPort)
		if err != nil {
			return newError("failed to create UDP connection").Base(err)
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
func (h *Client) makeVirtualTun() (Tunnel, error) {
	t, err := createTun(h.endpoints, int(h.conf.Mtu))
	if err != nil {
		return nil, err
	}

	h.bind.dnsOption.IPv4Enable = h.hasIPv4
	h.bind.dnsOption.IPv6Enable = h.hasIPv6

	if err = t.BuildDevice(createIPCRequest(h.conf.SecretKey, h.conf.Peers), h.bind); err != nil {
		_ = t.Close()
		return nil, err
	}
	return t, nil
}

func init() {
	common.Must(common.RegisterConfig((*ClientConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewClient(ctx, config.(*ClientConfig))
	}))
}
