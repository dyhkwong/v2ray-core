package juicity

import (
	"context"

	juicity "github.com/dyhkwong/sing-juicity"
	"github.com/sagernet/sing/common/bufio"
	"github.com/sagernet/sing/common/network"

	core "github.com/v2fly/v2ray-core/v5"
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

	o.options = juicity.ClientOptions{
		Context:       ctx,
		TLSConfig:     singbridge.NewTLSConfigWrapper(tlsConfig),
		ServerAddress: singbridge.ToSocksAddr(o.serverAddr),
		UUID:          uuid,
		Password:      config.Password,
	}

	return o, nil
}

func (o *Outbound) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	if o.client == nil {
		var err error
		options := o.options
		options.Dialer = singbridge.NewDialerWrapper(dialer)
		o.client, err = juicity.NewClient(options)
		if err != nil {
			return err
		}
	}

	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified")
	}
	destination := outbound.Target

	newError("tunneling request to ", destination, " via ", o.serverAddr.NetAddr()).WriteToLog(session.ExportIDToError(ctx))

	detachedCtx := core.ToBackgroundDetachedContext(ctx)
	if destination.Network == net.Network_TCP {
		serverConn, err := o.client.DialConn(detachedCtx, singbridge.ToSocksAddr(destination))
		if err != nil {
			return err
		}
		return singbridge.ReturnError(bufio.CopyConn(detachedCtx, singbridge.NewPipeConnWrapper(link), serverConn))
	} else {
		serverConn, err := o.client.ListenPacket(detachedCtx, singbridge.ToSocksAddr(destination))
		if err != nil {
			return err
		}
		return singbridge.ReturnError(bufio.CopyPacketConn(detachedCtx, singbridge.NewPacketConnWrapper(link, destination), serverConn.(network.PacketConn)))
	}
}
