//go:build !confonly

package http

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"sync"
	"time"

	core "github.com/v2fly/v2ray-core/v4"
	"github.com/v2fly/v2ray-core/v4/common"
	"github.com/v2fly/v2ray-core/v4/common/buf"
	"github.com/v2fly/v2ray-core/v4/common/bytespool"
	"github.com/v2fly/v2ray-core/v4/common/net"
	"github.com/v2fly/v2ray-core/v4/common/protocol"
	"github.com/v2fly/v2ray-core/v4/common/retry"
	"github.com/v2fly/v2ray-core/v4/common/session"
	"github.com/v2fly/v2ray-core/v4/common/signal"
	"github.com/v2fly/v2ray-core/v4/common/task"
	"github.com/v2fly/v2ray-core/v4/features/policy"
	"github.com/v2fly/v2ray-core/v4/proxy"
	"github.com/v2fly/v2ray-core/v4/transport"
	"github.com/v2fly/v2ray-core/v4/transport/internet"
	"github.com/v2fly/v2ray-core/v4/transport/internet/security"
)

type Client struct {
	serverPicker  protocol.ServerPicker
	policyManager policy.Manager
	transport     map[net.Destination]*http.Transport
	transportLock sync.Mutex
}

func (c *Client) Close() error {
	c.transportLock.Lock()
	clear(c.transport)
	c.transportLock.Unlock()
	return nil
}

// NewClient create a new http client based on the given config.
func NewClient(ctx context.Context, config *ClientConfig) (*Client, error) {
	serverList := protocol.NewServerList()
	for _, rec := range config.Server {
		s, err := protocol.NewServerSpecFromPB(rec)
		if err != nil {
			return nil, newError("failed to get server spec").Base(err)
		}
		serverList.AddServer(s)
	}
	if serverList.Size() == 0 {
		return nil, newError("0 target server")
	}

	v := core.MustFromContext(ctx)
	return &Client{
		serverPicker:  protocol.NewRoundRobinServerPicker(serverList),
		policyManager: v.GetFeature(policy.ManagerType()).(policy.Manager),
		transport:     make(map[net.Destination]*http.Transport),
	}, nil
}

// Process implements proxy.Outbound.Process. We first create a socket tunnel via HTTP CONNECT method, then redirect all inbound traffic to that tunnel.
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

	var server *protocol.ServerSpec
	var user *protocol.MemoryUser
	var conn internet.Connection

	var firstPayload []byte
	if reader, ok := link.Reader.(buf.TimeoutReader); ok {
		// 0-RTT optimization for HTTP/2: If the payload comes very soon, it can be
		// transmitted together. Note we should not get stuck here, as the payload may
		// not exist (considering to access MySQL database via a HTTP proxy, where the
		// server sends hello to the client first).
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
		server = c.serverPicker.PickServer()
		dest := server.Destination()
		user = server.PickUser()
		httpConn, err := c.setupHTTPTunnel(ctx, dest, targetAddr, user, dialer, firstPayload)
		if err != nil {
			return err
		}
		conn = httpConn
		return nil
	}); err != nil {
		return newError("failed to find an available destination").Base(err)
	}

	newError("tunneling request to ", target, " via ", server.Destination().NetAddr()).WriteToLog(session.ExportIDToError(ctx))

	defer func() {
		if err := conn.Close(); err != nil {
			newError("failed to closed connection").Base(err).WriteToLog(session.ExportIDToError(ctx))
		}
	}()

	p := c.policyManager.ForLevel(0)
	if user != nil {
		p = c.policyManager.ForLevel(user.Level)
	}

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
func (c *Client) setupHTTPTunnel(ctx context.Context, dest net.Destination, target string, user *protocol.MemoryUser, dialer internet.Dialer, firstPayload []byte) (net.Conn, error) {
	c.transportLock.Lock()
	transport := c.transport[dest]
	if transport == nil {
		protocols := new(http.Protocols)
		protocols.SetHTTP1(true)
		protocols.SetUnencryptedHTTP2(true)
		transport = &http.Transport{
			Protocols: protocols,
			HTTP2: &http.HTTP2Config{
				SendPingTimeout: time.Second * 10,
			},
			DialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
				detachedCtx := core.ToBackgroundDetachedContext(ctx)
				rawConn, err := dialer.Dial(detachedCtx, dest)
				if err != nil {
					return nil, err
				}
				iConn := rawConn
				if statConn, ok := iConn.(*internet.StatCouterConnection); ok {
					iConn = statConn.Connection
				}
				nextProto := ""
				if connALPNGetter, ok := iConn.(security.ConnectionApplicationProtocol); ok {
					nextProto, err = connALPNGetter.GetConnectionApplicationProtocol()
					if err != nil {
						rawConn.Close()
						return nil, err
					}
				}
				switch nextProto {
				case "", "http/1.1":
					protocols.SetHTTP1(true)
					protocols.SetUnencryptedHTTP2(false)
					return rawConn, nil
				case "h2":
					protocols.SetHTTP1(false)
					protocols.SetUnencryptedHTTP2(true)
					return rawConn, nil
				default:
					rawConn.Close()
					return nil, newError("negotiated unsupported application layer protocol: " + nextProto)
				}
			},
		}
		c.transport[dest] = transport
	}
	c.transportLock.Unlock()

	req := &http.Request{
		Method: http.MethodConnect,
		URL: &url.URL{
			Host:   target,
			Scheme: "http",
		},
		Header: make(http.Header),
		Host:   target,
	}

	if user != nil && user.Account != nil {
		account := user.Account.(*Account)
		username, password, headers := account.GetUsername(), account.GetPassword(), account.GetHeaders()
		auth := username + ":" + password
		req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

	var pErr error
	var wg sync.WaitGroup
	wg.Add(1)
	nextProto := ""
	pr, pw := io.Pipe()
	req.Body = pr
	req.Header.Set("Proxy-Connection", "Keep-Alive") // Go HTTP/2 will remove this header automatically
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			iConn := info.Conn
			if statConn, ok := iConn.(*internet.StatCouterConnection); ok {
				iConn = statConn.Connection
			}
			if connALPNGetter, ok := iConn.(security.ConnectionApplicationProtocol); ok {
				nextProto, _ = connALPNGetter.GetConnectionApplicationProtocol()
			}
			switch nextProto {
			case "", "http/1.1":
				go func() {
					wg.Done()
				}()
			case "h2":
				go func() {
					_, pErr = pw.Write(firstPayload)
					wg.Done()
				}()
			}
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
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
	if nextProto != "h2" {
		httpConn := &httpConn{in: pw, out: resp.Body}
		if _, err = httpConn.Write(firstPayload); err != nil {
			httpConn.Close()
			return nil, err
		}
		return httpConn, nil
	}
	return &httpConn{in: pw, out: resp.Body}, nil
}

type httpConn struct {
	in  *io.PipeWriter
	out io.ReadCloser
}

func (h *httpConn) Read(p []byte) (n int, err error) {
	return h.out.Read(p)
}

func (h *httpConn) Write(p []byte) (n int, err error) {
	return h.in.Write(p)
}

func (c *httpConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{
		IP:   []byte{0, 0, 0, 0},
		Port: 0,
	}
}

func (c *httpConn) LocalAddr() net.Addr {
	return &net.TCPAddr{
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

func (h *httpConn) Close() error {
	h.in.Close()
	return h.out.Close()
}

func init() {
	common.Must(common.RegisterConfig((*ClientConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewClient(ctx, config.(*ClientConfig))
	}))
}
