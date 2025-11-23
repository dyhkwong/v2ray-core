package mieru

import (
	"context"
	gonet "net"

	mieruclient "github.com/enfein/mieru/v3/apis/client"
	mierucommon "github.com/enfein/mieru/v3/apis/common"
	mierumodel "github.com/enfein/mieru/v3/apis/model"
	mierupb "github.com/enfein/mieru/v3/pkg/appctl/appctlpb"
	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/common/network"
	"google.golang.org/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/features/dns/localdns"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

var (
	_ mierucommon.Dialer       = (*dialerWrapper)(nil)
	_ mierucommon.PacketDialer = (*dialerWrapper)(nil)
	_ mierucommon.DNSResolver  = (*resolverWrapper)(nil)
	_ mierucommon.DNSResolver  = (*localResolverWrapper)(nil)
	_ network.PacketConn       = (*udpAssociateWrapper)(nil)
	_ network.FrontHeadroom    = (*udpAssociateWrapper)(nil)
)

var addressSerializer = metadata.NewSerializer(
	metadata.AddressFamilyByte(0x01, metadata.AddressFamilyIPv4),
	metadata.AddressFamilyByte(0x04, metadata.AddressFamilyIPv6),
	metadata.AddressFamilyByte(0x03, metadata.AddressFamilyFqdn),
)

type udpAssociateWrapper struct {
	*mierucommon.UDPAssociateWrapper
}

func (c *udpAssociateWrapper) ReadFrom(_ []byte) (int, net.Addr, error) {
	panic("do not call ReadFrom, call ReadPacket instead")
}

func (c *udpAssociateWrapper) WriteTo(_ []byte, _ net.Addr) (int, error) {
	panic("do not call WriteTo, call WritePacket instead")
}

func (c *udpAssociateWrapper) FrontHeadroom() int {
	return metadata.MaxSocksaddrLength + 3
}

func (c *udpAssociateWrapper) ReadPacket(buffer *buf.Buffer) (metadata.Socksaddr, error) {
	// mierucommon.UDPAssociateWrapper ReadFrom() does not support domain address
	// so read and parse raw data
	_, _, err := buffer.ReadPacketFrom(c.UDPAssociateWrapper.PacketConn) // nolint: staticcheck
	if err != nil {
		return metadata.Socksaddr{}, err
	}
	b, err := buffer.ReadBytes(3)
	if err != nil {
		return metadata.Socksaddr{}, err
	}
	if b[0] != 0x00 || b[1] != 0x00 || b[2] != 0x00 {
		return metadata.Socksaddr{}, newError("invalid UDP header")
	}
	return addressSerializer.ReadAddrPort(buffer)
}

func (c *udpAssociateWrapper) WritePacket(buffer *buf.Buffer, destination metadata.Socksaddr) error {
	addr := &mierumodel.NetAddrSpec{
		Net: "udp",
		AddrSpec: mierumodel.AddrSpec{
			IP:   destination.Addr.AsSlice(),
			FQDN: destination.Fqdn,
			Port: int(destination.Port),
		},
	}
	_, err := c.UDPAssociateWrapper.WriteTo(buffer.Bytes(), addr)
	buffer.Release()
	return err
}

type dialerWrapper struct {
	dialer internet.Dialer
}

func (d *dialerWrapper) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	destination, err := net.ParseDestination(network + ":" + address)
	if err != nil {
		return nil, err
	}
	newError("dialing to ", destination).AtDebug().WriteToLog(session.ExportIDToError(ctx))
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
	newError("dialing to ", destination).AtDebug().WriteToLog(session.ExportIDToError(ctx))
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
		serverEndpoint.PortBindings = append(serverEndpoint.PortBindings, &mierupb.PortBinding{
			Port:     proto.Int32(int32(config.Port)),
			Protocol: transportProtocol,
		})
	} else {
		for _, portRange := range config.PortRange {
			serverEndpoint.PortBindings = append(serverEndpoint.PortBindings, &mierupb.PortBinding{
				PortRange: proto.String(portRange),
				Protocol:  transportProtocol,
			})
		}
	}
	switch config.Address.AsAddress().Family() {
	case net.AddressFamilyDomain:
		serverEndpoint.DomainName = proto.String(config.Address.AsAddress().Domain())
	case net.AddressFamilyIPv4, net.AddressFamilyIPv6:
		serverEndpoint.IpAddress = proto.String(config.Address.AsAddress().IP().String())
	}
	return &mieruclient.ClientConfig{
		Profile: &mierupb.ClientProfile{
			ProfileName: proto.String("mieru"),
			User: &mierupb.User{
				Name:     proto.String(config.Username),
				Password: proto.String(config.Password),
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
