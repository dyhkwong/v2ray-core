package tuic

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/sagernet/sing-quic/tuic"
	"github.com/sagernet/sing/common/bufio"
	N "github.com/sagernet/sing/common/network"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/uuid"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

func init() {
	common.Must(common.RegisterConfig((*ClientConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewClient(ctx, config.(*ClientConfig))
	}))
}

type Outbound struct {
	mutex       sync.Mutex
	serverAddr  net.Destination
	dialer      internet.Dialer
	tuicOptions tuic.ClientOptions
	tuicClient  *tuic.Client
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
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.AllowInsecure,
	}

	if len(config.Alpn) > 0 {
		tlsConfig.NextProtos = config.Alpn
	}

	var serverName string
	switch {
	case len(config.ServerName) > 0:
		serverName = config.ServerName
	case config.Address.AsAddress().Family().IsIP():
		serverName = config.Address.AsAddress().IP().String()
	default:
		serverName = config.Address.AsAddress().String()
	}

	if config.DisableSni && !net.ParseAddress(serverName).Family().IsIP() {
		tlsConfig.ServerName = "127.0.0.1"
	} else {
		tlsConfig.ServerName = serverName
	}

	if config.DisableSni {
		tlsConfig.InsecureSkipVerify = true
		tlsConfig.VerifyConnection = func(state tls.ConnectionState) error {
			verifyOptions := x509.VerifyOptions{
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

	if len(config.Certificate) > 0 {
		certificate := []byte(strings.Join(config.Certificate, "\n"))
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(certificate) {
			return nil, newError("failed to parse certificate")
		}
		tlsConfig.RootCAs = certPool
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

	o.tuicOptions = tuic.ClientOptions{
		TLSConfig: &tlsConfigWrapper{
			config: tlsConfig,
		},
		ServerAddress:     toSocksaddr(o.serverAddr),
		UUID:              uuid,
		Password:          config.Password,
		CongestionControl: config.CongestionControl,
		UDPStream:         config.UdpRelayMode == "quic",
		ZeroRTTHandshake:  config.ZeroRttHandshake,
	}

	return o, nil
}

func (o *Outbound) updateDialer(ctx context.Context, dialer internet.Dialer) error {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	if o.dialer != dialer {
		if o.tuicClient != nil {
			_ = o.tuicClient.CloseWithError(os.ErrClosed)
		}
		o.tuicOptions.Context = ctx
		o.tuicOptions.Dialer = &dialerWrapper{dialer}
		tuicClient, err := tuic.NewClient(o.tuicOptions)
		if err != nil {
			return newError("failed to create TUIC client").Base(err)
		}
		o.tuicClient = tuicClient
		o.dialer = dialer
	}
	return nil
}

func (o *Outbound) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified")
	}
	destination := outbound.Target

	if err := o.updateDialer(ctx, dialer); err != nil {
		return err
	}

	newError("tunneling request to ", destination, " via ", o.serverAddr.NetAddr()).WriteToLog(session.ExportIDToError(ctx))

	if destination.Network == net.Network_TCP {
		serverConn, err := o.tuicClient.DialConn(ctx, toSocksaddr(destination))
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
		return returnError(bufio.CopyConn(ctx, conn, serverConn))
	} else {
		packetConn := &packetConnWrapper{
			Reader: link.Reader,
			Writer: link.Writer,
			Dest:   destination,
		}
		serverConn, err := o.tuicClient.ListenPacket(ctx)
		if err != nil {
			return err
		}
		return returnError(bufio.CopyPacketConn(ctx, packetConn, serverConn.(N.PacketConn)))
	}
}
