package juicity

import (
	"context"
	"os"
	"sync"

	juicity "github.com/dyhkwong/sing-juicity"
	"github.com/sagernet/sing/common/bufio"
	"github.com/sagernet/sing/common/network"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/app/proxyman/outbound"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/singbridge"
	"github.com/v2fly/v2ray-core/v5/common/uuid"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	v2tls "github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

func init() {
	common.Must(common.RegisterConfig((*ClientConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewClient(ctx, config.(*ClientConfig))
	}))
}

type Outbound struct {
	serverAddr net.Destination
	options    juicity.ClientOptions
	client     *juicity.Client
	clientLock sync.RWMutex
	createLock sync.Mutex
	closed     chan struct{}
}

func NewClient(ctx context.Context, config *ClientConfig) (*Outbound, error) {
	o := &Outbound{
		serverAddr: net.Destination{
			Address: config.Address.AsAddress(),
			Port:    net.Port(config.Port),
			Network: net.Network_UDP,
		},
		closed: make(chan struct{}),
	}
	uuid, err := uuid.ParseString(config.Uuid)
	if err != nil {
		return nil, newError("invalid uuid")
	}

	o.options = juicity.ClientOptions{
		Context:       ctx,
		ServerAddress: singbridge.ToSocksAddr(o.serverAddr),
		UUID:          uuid,
		Password:      config.Password,
	}

	return o, nil
}

func (o *Outbound) newClient(dialer internet.Dialer) (*juicity.Client, error) {
	select {
	case <-o.closed:
		return nil, newError("closed")
	default:
	}
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
	streamSettings := handler.StreamSettings()
	if streamSettings == nil || streamSettings.SecurityType != "v2ray.core.transport.internet.tls.Config" {
		return nil, newError("tls not enabled")
	}
	tlsSettings, ok := streamSettings.SecuritySettings.(*v2tls.Config)
	if !ok {
		return nil, newError("tls not enabled")
	}
	tlsConfig := tlsSettings.GetTLSConfig(v2tls.WithDestination(o.serverAddr), v2tls.WithNextProto("h3"))

	options := o.options
	options.TLSConfig = singbridge.NewTLSConfigWrapper(tlsConfig)
	options.Dialer = singbridge.NewDialerWrapper(dialer)
	return juicity.NewClient(options)
}

func (o *Outbound) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	o.clientLock.RLock()
	client := o.client
	o.clientLock.RUnlock()
	if client == nil {
		var err error
		o.createLock.Lock()
		client, err = o.newClient(dialer)
		if err != nil {
			o.createLock.Unlock()
			return err
		}
		o.clientLock.Lock()
		o.client = client
		o.clientLock.Unlock()
		o.createLock.Unlock()
	}

	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified")
	}
	destination := outbound.Target

	newError("tunneling request to ", destination, " via ", o.serverAddr.NetAddr()).WriteToLog(session.ExportIDToError(ctx))

	detachedCtx := core.ToBackgroundDetachedContext(ctx)
	if destination.Network == net.Network_TCP {
		serverConn, err := client.DialConn(detachedCtx, singbridge.ToSocksAddr(destination))
		if err != nil {
			return err
		}
		return singbridge.ReturnError(bufio.CopyConn(detachedCtx, singbridge.NewPipeConnWrapper(link), serverConn))
	} else {
		serverConn, err := client.ListenPacket(detachedCtx, singbridge.ToSocksAddr(destination))
		if err != nil {
			return err
		}
		return singbridge.ReturnError(bufio.CopyPacketConn(detachedCtx, singbridge.NewPacketConnWrapper(link, destination), serverConn.(network.PacketConn)))
	}
}

func (o *Outbound) InterfaceUpdate() {
	o.clientLock.RLock()
	client := o.client
	o.clientLock.RUnlock()
	if client != nil {
		_ = client.CloseWithError(newError("network changed"))
	}
}

func (o *Outbound) Close() error {
	close(o.closed)
	o.clientLock.RLock()
	client := o.client
	o.clientLock.RUnlock()
	if client != nil {
		return client.CloseWithError(os.ErrClosed)
	}
	return nil
}
