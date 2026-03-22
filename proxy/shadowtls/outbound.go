package shadowtls

import (
	"context"
	gotls "crypto/tls"
	"sync"

	shadowtls "github.com/sagernet/sing-shadowtls"
	"github.com/sagernet/sing/common/bufio"

	"github.com/v2fly/v2ray-core/v5/app/proxyman/outbound"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/singbridge"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

func init() {
	common.Must(common.RegisterConfig((*ClientConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewClient(ctx, config.(*ClientConfig))
	}))
}

type Outbound struct {
	serverAddr net.Destination
	config     shadowtls.ClientConfig
	client     *shadowtls.Client
	clientLock sync.RWMutex
	createLock sync.Mutex
	closed     chan struct{}
}

func NewClient(ctx context.Context, config *ClientConfig) (*Outbound, error) {
	serverAddr := net.Destination{
		Address: config.Address.AsAddress(),
		Port:    net.Port(config.Port),
		Network: net.Network_TCP,
	}
	o := &Outbound{
		serverAddr: serverAddr,
		config: shadowtls.ClientConfig{
			Version:  int(config.Version),
			Password: config.Password,
			Server:   singbridge.ToSocksAddr(serverAddr),
			Logger:   singbridge.NewLoggerWrapper(newError),
		},
		closed: make(chan struct{}),
	}
	return o, nil
}

func (o *Outbound) newClient(ctx context.Context, dialer internet.Dialer) (*shadowtls.Client, error) {
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
	tlsSettings, ok := streamSettings.SecuritySettings.(*tls.Config)
	if !ok {
		return nil, newError("tls not enabled")
	}
	tlsConfig := tlsSettings.GetTLSConfigWithContext(ctx, tls.WithDestination(o.serverAddr))
	var tlsHandshakeFunc shadowtls.TLSHandshakeFunc
	switch o.config.Version {
	case 0, 2:
		tlsHandshakeFunc = func(ctx context.Context, conn net.Conn, _ shadowtls.TLSSessionIDGeneratorFunc) error {
			tlsConn := gotls.Client(conn, tlsConfig)
			return tlsConn.HandshakeContext(ctx)
		}
	case 3:
		tlsHandshakeFunc = shadowtls.DefaultTLSHandshakeFunc(o.config.Password, tlsConfig)
	default:
		return nil, newError("unknown version")
	}

	config := o.config
	config.TLSHandshake = tlsHandshakeFunc
	return shadowtls.NewClient(config)
}

func (o *Outbound) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	o.clientLock.RLock()
	client := o.client
	o.clientLock.RUnlock()
	if client == nil {
		var err error
		o.createLock.Lock()
		client, err = o.newClient(ctx, dialer)
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
	if destination.Network != net.Network_TCP {
		return newError("only TCP is supported")
	}
	newError("tunneling request to ", destination, " via ", o.serverAddr.NetAddr()).WriteToLog(session.ExportIDToError(ctx))

	serverConn, err := client.DialContext(ctx)
	if err != nil {
		return err
	}

	return singbridge.ReturnError(bufio.CopyConn(ctx, singbridge.NewPipeConnWrapper(link), serverConn))
}

func (o *Outbound) Close() error {
	close(o.closed)
	return nil
}
