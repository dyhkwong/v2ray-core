package juicity

import (
	"context"
	"io"
	"sync"

	juicity "github.com/dyhkwong/sing-juicity"
	"github.com/sagernet/sing/common/bufio"
	"github.com/sagernet/sing/common/network"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
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
	mutex          sync.Mutex
	serverAddr     net.Destination
	juicityOptions juicity.ClientOptions
	juicityClients map[internet.Dialer]*juicity.Client
}

func NewClient(ctx context.Context, config *ClientConfig) (*Outbound, error) {
	o := &Outbound{
		serverAddr: net.Destination{
			Address: config.Address.AsAddress(),
			Port:    net.Port(config.Port),
			Network: net.Network_UDP,
		},
	}
	uuid, err := uuid.ParseString(config.Uuid)
	if err != nil {
		return nil, newError(err, "invalid uuid")
	}

	if config.TlsSettings == nil {
		config.TlsSettings = &v2tls.Config{}
	}
	tlsConfig := config.TlsSettings.GetTLSConfig(v2tls.WithDestination(o.serverAddr), v2tls.WithNextProto("h3"))

	o.juicityOptions = juicity.ClientOptions{
		Context: ctx,
		TLSConfig: &tlsConfigWrapper{
			config: tlsConfig,
		},
		ServerAddress: toSocksaddr(o.serverAddr),
		UUID:          uuid,
		Password:      config.Password,
	}

	o.juicityClients = make(map[internet.Dialer]*juicity.Client)

	return o, nil
}

func (o *Outbound) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	o.mutex.Lock()
	juicityClient, found := o.juicityClients[dialer]
	if !found {
		var err error
		options := o.juicityOptions
		options.Dialer = &dialerWrapper{dialer}
		juicityClient, err = juicity.NewClient(options)
		if err != nil {
			o.mutex.Unlock()
			return err
		}
		o.juicityClients[dialer] = juicityClient
	}
	o.mutex.Unlock()

	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified")
	}
	destination := outbound.Target

	newError("tunneling request to ", destination, " via ", o.serverAddr.NetAddr()).WriteToLog(session.ExportIDToError(ctx))

	detachedCtx := core.ToBackgroundDetachedContext(ctx)
	if destination.Network == net.Network_TCP {
		serverConn, err := juicityClient.DialConn(detachedCtx, toSocksaddr(destination))
		if err != nil {
			return err
		}
		conn := &pipeConnWrapper{
			W: link.Writer,
		}
		if ir, ok := link.Reader.(io.Reader); ok {
			conn.R = ir
		} else {
			conn.R = &buf.BufferedReader{Reader: link.Reader}
		}
		return returnError(bufio.CopyConn(detachedCtx, conn, serverConn))
	} else {
		packetConn := &packetConnWrapper{
			Reader: link.Reader,
			Writer: link.Writer,
			Dest:   destination,
		}
		serverConn, err := juicityClient.ListenPacket(detachedCtx, toSocksaddr(destination))
		if err != nil {
			return err
		}
		return returnError(bufio.CopyPacketConn(detachedCtx, packetConn, serverConn.(network.PacketConn)))
	}
}
