package mieru

import (
	"context"
	"sync"

	mieruclient "github.com/enfein/mieru/v3/apis/client"
	mierucommon "github.com/enfein/mieru/v3/apis/common"
	mierumodel "github.com/enfein/mieru/v3/apis/model"

	"github.com/v2fly/v2ray-core/v5/app/proxyman/outbound"
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
	serverAddr   net.Destination
	config       *ClientConfig
	client       mieruclient.Client
	clientAccess sync.Mutex
	create       sync.Mutex
	closed       bool
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

func (o *Outbound) getClient(dialer internet.Dialer, resolver func(ctx context.Context, domain string) net.Address) (mieruclient.Client, error) {
	o.create.Lock()
	defer o.create.Unlock()
	if o.closed {
		return nil, newError("closed")
	}
	o.clientAccess.Lock()
	if o.client != nil {
		defer o.clientAccess.Unlock()
		return o.client, nil
	}
	o.clientAccess.Unlock()
	handler, ok := dialer.(*outbound.Handler)
	if !ok {
		panic("dialer is not *outbound.Handler")
	}
	if handler.MuxEnabled() {
		return nil, newError("mux enabled")
	}
	if handler.TransportLayerEnabled() {
		return nil, newError("transport layer enabled")
	}
	if streamSettings := handler.StreamSettings(); streamSettings != nil && streamSettings.SecurityType != "" {
		return nil, newError("tls enabled")
	}
	mieruDialer := &dialerWrapper{
		dialer: dialer,
	}
	var mieruResolver mierucommon.DNSResolver
	if resolver != nil {
		mieruResolver = &resolverWrapper{
			resolver: resolver,
		}
	} else {
		mieruResolver = &localResolverWrapper{
			resolver: localdns.New(),
		}
	}
	config, err := buildMieruClientConfig(o.config, mieruDialer, mieruResolver)
	if err != nil {
		return nil, err
	}
	client := mieruclient.NewClient()
	if err = client.Store(config); err != nil {
		return nil, err
	}
	if err = client.Start(); err != nil {
		return nil, err
	}
	o.clientAccess.Lock()
	o.client = client
	o.clientAccess.Unlock()
	return client, nil
}

func (o *Outbound) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	ob := session.OutboundFromContext(ctx)
	if ob == nil || !ob.Target.IsValid() {
		return newError("target not specified")
	}

	client, err := o.getClient(dialer, ob.Resolver)
	if err != nil {
		return err
	}

	destination := ob.Target

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

func (o *Outbound) Close() error {
	o.closed = true
	o.clientAccess.Lock()
	if o.client != nil {
		o.client.Stop()
		o.client = nil
	}
	o.clientAccess.Unlock()
	return nil
}
