package shadowtls

import (
	"context"

	shadowtls "github.com/sagernet/sing-shadowtls"
	"github.com/sagernet/sing/common/bufio"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/singbridge"
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
	config     shadowtls.ClientConfig
	client     *shadowtls.Client
}

func NewClient(ctx context.Context, config *ClientConfig) (*Outbound, error) {
	serverAddr := net.Destination{
		Address: config.Address.AsAddress(),
		Port:    net.Port(config.Port),
		Network: net.Network_TCP,
	}
	if config.TlsSettings == nil {
		config.TlsSettings = &v2tls.Config{}
	}
	o := &Outbound{
		serverAddr: serverAddr,
		config: shadowtls.ClientConfig{
			Version:      int(config.Version),
			Password:     config.Password,
			Server:       singbridge.ToSocksAddr(serverAddr),
			Logger:       singbridge.NewLoggerWrapper(newError),
			TLSHandshake: shadowtls.DefaultTLSHandshakeFunc(config.Password, config.TlsSettings.GetTLSConfig(v2tls.WithDestination(serverAddr))),
		},
	}

	return o, nil
}

func (o *Outbound) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	if o.client == nil {
		var err error
		config := o.config
		config.Dialer = singbridge.NewDialerWrapper(dialer)
		o.client, err = shadowtls.NewClient(config)
		if err != nil {
			return err
		}
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

	serverConn, err := o.client.DialContext(ctx)
	if err != nil {
		return err
	}

	return singbridge.ReturnError(bufio.CopyConn(ctx, singbridge.NewPipeConnWrapper(link), serverConn))
}
