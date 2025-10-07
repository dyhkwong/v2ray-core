package http3

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/bytespool"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/retry"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal"
	"github.com/v2fly/v2ray-core/v5/common/task"
	"github.com/v2fly/v2ray-core/v5/features/policy"
	"github.com/v2fly/v2ray-core/v5/features/stats"
	"github.com/v2fly/v2ray-core/v5/proxy"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	v2tls "github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

type Client struct {
	config        *ClientConfig
	serverAddress net.Destination
	policyManager policy.Manager
	transport     *http3.Transport
	transportLock sync.Mutex
}

func (c *Client) Close() error {
	c.transportLock.Lock()
	c.transport = nil
	c.transportLock.Unlock()
	return nil
}

func NewClient(ctx context.Context, config *ClientConfig) (*Client, error) {
	v := core.MustFromContext(ctx)
	serverAddress := net.Destination{
		Address: config.Address.AsAddress(),
		Port:    net.Port(config.Port),
		Network: net.Network_UDP,
	}
	return &Client{
		config:        config,
		serverAddress: serverAddress,
		policyManager: v.GetFeature(policy.ManagerType()).(policy.Manager),
	}, nil
}

func (c *Client) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified.")
	}
	target := outbound.Target
	targetAddr := target.NetAddr()

	if target.Network == net.Network_UDP {
		return newError("UDP is not supported by HTTP outbound")
	}

	var conn internet.Connection

	var firstPayload []byte
	if reader, ok := link.Reader.(buf.TimeoutReader); ok {
		waitTime := proxy.FirstPayloadTimeout
		if mbuf, _ := reader.ReadMultiBufferTimeout(waitTime); mbuf != nil {
			mlen := mbuf.Len()
			firstPayload = bytespool.Alloc(mlen)
			mbuf, _ = buf.SplitBytes(mbuf, firstPayload)
			firstPayload = firstPayload[:mlen]

			buf.ReleaseMulti(mbuf)
			defer bytespool.Free(firstPayload)
		}
	}

	if err := retry.ExponentialBackoff(5, 100).On(func() error {
		httpConn, err := c.setupHTTPTunnel(ctx, targetAddr, dialer, firstPayload, c.config)
		if err != nil {
			return err
		}
		conn = httpConn
		return nil
	}); err != nil {
		return newError("failed to find an available destination").Base(err)
	}

	newError("tunneling request to ", target, " via ", c.serverAddress.NetAddr()).WriteToLog(session.ExportIDToError(ctx))

	defer func() {
		if err := conn.Close(); err != nil {
			newError("failed to closed connection").Base(err).WriteToLog(session.ExportIDToError(ctx))
		}
	}()

	p := c.policyManager.ForLevel(c.config.Level)

	ctx, cancel := context.WithCancel(ctx)
	timer := signal.CancelAfterInactivity(ctx, cancel, p.Timeouts.ConnectionIdle)

	requestFunc := func() error {
		defer timer.SetTimeout(p.Timeouts.DownlinkOnly)
		return buf.Copy(link.Reader, buf.NewWriter(conn), buf.UpdateActivity(timer))
	}
	responseFunc := func() error {
		defer timer.SetTimeout(p.Timeouts.UplinkOnly)
		return buf.Copy(buf.NewReader(conn), link.Writer, buf.UpdateActivity(timer))
	}

	responseDonePost := task.OnSuccess(responseFunc, task.Close(link.Writer))
	if err := task.Run(ctx, requestFunc, responseDonePost); err != nil {
		return newError("connection ends").Base(err)
	}

	return nil
}

// setupHTTPTunnel will create a socket tunnel via HTTP CONNECT method
func (c *Client) setupHTTPTunnel(ctx context.Context, target string, dialer internet.Dialer, firstPayload []byte, config *ClientConfig) (net.Conn, error) {
	dest := net.Destination{
		Address: config.Address.AsAddress(),
		Port:    net.Port(config.Port),
		Network: net.Network_UDP,
	}
	c.transportLock.Lock()
	transport := c.transport
	if transport == nil {
		transport = &http3.Transport{
			QUICConfig: &quic.Config{
				KeepAlivePeriod: time.Second * 10,
			},
			Dial: func(_ context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (*quic.Conn, error) {
				detachedCtx := core.ToBackgroundDetachedContext(ctx)
				rawConn, err := dialer.Dial(detachedCtx, dest)
				if err != nil {
					return nil, err
				}
				var readCounter, writeCounter stats.Counter
				iConn := rawConn
				statConn, ok := iConn.(*internet.StatCouterConnection)
				if ok {
					iConn = statConn.Connection
				}
				var packetConn net.PacketConn
				switch iConn := iConn.(type) {
				case *internet.PacketConnWrapper:
					if statConn != nil {
						readCounter = statConn.ReadCounter
						writeCounter = statConn.WriteCounter
					}
					packetConn = wrapPacketConn(iConn.Conn, readCounter, writeCounter)
				case net.PacketConn:
					if statConn != nil {
						readCounter = statConn.ReadCounter
						writeCounter = statConn.WriteCounter
					}
					packetConn = wrapPacketConn(iConn, readCounter, writeCounter)
				default:
					packetConn = internet.NewConnWrapper(iConn)
				}
				tlsSettings := config.TlsSettings
				if tlsSettings == nil {
					tlsSettings = &v2tls.Config{}
				}
				quicConn, err := quic.Dial(detachedCtx, packetConn, rawConn.RemoteAddr(),
					tlsSettings.GetTLSConfig(v2tls.WithNextProto("h3"), v2tls.WithDestination(dest)),
					cfg,
				)
				if err != nil {
					rawConn.Close()
					return nil, err
				}
				return quicConn, nil
			},
		}
		c.transport = transport
	}
	c.transportLock.Unlock()

	req := &http.Request{
		Method: http.MethodConnect,
		URL: &url.URL{
			Host:   target,
			Scheme: "https",
		},
		Header: make(http.Header),
		Host:   target,
	}
	if config.Username != nil || config.Password != nil {
		auth := config.GetUsername() + ":" + config.GetPassword()
		req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	}
	headers := config.GetHeaders()
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	pr, pw := io.Pipe()
	req.Body = pr
	var pErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		_, pErr = pw.Write(firstPayload)
		wg.Done()
	}()
	resp, err := transport.RoundTrip(req) // nolint: bodyclose
	if err != nil {
		return nil, err
	}
	wg.Wait()
	if pErr != nil {
		return nil, pErr
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newError("Proxy responded with non 200 code: " + resp.Status)
	}
	return &httpConn{in: pw, out: resp.Body}, err
}

type httpConn struct {
	in  *io.PipeWriter
	out io.ReadCloser
}

func (c *httpConn) Read(p []byte) (n int, err error) {
	n, err = c.out.Read(p)
	return n, err
}

func (c *httpConn) Write(p []byte) (n int, err error) {
	n, err = c.in.Write(p)
	return n, err
}

func (c *httpConn) RemoteAddr() net.Addr {
	return &net.UDPAddr{
		IP:   []byte{0, 0, 0, 0},
		Port: 0,
	}
}

func (c *httpConn) LocalAddr() net.Addr {
	return &net.UDPAddr{
		IP:   []byte{0, 0, 0, 0},
		Port: 0,
	}
}

func (c *httpConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *httpConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *httpConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (c *httpConn) Close() error {
	c.in.Close()
	return c.out.Close()
}

func init() {
	common.Must(common.RegisterConfig((*ClientConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewClient(ctx, config.(*ClientConfig))
	}))
}

func (*Client) DisallowMuxCool() {}
