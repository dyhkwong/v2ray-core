package trusttunnel

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptrace"
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
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal"
	"github.com/v2fly/v2ray-core/v5/common/task"
	"github.com/v2fly/v2ray-core/v5/features/dns"
	"github.com/v2fly/v2ray-core/v5/features/policy"
	"github.com/v2fly/v2ray-core/v5/features/stats"
	"github.com/v2fly/v2ray-core/v5/proxy"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/security"
	v2tls "github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

type Client struct {
	config           *ClientConfig
	serverAddress    net.Destination
	policyManager    policy.Manager
	resolver         func(domain string) (net.Address, error)
	roundTripper     http.RoundTripper
	roundTripperLock sync.Mutex
	resetAt          time.Time
}

func (c *Client) InterfaceUpdate() {
	_ = c.Close()
}

func (c *Client) Close() error {
	c.roundTripperLock.Lock()
	if transport, ok := c.roundTripper.(interface{ CloseIdleConnections() }); ok {
		transport.CloseIdleConnections()
	}
	c.roundTripper = nil
	c.resetAt = time.Now()
	c.roundTripperLock.Unlock()
	return nil
}

// NewClient create a new http client based on the given config.
func NewClient(ctx context.Context, config *ClientConfig) (*Client, error) {
	if len(config.Username) == 0 {
		return nil, newError("username not set")
	}
	if len(config.Password) == 0 {
		return nil, newError("password not set")
	}
	serverAddress := net.Destination{
		Address: config.Address.AsAddress(),
		Port:    net.Port(config.Port),
		Network: net.Network_TCP,
	}
	if config.Http3 {
		serverAddress.Network = net.Network_UDP
	}
	v := core.MustFromContext(ctx)
	client := &Client{
		config:        config,
		serverAddress: serverAddress,
		policyManager: v.GetFeature(policy.ManagerType()).(policy.Manager),
	}
	dnsClient := v.GetFeature(dns.ClientType()).(dns.Client)
	client.resolver = func(domain string) (net.Address, error) {
		ips, err := dns.LookupIPWithOption(dnsClient, domain, dns.IPOption{
			IPv4Enable: config.DomainStrategy != ClientConfig_USE_IP6,
			IPv6Enable: config.DomainStrategy != ClientConfig_USE_IP4,
		})
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, dns.ErrEmptyResponse
		}
		if config.DomainStrategy == ClientConfig_PREFER_IP4 || config.DomainStrategy == ClientConfig_PREFER_IP6 {
			var addr net.Address
			for _, ip := range ips {
				addr = net.IPAddress(ip)
				if addr.Family().IsIPv4() == (config.DomainStrategy == ClientConfig_PREFER_IP4) {
					return addr, nil
				}
			}
		}
		return net.IPAddress(ips[0]), nil
	}
	return client, nil
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
		targetAddr = uotMagicAddress
	}

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

	conn, httpVersion, err := c.setupHTTPTunnel(ctx, targetAddr, dialer, firstPayload)
	if err != nil {
		return newError("failed to find an available destination").Base(err)
	}
	defer conn.Close()

	newError("tunneling request to ", target, " via ", c.serverAddress.NetAddr()).WriteToLog(session.ExportIDToError(ctx))

	var targetIP net.Address
	if target.Network == net.Network_UDP && target.Address.Family().IsDomain() {
		if ip, err := c.resolver(target.Address.Domain()); err != nil {
			return err
		} else {
			targetIP = ip
		}
	} else {
		targetIP = target.Address
	}

	p := c.policyManager.ForLevel(c.config.Level)

	ctx, cancel := context.WithCancel(ctx)
	timer := signal.CancelAfterInactivity(ctx, cancel, p.Timeouts.ConnectionIdle)

	requestFunc := func() error {
		defer timer.SetTimeout(p.Timeouts.DownlinkOnly)
		if target.Network == net.Network_UDP {
			var userAgent string
			switch httpVersion {
			case httpVersion1:
				userAgent = defaultH1UserAgent
			case httpVersion2:
				userAgent = defaultH2UserAgent
			case httpVersion3:
				userAgent = defaultH3UserAgent
			default:
				panic("invalid")
			}
			return buf.Copy(link.Reader, newUoTWriter(conn, target, targetIP, userAgent, c.resolver), buf.UpdateActivity(timer))
		}
		return buf.Copy(link.Reader, buf.NewWriter(conn), buf.UpdateActivity(timer))
	}
	responseFunc := func() error {
		defer timer.SetTimeout(p.Timeouts.UplinkOnly)
		if target.Network == net.Network_UDP {
			return buf.Copy(newUoTReader(conn, target, targetIP), link.Writer, buf.UpdateActivity(timer))
		}
		return buf.Copy(buf.NewReader(conn), link.Writer, buf.UpdateActivity(timer))
	}

	responseDonePost := task.OnSuccess(responseFunc, task.Close(link.Writer))
	if err := task.Run(ctx, requestFunc, responseDonePost); err != nil {
		return newError("connection ends").Base(err)
	}

	return nil
}

// setupHTTPTunnel will create a socket tunnel via HTTP CONNECT method
func (c *Client) setupHTTPTunnel(ctx context.Context, target string, dialer internet.Dialer, firstPayload []byte) (net.Conn, int, error) {
	handler, ok := dialer.(*outbound.Handler)
	if !ok {
		panic("dialer is not *outbound.Handler")
	}
	if handler.MuxEnabled() {
		return nil, httpVersionUndefined, newError("mux enabled")
	}
	if handler.TransportLayerEnabled() {
		return nil, httpVersionUndefined, newError("transport layer enabled")
	}
	streamSettings := handler.StreamSettings()
	if streamSettings == nil {
		return nil, httpVersionUndefined, newError("tls not enabled")
	}
	if c.config.Http3 {
		if streamSettings.SecurityType != "v2ray.core.transport.internet.tls.Config" {
			return nil, httpVersionUndefined, newError("tls not enabled")
		}
	} else {
		if streamSettings.SecurityType != "v2ray.core.transport.internet.tls.Config" && streamSettings.SecurityType != "v2ray.core.transport.internet.tls.utls.Config" {
			return nil, httpVersionUndefined, newError("tls not enabled")
		}
	}
	if len(c.config.ServerNameToVerify) > 0 {
		ctx = session.ContextWithServerNameToVerify(ctx, c.config.ServerNameToVerify)
	}

	c.roundTripperLock.Lock()
	roundTripper := c.roundTripper
	if roundTripper == nil {
		if c.config.Http3 {
			roundTripper = &http3.Transport{
				QUICConfig: &quic.Config{
					KeepAlivePeriod: time.Second * 10,
				},
				Dial: func(_ context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (*quic.Conn, error) {
					detachedCtx := core.ToBackgroundDetachedContext(ctx)
					rawConn, err := dialer.Dial(detachedCtx, c.serverAddress)
					if err != nil {
						return nil, err
					}
					var readCounter, writeCounter stats.Counter
					iConn := rawConn
					if trackedConn, ok := iConn.(*internet.TrackedConn); ok {
						iConn = trackedConn.NetConn()
					}
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
					tlsSettings := streamSettings.SecuritySettings.(*v2tls.Config)
					tlsConfig := tlsSettings.GetTLSConfigWithContext(detachedCtx, v2tls.WithNextProto("h3"), v2tls.WithDestination(c.serverAddress))
					quicConn, err := quic.Dial(detachedCtx, packetConn, rawConn.RemoteAddr(), tlsConfig, cfg)
					if err != nil {
						rawConn.Close()
						return nil, err
					}
					return quicConn, nil
				},
			}
		} else {
			protocols := new(http.Protocols)
			protocols.SetHTTP1(true)
			protocols.SetUnencryptedHTTP2(true)
			roundTripper = &http.Transport{
				Protocols: protocols,
				HTTP2: &http.HTTP2Config{
					SendPingTimeout: time.Second * 10,
				},
				DialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
					detachedCtx := core.ToBackgroundDetachedContext(ctx)
					rawConn, err := dialer.Dial(detachedCtx, c.serverAddress)
					if err != nil {
						return nil, err
					}
					iConn := rawConn
					if trackedConn, ok := iConn.(*internet.TrackedConn); ok {
						iConn = trackedConn.NetConn()
					}
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
		}
		c.roundTripper = roundTripper
	}
	c.roundTripperLock.Unlock()

	var u *url.URL
	if c.config.Http3 {
		u = &url.URL{Host: target, Scheme: "https"}
	} else {
		u = &url.URL{Host: target, Scheme: "http"}
	}
	req := &http.Request{
		Method: http.MethodConnect,
		URL:    u,
		Header: make(http.Header),
		Host:   target,
	}

	auth := c.config.GetUsername() + ":" + c.config.GetPassword()
	req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))

	var pErr error
	var wg sync.WaitGroup
	wg.Add(1)
	var httpVersion int
	pr, pw := io.Pipe()
	req.Body = pr
	if c.config.Http3 {
		httpVersion = httpVersion3
		go func() {
			_, pErr = pw.Write(firstPayload)
			wg.Done()
		}()
	} else {
		req.Header.Set("Proxy-Connection", "Keep-Alive") // Go HTTP/2 will remove this header automatically
		trace := &httptrace.ClientTrace{
			GotConn: func(info httptrace.GotConnInfo) {
				iConn := info.Conn
				if trackedConn, ok := iConn.(*internet.TrackedConn); ok {
					iConn = trackedConn.NetConn()
				}
				if statConn, ok := iConn.(*internet.StatCouterConnection); ok {
					iConn = statConn.Connection
				}
				nextProto := ""
				if connALPNGetter, ok := iConn.(security.ConnectionApplicationProtocol); ok {
					nextProto, _ = connALPNGetter.GetConnectionApplicationProtocol()
				}
				switch nextProto {
				case "", "http/1.1":
					httpVersion = httpVersion1
					go func() {
						wg.Done()
					}()
				case "h2":
					httpVersion = httpVersion2
					go func() {
						_, pErr = pw.Write(firstPayload)
						wg.Done()
					}()
				}
			},
		}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	}
	resp, err := c.roundTripWrapper(roundTripper, req) // nolint: bodyclose
	if err != nil {
		return nil, httpVersion, err
	}
	wg.Wait()
	if pErr != nil {
		return nil, httpVersion, pErr
	}
	if resp.StatusCode != http.StatusOK {
		return nil, httpVersion, newError("Proxy responded with non 200 code: " + resp.Status)
	}
	if httpVersion == httpVersion1 {
		httpConn := &httpConn{in: pw, out: resp.Body}
		if _, err = httpConn.Write(firstPayload); err != nil {
			httpConn.Close()
			return nil, httpVersion, err
		}
		return httpConn, httpVersion, nil
	}
	return &httpConn{in: pw, out: resp.Body}, httpVersion, nil
}

func (c *Client) roundTripWrapper(roundTripper http.RoundTripper, req *http.Request) (*http.Response, error) {
	type result struct {
		resp *http.Response
		err  error
	}
	ch := make(chan result, 1)
	startAt := time.Now()
	go func() {
		// do not use req.WithContext here
		resp, err := roundTripper.RoundTrip(req) // nolint: bodyclose
		ch <- result{
			resp: resp,
			err:  err,
		}
	}()
	select {
	case <-time.After(time.Second * 5):
		c.roundTripperLock.Lock()
		if c.resetAt.Before(startAt) {
			if transport, ok := c.roundTripper.(interface{ CloseIdleConnections() }); ok {
				transport.CloseIdleConnections()
			}
			c.roundTripper = nil
			c.resetAt = time.Now()
		}
		c.roundTripperLock.Unlock()
		return nil, context.DeadlineExceeded
	case result := <-ch:
		return result.resp, result.err
	}
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
