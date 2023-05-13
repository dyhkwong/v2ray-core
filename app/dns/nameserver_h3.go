//go:build !confonly

package dns

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"

	core "github.com/v2fly/v2ray-core/v4"
	"github.com/v2fly/v2ray-core/v4/common/net"
	"github.com/v2fly/v2ray-core/v4/common/net/cnc"
	"github.com/v2fly/v2ray-core/v4/features/routing"
	"github.com/v2fly/v2ray-core/v4/transport/internet"
)

// NewH3NameServer creates DOH server object for remote resolving.
func NewH3NameServer(url *url.URL, dispatcher routing.Dispatcher) (*DoHNameServer, error) {
	url.Scheme = "https"
	s := baseDOHNameServer(url, "H3", "quic")
	s.httpClient = &http.Client{
		Transport: &http3.Transport{
			Dial: func(ctx context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (*quic.Conn, error) {
				dest, err := net.ParseDestination("udp:" + addr)
				if err != nil {
					return nil, err
				}
				detachedCtx := core.ToBackgroundDetachedContext(ctx)
				link, err := dispatcher.Dispatch(detachedCtx, dest)
				if err != nil {
					return nil, err
				}
				rawConn := cnc.NewConnection(
					cnc.ConnectionInputMulti(link.Writer),
					cnc.ConnectionOutputMultiUDP(link.Reader),
				)
				return quic.Dial(detachedCtx, internet.NewConnWrapper(rawConn), rawConn.RemoteAddr(), tlsCfg, cfg)
			},
		},
	}
	newError("DNS: created Remote H3 client for ", url.String()).AtInfo().WriteToLog()
	return s, nil
}

// NewH3LocalNameServer creates DOH client object for local resolving
func NewH3LocalNameServer(url *url.URL) *DoHNameServer {
	url.Scheme = "https"
	s := baseDOHNameServer(url, "H3L", "quic")
	s.httpClient = &http.Client{
		Transport: &http3.Transport{
			Dial: func(ctx context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (*quic.Conn, error) {
				dest, err := net.ParseDestination("udp:" + addr)
				if err != nil {
					return nil, err
				}
				rawConn, err := internet.DialSystem(ctx, dest, nil)
				if err != nil {
					return nil, err
				}
				var packetConn net.PacketConn
				switch rawConn := rawConn.(type) {
				case *internet.PacketConnWrapper:
					packetConn = rawConn.Conn
				case net.PacketConn:
					packetConn = rawConn
				default:
					packetConn = internet.NewConnWrapper(rawConn)
				}
				return quic.Dial(ctx, packetConn, rawConn.RemoteAddr(), tlsCfg, cfg)
			},
		},
	}
	newError("DNS: created Local H3 client for ", url.String()).AtInfo().WriteToLog()
	return s
}
