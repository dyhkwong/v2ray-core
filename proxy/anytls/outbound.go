package anytls

import (
	"context"
	"io"
	"sync"
	"time"

	anytls "github.com/anytls/sing-anytls"
	"github.com/sagernet/sing/common/bufio"
	"github.com/sagernet/sing/common/uot"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

func init() {
	common.Must(common.RegisterConfig((*ClientConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewClient(ctx, config.(*ClientConfig))
	}))
}

type Outbound struct {
	mutex                    sync.Mutex
	ctx                      context.Context
	serverAddr               net.Destination
	password                 string
	idleSessionCheckInterval int64
	idleSessionTimeout       int64
	minIdleSession           int64
	clients                  map[internet.Dialer]*anytls.Client
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
		clients:                  make(map[internet.Dialer]*anytls.Client),
	}, nil
}

func (o *Outbound) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	var err error
	o.mutex.Lock()
	client, found := o.clients[dialer]
	if !found {
		client, err = anytls.NewClient(o.ctx, anytls.ClientConfig{
			Password:                 o.password,
			IdleSessionCheckInterval: time.Duration(o.idleSessionCheckInterval) * time.Second,
			IdleSessionTimeout:       time.Duration(o.idleSessionTimeout) * time.Second,
			MinIdleSession:           int(o.minIdleSession),
			DialOut: func(ctx context.Context) (net.Conn, error) {
				return dialer.Dial(ctx, o.serverAddr)
			},
			Logger: newLogger(newError),
		})
		if err != nil {
			o.mutex.Unlock()
			return err
		}
		o.clients[dialer] = client
	}
	o.mutex.Unlock()

	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified")
	}
	destination := outbound.Target

	newError("tunneling request to ", destination, " via ", o.serverAddr.NetAddr()).WriteToLog(session.ExportIDToError(ctx))

	var serverConn net.Conn
	if destination.Network == net.Network_TCP {
		serverConn, err = client.CreateProxy(ctx, toSocksaddr(destination))
	} else {
		serverConn, err = client.CreateProxy(ctx, uot.RequestDestination(uot.Version))
	}
	if err != nil {
		return err
	}
	if destination.Network == net.Network_TCP {
		conn := &pipeConnWrapper{
			W: link.Writer,
		}
		if ir, ok := link.Reader.(io.Reader); ok {
			conn.R = ir
		} else {
			conn.R = &buf.BufferedReader{Reader: link.Reader}
		}
		return returnError(bufio.CopyConn(ctx, conn, serverConn))
	} else {
		packetConn := &packetConnWrapper{
			Reader: link.Reader,
			Writer: link.Writer,
			Dest:   destination,
		}
		uotConn := uot.NewLazyConn(serverConn, uot.Request{Destination: toSocksaddr(destination)})
		return returnError(bufio.CopyPacketConn(ctx, packetConn, uotConn))
	}
}
