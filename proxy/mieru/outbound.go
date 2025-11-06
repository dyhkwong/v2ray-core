package mieru

import (
	"context"
	"sync"

	mieruclient "github.com/enfein/mieru/v3/apis/client"
	mierucommon "github.com/enfein/mieru/v3/apis/common"
	mierumodel "github.com/enfein/mieru/v3/apis/model"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/task"
	"github.com/v2fly/v2ray-core/v5/features/dns/localdns"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/udp"
)

func init() {
	common.Must(common.RegisterConfig((*ClientConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewClient(ctx, config.(*ClientConfig))
	}))
}

type Outbound struct {
	sync.Mutex
	serverAddr      net.Destination
	config          *ClientConfig
	client          mieruclient.Client
	clientIsRunning bool // do not use mieruclient.Client IsRunning() because Stop() takes too much time to finish
}

func NewClient(ctx context.Context, config *ClientConfig) (*Outbound, error) {
	serverAddr := net.Destination{
		Address: config.Address.AsAddress(),
		Port:    net.Port(config.Port),
	}
	if len(config.PortRange) > 0 {
		serverAddr.Port = 0
	}
	switch config.Protocol {
	case "", "tcp":
		serverAddr.Network = net.Network_TCP
	case "udp":
		serverAddr.Network = net.Network_UDP
	default:
		return nil, newError("unknown protocol")
	}
	return &Outbound{
		serverAddr: serverAddr,
		config:     config,
	}, nil
}

func (o *Outbound) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified")
	}

	var client mieruclient.Client
	o.Lock()
	if o.client == nil || !o.clientIsRunning {
		dialer := &dialerWrapper{
			dialer: dialer,
		}
		var resolver mierucommon.DNSResolver
		if outbound.Resolver != nil {
			resolver = &resolverWrapper{
				resolver: outbound.Resolver,
			}
		} else {
			resolver = &localResolverWrapper{
				resolver: localdns.New(),
			}
		}
		config, err := buildMieruClientConfig(o.config, dialer, resolver)
		if err != nil {
			o.Unlock()
			return err
		}
		client = mieruclient.NewClient()
		if err = client.Store(config); err != nil {
			o.Unlock()
			return err
		}
		if err = client.Start(); err != nil {
			o.Unlock()
			return err
		}
		o.client = client
		o.clientIsRunning = true
	} else {
		client = o.client
	}
	o.Unlock()

	destination := outbound.Target

	newError("tunneling request to ", destination, " via ", o.serverAddr.NetAddr()).WriteToLog(session.ExportIDToError(ctx))

	addr := &mierumodel.NetAddrSpec{}
	if destination.Network == net.Network_TCP {
		addr.Net = "tcp"
	} else {
		addr.Net = "udp"
	}
	switch destination.Address.Family() {
	case net.AddressFamilyDomain:
		addr.AddrSpec = mierumodel.AddrSpec{
			FQDN: destination.Address.Domain(),
			Port: int(destination.Port),
		}
	case net.AddressFamilyIPv4, net.AddressFamilyIPv6:
		addr.AddrSpec = mierumodel.AddrSpec{
			IP:   destination.Address.IP(),
			Port: int(destination.Port),
		}
	}

	conn, err := client.DialContext(ctx, addr)
	if err != nil {
		return err
	}

	var reader buf.Reader
	var writer buf.Writer
	if destination.Network == net.Network_TCP {
		reader = buf.NewReader(conn)
		writer = buf.NewWriter(conn)
	} else {
		packetConn := udp.NewMonoDestUDPConn(&udpAssociateWrapper{
			UDPAssociateWrapper: mierucommon.NewUDPAssociateWrapper(mierucommon.NewPacketOverStreamTunnel(conn)),
		}, udp.NewMonoDestUDPAddr(destination.Address, destination.Port))
		reader = packetConn
		writer = packetConn
	}

	if err := task.Run(ctx, func() error {
		return buf.Copy(link.Reader, writer)
	}, func() error {
		return buf.Copy(reader, link.Writer)
	}); err != nil {
		return newError("connection ends").Base(err)
	}
	return nil
}

func (o *Outbound) InterfaceUpdate() {
	o.Close()
}

func (o *Outbound) Close() error {
	o.Lock()
	if o.client != nil && o.clientIsRunning {
		o.clientIsRunning = false
		go o.client.Stop() // this takes too much time to finish
	}
	o.Unlock()
	return nil
}

func (*Outbound) DisallowMuxCool() {}
