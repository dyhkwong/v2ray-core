package anytls

import (
	"context"
	"sync"
	"time"

	anytls "github.com/anytls/sing-anytls"
	"github.com/sagernet/sing/common/bufio"
	"github.com/sagernet/sing/common/uot"

	"github.com/v2fly/v2ray-core/v5/app/proxyman/outbound"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/singbridge"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

func init() {
	common.Must(common.RegisterConfig((*ClientConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewClient(ctx, config.(*ClientConfig))
	}))
}

type Outbound struct {
	ctx                      context.Context
	serverAddr               net.Destination
	password                 string
	idleSessionCheckInterval int64
	idleSessionTimeout       int64
	minIdleSession           int64
	client                   *anytls.Client
	clientAccess             sync.Mutex
	create                   sync.Mutex
	closed                   bool
}

func NewClient(ctx context.Context, config *ClientConfig) (*Outbound, error) {
	return &Outbound{
		ctx: ctx,
		serverAddr: net.Destination{
			Address: config.Address.AsAddress(),
			Port:    net.Port(config.Port),
			Network: net.Network_TCP,
		},
		password:                 config.Password,
		idleSessionCheckInterval: config.IdleSessionCheckInterval,
		idleSessionTimeout:       config.IdleSessionTimeout,
		minIdleSession:           config.MinIdleSession,
	}, nil
}

func (o *Outbound) getClient(dialer internet.Dialer) (*anytls.Client, error) {
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
	if streamSettings := handler.StreamSettings(); streamSettings == nil || streamSettings.SecurityType == "" {
		return nil, newError("tls not enabled")
	}
	client, err := anytls.NewClient(o.ctx, anytls.ClientConfig{
		Password:                 o.password,
		IdleSessionCheckInterval: time.Duration(o.idleSessionCheckInterval) * time.Second,
		IdleSessionTimeout:       time.Duration(o.idleSessionTimeout) * time.Second,
		MinIdleSession:           int(o.minIdleSession),
		DialOut: func(ctx context.Context) (net.Conn, error) {
			return dialer.Dial(ctx, o.serverAddr)
		},
		Logger: singbridge.NewLoggerWrapper(newError),
	})
	if err != nil {
		return nil, err
	}
	o.clientAccess.Lock()
	o.client = client
	o.clientAccess.Unlock()
	return client, nil
}

func (o *Outbound) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified")
	}
	destination := outbound.Target

	client, err := o.getClient(dialer)
	if err != nil {
		return err
	}

	newError("tunneling request to ", destination, " via ", o.serverAddr.NetAddr()).WriteToLog(session.ExportIDToError(ctx))

	var serverConn net.Conn
	if destination.Network == net.Network_TCP {
		serverConn, err = client.CreateProxy(ctx, singbridge.ToSocksAddr(destination))
	} else {
		serverConn, err = client.CreateProxy(ctx, uot.RequestDestination(uot.Version))
	}
	if err != nil {
		return err
	}
	if destination.Network == net.Network_TCP {
		return singbridge.ReturnError(bufio.CopyConn(ctx, singbridge.NewPipeConnWrapper(link), serverConn))
	} else {
		uotConn := uot.NewLazyConn(serverConn, uot.Request{Destination: singbridge.ToSocksAddr(destination)})
		return singbridge.ReturnError(bufio.CopyPacketConn(ctx, singbridge.NewPacketConnWrapper(link, destination), uotConn))
	}
}

func (o *Outbound) InterfaceUpdate() {
	o.clientAccess.Lock()
	if o.client != nil {
		o.client.Close()
		o.client = nil
	}
	o.clientAccess.Unlock()
}

func (o *Outbound) Close() error {
	o.closed = true
	o.clientAccess.Lock()
	if o.client != nil {
		o.client.Close()
		o.client = nil
	}
	o.clientAccess.Unlock()
	return nil
}
