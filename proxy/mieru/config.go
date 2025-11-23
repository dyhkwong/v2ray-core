package mieru

import (
	"bytes"
	"context"
	gonet "net"

	mieruclient "github.com/enfein/mieru/v3/apis/client"
	mierucommon "github.com/enfein/mieru/v3/apis/common"
	mierumodel "github.com/enfein/mieru/v3/apis/model"
	mierupb "github.com/enfein/mieru/v3/pkg/appctl/appctlpb"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/features/dns/localdns"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/udp"
)

var (
	_ mierucommon.Dialer       = (*dialerWrapper)(nil)
	_ mierucommon.PacketDialer = (*dialerWrapper)(nil)
	_ mierucommon.DNSResolver  = (*resolverWrapper)(nil)
	_ mierucommon.DNSResolver  = (*localResolverWrapper)(nil)
	_ net.PacketConn           = (*udpAssociateWrapper)(nil)
)

type udpAssociateWrapper struct {
	*mierucommon.UDPAssociateWrapper
}

func (c *udpAssociateWrapper) ReadFrom(p []byte) (int, net.Addr, error) {
	// mierucommon.UDPAssociateWrapper ReadFrom() does not support domain address
	// so read and parse raw data
	b := make([]byte, len(p)+256)
	n, _, err := c.PacketConn.ReadFrom(b)
	if err != nil {
		return 0, nil, err
	}
	b = b[:n]
	if n <= 6 {
		return 0, nil, newError("packet size ", n, "is too short to hold UDP associate header")
	}
	if b[0] != 0x00 || b[1] != 0x00 {
		return 0, nil, newError("invalid UDP header")
	}
	if b[2] != 0x00 {
		return 0, nil, newError("UDP fragment is not supported")
	}
	var dest mierumodel.NetAddrSpec
	r := bytes.NewReader(b[3:])
	if err = dest.ReadFromSocks5(r); err != nil {
		return 0, nil, err
	}
	var addr net.Addr
	if dest.FQDN != "" {
		addr = udp.NewMonoDestUDPAddr(net.DomainAddress(dest.FQDN), net.Port(dest.Port))
	} else {
		addr = &net.UDPAddr{
			IP:   dest.IP,
			Port: dest.Port,
		}
	}
	n, err = r.Read(p)
	return n, addr, err
}

func (c *udpAssociateWrapper) WriteTo(p []byte, addr net.Addr) (int, error) {
	var destAddr net.Addr
	switch addr := addr.(type) {
	case *net.UDPAddr:
		destAddr = addr
	case *udp.MonoDestUDPAddr:
		if addr.Address.Family().IsDomain() {
			destAddr = &mierumodel.NetAddrSpec{
				Net: "udp",
				AddrSpec: mierumodel.AddrSpec{
					FQDN: addr.Address.Domain(),
					Port: int(addr.Port),
				},
			}
		} else {
			destAddr = &net.UDPAddr{
				IP:   addr.Address.IP(),
				Port: int(addr.Port),
			}
		}
	default:
		dest, err := net.ParseDestination("udp:" + addr.String())
		if err != nil {
			return 0, err
		}
		if dest.Address.Family().IsDomain() {
			destAddr = &mierumodel.NetAddrSpec{
				Net: "udp",
				AddrSpec: mierumodel.AddrSpec{
					FQDN: dest.Address.Domain(),
					Port: int(dest.Port),
				},
			}
		} else {
			destAddr = &net.UDPAddr{
				IP:   dest.Address.IP(),
				Port: int(dest.Port),
			}
		}
	}
	return c.UDPAssociateWrapper.WriteTo(p, destAddr)
}

type dialerWrapper struct {
	dialer internet.Dialer
}

func (d *dialerWrapper) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	destination, err := net.ParseDestination(network + ":" + address)
	if err != nil {
		return nil, err
	}
	return d.dialer.Dial(ctx, destination)
}

func (d *dialerWrapper) ListenPacket(ctx context.Context, network, _, raddr string) (net.PacketConn, error) {
	destination, err := net.ParseDestination(network + ":" + raddr)
	if err != nil {
		return nil, err
	}
	conn, err := d.dialer.Dial(ctx, destination)
	if err != nil {
		return nil, err
	}
	return internet.NewConnWrapper(conn), nil
}

type resolverWrapper struct {
	resolver func(ctx context.Context, domain string) net.Address
}

func (r *resolverWrapper) LookupIP(ctx context.Context, _, host string) ([]net.IP, error) {
	if addr := r.resolver(ctx, host); addr != nil {
		return []net.IP{addr.IP()}, nil
	}
	return nil, newError("failed to resolve domain ", host)
}

type localResolverWrapper struct {
	resolver *localdns.Client
}

func (r *localResolverWrapper) LookupIP(_ context.Context, network, host string) ([]net.IP, error) {
	switch network {
	case "ip4":
		return r.resolver.LookupIPv4(host)
	case "ip6":
		return r.resolver.LookupIPv6(host)
	case "ip":
		return r.resolver.LookupIP(host)
	default:
		return nil, gonet.UnknownNetworkError(network)
	}
}

func buildMieruClientConfig(config *ClientConfig, dialer *dialerWrapper, resolver mierucommon.DNSResolver) (*mieruclient.ClientConfig, error) {
	var transportProtocol *mierupb.TransportProtocol
	switch config.Protocol {
	case "", "tcp":
		transportProtocol = mierupb.TransportProtocol_TCP.Enum()
	case "udp":
		transportProtocol = mierupb.TransportProtocol_UDP.Enum()
	default:
		return nil, newError("unknown protocol")
	}
	var multiplexingLevel *mierupb.MultiplexingLevel
	switch config.Multiplexing {
	case "", "default":
		multiplexingLevel = mierupb.MultiplexingLevel_MULTIPLEXING_DEFAULT.Enum()
	case "off":
		multiplexingLevel = mierupb.MultiplexingLevel_MULTIPLEXING_OFF.Enum()
	case "low":
		multiplexingLevel = mierupb.MultiplexingLevel_MULTIPLEXING_LOW.Enum()
	case "middle":
		multiplexingLevel = mierupb.MultiplexingLevel_MULTIPLEXING_MIDDLE.Enum()
	case "high":
		multiplexingLevel = mierupb.MultiplexingLevel_MULTIPLEXING_HIGH.Enum()
	default:
		return nil, newError("unknown multiplexing")
	}
	var handshakeMode *mierupb.HandshakeMode
	switch config.HandshakeMode {
	case "", "default":
		handshakeMode = mierupb.HandshakeMode_HANDSHAKE_DEFAULT.Enum()
	case "standard":
		handshakeMode = mierupb.HandshakeMode_HANDSHAKE_STANDARD.Enum()
	case "nowait":
		handshakeMode = mierupb.HandshakeMode_HANDSHAKE_NO_WAIT.Enum()
	default:
		return nil, newError("unknown handshakeMode")
	}
	serverEndpoint := &mierupb.ServerEndpoint{}
	if len(config.PortRange) == 0 {
		port := int32(config.Port)
		serverEndpoint.PortBindings = append(serverEndpoint.PortBindings, &mierupb.PortBinding{
			Port:     &port,
			Protocol: transportProtocol,
		})
	} else {
		for _, portRange := range config.PortRange {
			serverEndpoint.PortBindings = append(serverEndpoint.PortBindings, &mierupb.PortBinding{
				PortRange: &portRange,
				Protocol:  transportProtocol,
			})
		}
	}
	switch config.Address.AsAddress().Family() {
	case net.AddressFamilyDomain:
		domain := config.Address.AsAddress().Domain()
		serverEndpoint.DomainName = &domain
	case net.AddressFamilyIPv4, net.AddressFamilyIPv6:
		ip := config.Address.AsAddress().IP().String()
		serverEndpoint.IpAddress = &ip
	}
	profileName := "mieru"
	return &mieruclient.ClientConfig{
		Profile: &mierupb.ClientProfile{
			ProfileName: &profileName,
			User: &mierupb.User{
				Name:     &config.Username,
				Password: &config.Password,
			},
			Servers: []*mierupb.ServerEndpoint{
				serverEndpoint,
			},
			Multiplexing: &mierupb.MultiplexingConfig{
				Level: multiplexingLevel,
			},
			HandshakeMode: handshakeMode,
		},
		Dialer:       dialer,
		PacketDialer: dialer,
		Resolver:     resolver, // must use Resolver for UDP protocol
		DNSConfig: &mierucommon.ClientDNSConfig{
			BypassDialerDNS: true, // do not use Resolver for TCP protocol
		},
	}, nil
}
