package tuic

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"sync"
	"time"

	"github.com/sagernet/sing-quic/tuic"
	"github.com/sagernet/sing/common/bufio"
	N "github.com/sagernet/sing/common/network"

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
	mutex       sync.Mutex
	serverAddr  net.Destination
	tuicOptions tuic.ClientOptions
	tuicClients map[internet.Dialer]*tuic.Client
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
		return nil, newError(err, "invalid uuid: ", config.Uuid)
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

	o.tuicOptions = tuic.ClientOptions{
		Context: ctx,
		TLSConfig: &tlsConfigWrapper{
			config: tlsConfig,
		},
		ServerAddress:     toSocksaddr(o.serverAddr),
		UUID:              uuid,
		Password:          config.Password,
		CongestionControl: config.CongestionControl,
		UDPStream:         config.UdpRelayMode == "quic",
		ZeroRTTHandshake:  config.ZeroRttHandshake,
		Heartbeat:         time.Second * time.Duration(config.Heartbeat),
	}

	o.tuicClients = make(map[internet.Dialer]*tuic.Client)

	return o, nil
}

func (o *Outbound) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	o.mutex.Lock()
	tuicClient, found := o.tuicClients[dialer]
	if !found {
		var err error
		options := o.tuicOptions
		options.Dialer = &dialerWrapper{dialer}
		tuicClient, err = tuic.NewClient(options)
		if err != nil {
			o.mutex.Unlock()
			return err
		}
		o.tuicClients[dialer] = tuicClient
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
		serverConn, err := tuicClient.DialConn(detachedCtx, toSocksaddr(destination))
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
		serverConn, err := tuicClient.ListenPacket(detachedCtx)
		if err != nil {
			return err
		}
		return returnError(bufio.CopyPacketConn(detachedCtx, packetConn, serverConn.(N.PacketConn)))
	}
}
