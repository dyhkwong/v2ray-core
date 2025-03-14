package http3

import (
	"container/list"
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
	"github.com/v2fly/v2ray-core/v5/app/proxyman/outbound"
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
	serverAddress net.Destination
	config        *ClientConfig
	policyManager policy.Manager
	transport     *http3.Transport
	cachedH3Mutex sync.Mutex
	cachedH3Conns list.List
}

func (c *Client) Close() error {
	c.cachedH3Mutex.Lock()
	for elem := c.cachedH3Conns.Front(); elem != nil; elem = elem.Next() {
		_ = elem.Value.(*h3Conn).h3Conn.CloseWithError(0, "")
		_ = elem.Value.(*h3Conn).quicConn.CloseWithError(0, "")
		_ = elem.Value.(*h3Conn).rawConn.Close()
		c.cachedH3Conns.Remove(elem)
	}
	c.cachedH3Mutex.Unlock()
	return nil
}

type h3Conn struct {
	rawConn      net.Conn
	quicConn     *quic.Conn
	h3Conn       *http3.ClientConn
	readCounter  stats.Counter
	writeCounter stats.Counter
}

func NewClient(ctx context.Context, config *ClientConfig) (*Client, error) {
	serverAddress := net.Destination{
		Address: config.Address.AsAddress(),
		Port:    net.Port(config.Port),
		Network: net.Network_UDP,
	}
	v := core.MustFromContext(ctx)
	return &Client{
		serverAddress: serverAddress,
		config:        config,
		policyManager: v.GetFeature(policy.ManagerType()).(policy.Manager),
		transport:     &http3.Transport{},
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
		netConn, err := c.setupHTTPTunnel(ctx, targetAddr, dialer, firstPayload, c.config)
		if err != nil {
			return err
		}
		conn = netConn
		return nil
	}); err != nil {
		return newError("failed to find an available destination").Base(err)
	}

	defer func() {
		if err := conn.Close(); err != nil {
			newError("failed to closed connection").Base(err).WriteToLog(session.ExportIDToError(ctx))
		}
	}()

	newError("tunneling request to ", target, " via ", c.serverAddress.NetAddr()).WriteToLog(session.ExportIDToError(ctx))

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
	handler, ok := dialer.(*outbound.Handler)
	if !ok {
		panic("dialer is not *outbound.Handler")
	}
	if handler.MuxEnabled() {
		return nil, newError("mux enabled")
	}
	if handler.TransportLayerEnabled() {
		return nil, newError("transport layer enabled")
	}
	streamSettings := handler.StreamSettings()
	if streamSettings == nil || streamSettings.SecurityType != "v2ray.core.transport.internet.tls.Config" {
		return nil, newError("tls not enabled")
	}
	tlsSettings, ok := streamSettings.SecuritySettings.(*v2tls.Config)
	if !ok {
		return nil, newError("tls not enabled")
	}

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
	if config.Username != nil || config.Password != nil {
		auth := config.GetUsername() + ":" + config.GetPassword()
		req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	}
	headers := config.GetHeaders()
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	connectHTTP3 := func(rawConn net.Conn, quicConn *quic.Conn, h3clientConn *http3.ClientConn, readCounter, writeCounter stats.Counter, elem *list.Element) (net.Conn, error) {
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
			h3clientConn.CloseWithError(0, "")
			quicConn.CloseWithError(0, "")
			rawConn.Close()
			if elem != nil {
				c.cachedH3Mutex.Lock()
				c.cachedH3Conns.Remove(elem)
				c.cachedH3Mutex.Unlock()
			}
			return nil, err
		}

		wg.Wait()
		if pErr != nil {
			h3clientConn.CloseWithError(0, "")
			quicConn.CloseWithError(0, "")
			rawConn.Close()
			if elem != nil {
				c.cachedH3Mutex.Lock()
				c.cachedH3Conns.Remove(elem)
				c.cachedH3Mutex.Unlock()
			}
			return nil, pErr
		}

		if resp.StatusCode != http.StatusOK {
			return nil, newError("Proxy responded with non 200 code: " + resp.Status)
		}
		return &http3Conn{
			Conn:         rawConn,
			in:           pw,
			out:          resp.Body,
			readCounter:  readCounter,
			writeCounter: writeCounter,
		}, nil
	}

	c.cachedH3Mutex.Lock()
	elem := c.cachedH3Conns.Front()
	c.cachedH3Mutex.Unlock()

	if elem != nil {
		readCounter, writeCounter := elem.Value.(*h3Conn).readCounter, elem.Value.(*h3Conn).writeCounter
		rawConn, quicConn, h3Conn := elem.Value.(*h3Conn).rawConn, elem.Value.(*h3Conn).quicConn, elem.Value.(*h3Conn).h3Conn
		select {
		case <-h3Conn.Context().Done():
		default:
			proxyConn, err := connectHTTP3(rawConn, quicConn, h3Conn, readCounter, writeCounter, elem)
			if err != nil {
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
	if trackedConn, ok := iConn.(*internet.TrackedConn); ok {
		iConn = trackedConn.NetConn()
	}
	var readCounter, writeCounter stats.Counter
	if statConn, ok := iConn.(*internet.StatCouterConnection); ok {
		iConn = statConn.Connection
	}

	var packetConn net.PacketConn
	switch iConn := iConn.(type) {
	case *internet.PacketConnWrapper:
		packetConn = iConn.Conn
	case net.PacketConn:
		packetConn = iConn
	default:
		packetConn = internet.NewConnWrapper(iConn)
	}

	quicConn, err := quic.Dial(ctx, packetConn, rawConn.RemoteAddr(),
		tlsSettings.GetTLSConfig(v2tls.WithNextProto("h3"), v2tls.WithDestination(dest)),
		&quic.Config{
			KeepAlivePeriod:      time.Second * 15,
			HandshakeIdleTimeout: time.Second * 8,
		})
	if err != nil {
		rawConn.Close()
		return nil, err
	}

	h3clientConn := c.transport.NewClientConn(quicConn)
	proxyConn, err := connectHTTP3(rawConn, quicConn, h3clientConn, readCounter, writeCounter, nil)
	if err != nil {
		return nil, err
	}

	c.cachedH3Mutex.Lock()
	c.cachedH3Conns.PushFront(&h3Conn{
		rawConn:      rawConn,
		quicConn:     quicConn,
		h3Conn:       h3clientConn,
		readCounter:  readCounter,
		writeCounter: writeCounter,
	})
	c.cachedH3Mutex.Unlock()

	return proxyConn, err
}

type http3Conn struct {
	net.Conn
	in           *io.PipeWriter
	out          io.ReadCloser
	readCounter  stats.Counter
	writeCounter stats.Counter
}

func (c *http3Conn) Read(p []byte) (n int, err error) {
	n, err = c.out.Read(p)
	if c.readCounter != nil {
		c.readCounter.Add(int64(n))
	}
	return n, err
}

func (c *http3Conn) Write(p []byte) (n int, err error) {
	n, err = c.in.Write(p)
	if c.writeCounter != nil {
		c.writeCounter.Add(int64(n))
	}
	return n, err
}

func (c *http3Conn) Close() error {
	c.in.Close()
	return c.out.Close()
}

func init() {
	common.Must(common.RegisterConfig((*ClientConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewClient(ctx, config.(*ClientConfig))
	}))
}
