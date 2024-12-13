package tuic

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"time"

	"github.com/sagernet/sing-quic/tuic"
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
	options    tuic.ClientOptions
	client     *tuic.Client
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

	switch config.UdpRelayMode {
	case "", "native", "quic":
	default:
		return nil, newError("invalid UDP relay mode: ", config.UdpRelayMode)
	}
	switch config.CongestionControl {
	case "", "bbr", "new_reno", "cubic":
	default:
		return nil, newError("invalid congestion control: ", config.CongestionControl)
	}

	if config.TlsSettings == nil {
		config.TlsSettings = &v2tls.Config{}
	}
	tlsConfig := config.TlsSettings.GetTLSConfig(v2tls.WithDestination(o.serverAddr))
	if len(config.TlsSettings.NextProtocol) == 0 {
		// TUIC does not send ALPN if not explicitly set
		tlsConfig.NextProtos = nil
	}
	serverName := tlsConfig.ServerName
	if config.DisableSni {
		tlsConfig.ServerName = ""
	}
	if !tlsConfig.InsecureSkipVerify && config.DisableSni {
		tlsConfig.InsecureSkipVerify = true
		tlsConfig.VerifyConnection = func(state tls.ConnectionState) error {
			verifyOptions := x509.VerifyOptions{
				Roots:         tlsConfig.RootCAs,
				DNSName:       serverName,
				Intermediates: x509.NewCertPool(),
			}
			for _, cert := range state.PeerCertificates[1:] {
				verifyOptions.Intermediates.AddCert(cert)
			}
			_, err := state.PeerCertificates[0].Verify(verifyOptions)
			return err
		}
	}

	o.options = tuic.ClientOptions{
		Context:           ctx,
		TLSConfig:         singbridge.NewTLSConfigWrapper(tlsConfig),
		ServerAddress:     singbridge.ToSocksAddr(o.serverAddr),
		UUID:              uuid,
		Password:          config.Password,
		CongestionControl: config.CongestionControl,
		UDPStream:         config.UdpRelayMode == "quic",
		ZeroRTTHandshake:  config.ZeroRttHandshake,
		Heartbeat:         time.Second * time.Duration(config.Heartbeat),
	}

	return o, nil
}

func (o *Outbound) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	if o.client == nil {
		var err error
		options := o.options
		options.Dialer = singbridge.NewDialerWrapper(dialer)
		o.client, err = tuic.NewClient(options)
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
		serverConn, err := o.client.ListenPacket(detachedCtx)
		if err != nil {
			return err
		}
		return singbridge.ReturnError(bufio.CopyPacketConn(detachedCtx, singbridge.NewPacketConnWrapper(link, destination), serverConn.(network.PacketConn)))
	}
}
