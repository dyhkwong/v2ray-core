package trusttunnel

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
	"golang.org/x/net/http2"

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
	config        *ClientConfig
	serverAddress net.Destination
	policyManager policy.Manager
	resolver      func(domain string) (net.Address, error)
	transportLock sync.Mutex
	transport     http.RoundTripper
}

func (c *Client) InterfaceUpdate() {
	_ = c.Close()
}

func (c *Client) Close() error {
	c.transportLock.Lock()
	if c.transport != nil {
		switch transport := c.transport.(type) {
		case *http3.Transport:
			transport.Close()
		case *http2.Transport:
			closeHTTP2Transport(transport)
		}
		c.transport = nil
	}
	c.transportLock.Unlock()
	return nil
}

func NewClient(ctx context.Context, config *ClientConfig) (*Client, error) {
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

	newError("tunneling request to ", target, " via ", c.serverAddress.NetAddr()).WriteToLog(session.ExportIDToError(ctx))

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

	conn, err := c.setupHTTPTunnel(ctx, target, targetAddr, dialer, firstPayload)
	if err != nil {
		return newError("failed to find an available destination").Base(err)
	}
	defer conn.Close()

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
			var appName string
			if c.config.Http3 {
				appName = defaultH3AppName
			} else {
				appName = defaultH2AppName
			}
			return buf.Copy(link.Reader, newUoTWriter(conn, target, targetIP, appName, c.resolver), buf.UpdateActivity(timer))
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

func (c *Client) setupHTTPTunnel(ctx context.Context, target net.Destination, targetAddr string, dialer internet.Dialer, firstPayload []byte) (net.Conn, error) {
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
	if streamSettings == nil {
		return nil, newError("tls not enabled")
	}
	if c.config.Http3 {
		if streamSettings.SecurityType != "v2ray.core.transport.internet.tls.Config" {
			return nil, newError("tls not enabled")
		}
	} else {
		if streamSettings.SecurityType != "v2ray.core.transport.internet.tls.Config" && streamSettings.SecurityType != "v2ray.core.transport.internet.tls.utls.Config" {
			return nil, newError("tls not enabled")
		}
	}

	c.transportLock.Lock()
	transport := c.transport
	if transport == nil {
		if c.config.Http3 {
			transport = &http3.Transport{
				QUICConfig: &quic.Config{
					KeepAlivePeriod:      time.Second * 15,
					HandshakeIdleTimeout: time.Second * 8,
				},
				Dial: func(_ context.Context, _ string, _ *tls.Config, cfg *quic.Config) (*quic.Conn, error) {
					detachedContext := core.ToBackgroundDetachedContext(ctx)
					tlsSettings := streamSettings.SecuritySettings.(*v2tls.Config)
					tlsCfg, err := tlsSettings.GetTLSConfigWithContext(detachedContext, v2tls.WithNextProto("h3"), v2tls.WithDestination(c.serverAddress))
					conn, err := dialer.Dial(detachedContext, c.serverAddress)
					if err != nil {
						return nil, err
					}
					var readCounter, writeCounter stats.Counter
					iConn := conn
					if statConn, ok := iConn.(*internet.StatCouterConnection); ok {
						iConn = statConn.Connection
						readCounter = statConn.ReadCounter
						writeCounter = statConn.WriteCounter
					}
					var packetConn net.PacketConn
					switch iConn := iConn.(type) {
					case *internet.PacketConnWrapper:
						if readCounter != nil || writeCounter != nil {
							packetConn = newStatCounterConn(iConn.Conn, readCounter, writeCounter)
						} else {
							packetConn = iConn.Conn
						}
					case net.PacketConn:
						if readCounter != nil || writeCounter != nil {
							packetConn = newStatCounterConn(iConn, readCounter, writeCounter)
						} else {
							packetConn = iConn
						}
					default:
						packetConn = internet.NewConnWrapper(iConn)
					}
					quicConn, err := quic.Dial(detachedContext, packetConn, conn.RemoteAddr(), tlsCfg, cfg)
					if err != nil {
						conn.Close()
						return nil, err
					}
					return quicConn, nil
				},
			}
		} else {
			transport = &http2.Transport{
				ReadIdleTimeout: time.Second * 15,
				DialTLSContext: func(_ context.Context, _, _ string, _ *tls.Config) (net.Conn, error) {
					detachedContext := core.ToBackgroundDetachedContext(ctx)
					conn, err := dialer.Dial(detachedContext, c.serverAddress)
					if err != nil {
						return nil, err
					}
					iConn := conn
					if statConn, ok := iConn.(*internet.StatCouterConnection); ok {
						iConn = statConn.Connection
					}
					connALPNGetter, ok := iConn.(security.ConnectionApplicationProtocol)
					if !ok {
						conn.Close()
						return nil, newError("application layer protocol unsupported")
					}
					nextProto, err := connALPNGetter.GetConnectionApplicationProtocol()
					if err != nil {
						conn.Close()
						return nil, err
					}
					if nextProto == "http/1.1" {
						conn.Close()
						return nil, newError("negotiated unsupported application layer protocol: " + nextProto)
					}
					return conn, nil
				},
			}
		}
		c.transport = transport
	}
	c.transportLock.Unlock()

	req := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Scheme: "https", Host: targetAddr},
		Header: make(http.Header),
		Host:   targetAddr,
	}
	req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.config.Username+":"+c.config.Password)))
	var userAgent string
	switch {
	case target.Network == net.Network_UDP:
		userAgent = defaultUoTUserAgent
	case c.config.Http3:
		userAgent = defaultH3UserAgent
	default:
		userAgent = defaultH2UserAgent
	}
	req.Header.Set("User-Agent", userAgent)

	pr, pw := io.Pipe()
	req.Body = pr

	var wg sync.WaitGroup
	var pErr error
	wg.Go(func() {
		_, pErr = pw.Write(firstPayload)
	})

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

	return &httpConn{
		in:  pw,
		out: resp.Body,
	}, nil
}

type httpConn struct {
	in  *io.PipeWriter
	out io.ReadCloser
}

func (c *httpConn) Read(p []byte) (n int, err error) {
	return c.out.Read(p)
}

func (c *httpConn) Write(p []byte) (n int, err error) {
	return c.in.Write(p)
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
	common.Must(common.RegisterConfig((*ClientConfig)(nil), func(ctx context.Context, config any) (any, error) {
		return NewClient(ctx, config.(*ClientConfig))
	}))
}
