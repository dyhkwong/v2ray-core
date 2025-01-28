package http3

import (
	"context"
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
	"github.com/v2fly/v2ray-core/v5/proxy"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	v2tls "github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

type Client struct {
	config        *ClientConfig
	policyManager policy.Manager
}

type h3Conn struct {
	rawConn net.Conn
	h3Conn  *http3.ClientConn
}

var (
	cachedH3Mutex sync.Mutex
	cachedH3Conns map[net.Destination]h3Conn
)

func NewClient(ctx context.Context, config *ClientConfig) (*Client, error) {
	if config.TlsSettings == nil {
		return nil, newError("empty TLS settings")
	}
	v := core.MustFromContext(ctx)
	return &Client{
		config:        config,
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
		netConn, err := setUpHTTPTunnel(ctx, targetAddr, dialer, firstPayload, c.config)
		if err != nil {
			return err
		}
		conn = internet.Connection(netConn)
		return nil
	}); err != nil {
		return newError("failed to find an available destination").Base(err)
	}

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

// setUpHTTPTunnel will create a socket tunnel via HTTP CONNECT method
func setUpHTTPTunnel(ctx context.Context, target string, dialer internet.Dialer, firstPayload []byte, config *ClientConfig) (net.Conn, error) {
	req := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Host: target},
		Header: make(http.Header),
		Host:   target,
	}

	dest := net.Destination{
		Address: config.Address.AsAddress(),
		Port:    net.Port(config.Port),
		Network: net.Network_UDP,
	}
	username, password, headers := config.GetUsername(), config.GetPassword(), config.GetHeaders()
	if username != "" && password != "" {
		auth := username + ":" + password
		req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	connectHTTP3 := func(rawConn net.Conn, h3clientConn *http3.ClientConn) (net.Conn, error) {
		pr, pw := io.Pipe()
		req.Body = pr

		var pErr error
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			_, pErr = pw.Write(firstPayload)
			wg.Done()
		}()

		resp, err := h3clientConn.RoundTrip(req) // nolint: bodyclose
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
		return newHTTP3Conn(rawConn, pw, resp.Body), nil
	}

	cachedH3Mutex.Lock()
	cachedConn, cachedConnFound := cachedH3Conns[dest]
	cachedH3Mutex.Unlock()

	if cachedConnFound {
		rc, cc := cachedConn.rawConn, cachedConn.h3Conn
		select {
		case <-cc.Context().Done():
			cc.CloseWithError(0, "")
			rc.Close()
			cachedH3Mutex.Lock()
			delete(cachedH3Conns, dest)
			cachedH3Mutex.Unlock()
		default:
			proxyConn, err := connectHTTP3(rc, cc)
			if err != nil {
				cc.CloseWithError(0, "")
				rc.Close()
				cachedH3Mutex.Lock()
				delete(cachedH3Conns, dest)
				cachedH3Mutex.Unlock()
				return nil, err
			}
			return proxyConn, nil
		}
	}

	rawConn, err := dialer.Dial(ctx, dest)
	if err != nil {
		return nil, err
	}

	iConn := rawConn
	if statConn, ok := iConn.(*internet.StatCouterConnection); ok {
		iConn = statConn.Connection
	}

	var pc net.PacketConn
	switch c := iConn.(type) {
	case *internet.PacketConnWrapper:
		pc = c.Conn
	case net.PacketConn:
		pc = c
	default:
		pc = internet.NewConnWrapper(c)
	}

	quicConn, err := quic.DialEarly(ctx, pc, rawConn.RemoteAddr(),
		config.TlsSettings.GetTLSConfig(v2tls.WithNextProto("h3"), v2tls.WithDestination(dest)),
		&quic.Config{
			HandshakeIdleTimeout: time.Second * 8,
		})
	if err != nil {
		rawConn.Close()
		return nil, err
	}

	t := http3.Transport{}
	h3clientConn := t.NewClientConn(quicConn)
	proxyConn, err := connectHTTP3(rawConn, h3clientConn)
	if err != nil {
		quicConn.CloseWithError(0, "")
		rawConn.Close()
		return nil, err
	}

	cachedH3Mutex.Lock()
	if cachedH3Conns == nil {
		cachedH3Conns = make(map[net.Destination]h3Conn)
	}

	cachedH3Conns[dest] = h3Conn{
		rawConn: rawConn,
		h3Conn:  h3clientConn,
	}
	cachedH3Mutex.Unlock()

	return proxyConn, err
}

func newHTTP3Conn(c net.Conn, pipedReqBody *io.PipeWriter, respBody io.ReadCloser) net.Conn {
	return &http3Conn{Conn: c, in: pipedReqBody, out: respBody}
}

type http3Conn struct {
	net.Conn
	in  *io.PipeWriter
	out io.ReadCloser
}

func (h *http3Conn) Read(p []byte) (n int, err error) {
	return h.out.Read(p)
}

func (h *http3Conn) Write(p []byte) (n int, err error) {
	return h.in.Write(p)
}

func (h *http3Conn) Close() error {
	h.in.Close()
	return h.out.Close()
}

func init() {
	common.Must(common.RegisterConfig((*ClientConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewClient(ctx, config.(*ClientConfig))
	}))
}
