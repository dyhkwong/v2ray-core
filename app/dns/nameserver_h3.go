package dns

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"sync"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/net/cnc"
	"github.com/v2fly/v2ray-core/v5/common/signal/pubsub"
	"github.com/v2fly/v2ray-core/v5/common/task"
	"github.com/v2fly/v2ray-core/v5/common/uuid"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

type H3NameServer struct {
	sync.RWMutex
	ips        map[string]record
	pub        *pubsub.Service
	cleanup    *task.Periodic
	dispatcher routing.Dispatcher
	httpClient *http.Client
	dohURL     string
	name       string
}

// NewH3NameServer creates DOH server object for remote resolving.
func NewH3NameServer(url *url.URL, dispatcher routing.Dispatcher) (*DoHNameServer, error) {
	url.Scheme = "https"
	s := baseDOHNameServer(url, "H3")
	s.httpClient = &http.Client{
		Transport: &http3.RoundTripper{
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
				tr := quic.Transport{}
				switch conn := rawConn.(type) {
				case *internet.PacketConnWrapper:
					tr.Conn = conn.Conn
				case net.PacketConn:
					tr.Conn = conn
				default:
					tr.Conn = NewConnWrapper(conn)
				}
				return tr.DialEarly(ctx, rawConn.RemoteAddr(), tlsCfg, cfg)
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
		Transport: &http3.RoundTripper{
			Dial: func(ctx context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlyConnection, error) {
				dest, err := net.ParseDestination("udp:" + addr)
				if err != nil {
					return nil, err
				}
				rawConn, err := internet.DialSystem(ctx, dest, nil)
				if err != nil {
					return nil, err
				}
				tr := quic.Transport{}
				switch conn := rawConn.(type) {
				case *internet.PacketConnWrapper:
					tr.Conn = conn.Conn
				case net.PacketConn:
					tr.Conn = conn
				default:
					tr.Conn = NewConnWrapper(conn)
				}
				return tr.DialEarly(ctx, rawConn.RemoteAddr(), tlsCfg, cfg)
			},
		},
	}
	newError("DNS: created Local H3 client for ", url.String()).AtInfo().WriteToLog()
	return s
}

type connWrapper struct {
	net.Conn
	localAddr net.Addr
}

func (c *connWrapper) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	n, err = c.Read(p)
	return n, c.RemoteAddr(), err
}

func (c *connWrapper) WriteTo(p []byte, _ net.Addr) (n int, err error) {
	return c.Write(p)
}

func NewConnWrapper(conn net.Conn) net.PacketConn {
	// https://github.com/quic-go/quic-go/commit/8189e75be6121fdc31dc1d6085f17015e9154667#diff-4c6aaadced390f3ce9bec0a9c9bb5203d5fa85df79023e3e0eec423dc9baa946R48-R62
	uuid := uuid.New()
	return &connWrapper{
		Conn:      conn,
		localAddr: &net.UnixAddr{Name: uuid.String()},
	}
}

func (c *connWrapper) LocalAddr() net.Addr {
	return c.localAddr
}
