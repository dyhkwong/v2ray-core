package dns

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/net/cnc"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

// NewH3NameServer creates DOH server object for remote resolving.
func NewH3NameServer(url *url.URL, dispatcher routing.Dispatcher) (*DoHNameServer, error) {
	url.Scheme = "https"
	s := baseDOHNameServer(url, "H3")
	s.httpClient = &http.Client{
		Transport: &http3.Transport{
			Dial: func(ctx context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlyConnection, error) {
				dest, err := net.ParseDestination("udp:" + addr)
				if err != nil {
					return nil, err
				}
				ctx = core.ToBackgroundDetachedContext(ctx)
				link, err := dispatcher.Dispatch(ctx, dest)
				if err != nil {
					return nil, err
				}
				rawConn := cnc.NewConnection(
					cnc.ConnectionInputMulti(link.Writer),
					cnc.ConnectionOutputMultiUDP(link.Reader),
				)
				return quic.DialEarly(ctx, internet.NewQUICConnWrapper(rawConn), rawConn.RemoteAddr(), tlsCfg, cfg)
			},
		},
	}
	newError("DNS: created Remote H3 client for ", url.String()).AtInfo().WriteToLog()
	return s, nil
}

// NewH3LocalNameServer creates DOH client object for local resolving
func NewH3LocalNameServer(url *url.URL) *DoHNameServer {
	url.Scheme = "https"
	s := baseDOHNameServer(url, "H3L")
	s.httpClient = &http.Client{
		Transport: &http3.Transport{
			Dial: func(ctx context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlyConnection, error) {
				dest, err := net.ParseDestination("udp:" + addr)
				if err != nil {
					return nil, err
				}
				rawConn, err := internet.DialSystem(ctx, dest, nil)
				if err != nil {
					return nil, err
				}
				var packetConn net.PacketConn
				switch conn := rawConn.(type) {
				case *internet.PacketConnWrapper:
					if udpConn, ok := conn.Conn.(*net.UDPConn); ok {
						packetConn = internet.NewQUICUDPConnWrapper(udpConn)
					} else {
						packetConn = internet.NewQUICPacketConnWrapper(conn.Conn)
					}
				case net.PacketConn:
					if udpConn, ok := conn.(*net.UDPConn); ok {
						packetConn = internet.NewQUICUDPConnWrapper(udpConn)
					} else {
						packetConn = internet.NewQUICPacketConnWrapper(conn)
					}
				default:
					packetConn = internet.NewQUICConnWrapper(conn)
				}
				return quic.DialEarly(ctx, packetConn, rawConn.RemoteAddr(), tlsCfg, cfg)
			},
		},
	}
	newError("DNS: created Local H3 client for ", url.String()).AtInfo().WriteToLog()
	return s
}
