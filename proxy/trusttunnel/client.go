package trusttunnel

import (
	"bufio"
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
	config         *ClientConfig
	serverAddress  net.Destination
	policyManager  policy.Manager
	resolver       func(domain string) (net.Address, error)
	h2Transport    *http2.Transport
	h3Transport    *http3.Transport
	cacheConnsLock sync.Mutex
	cachedConns    list.List
}

func (c *Client) InterfaceUpdate() {
	_ = c.Close()
}

func (c *Client) Close() error {
	c.cacheConnsLock.Lock()
	for elem := c.cachedConns.Front(); elem != nil; elem = elem.Next() {
		if cachedConn, ok := elem.Value.(*h2Conn); ok {
			_ = cachedConn.h2Conn.Close()
			_ = cachedConn.rawConn.Close()
		}
		if cachedConn, ok := elem.Value.(*h3Conn); ok {
			_ = cachedConn.h3Conn.CloseWithError(0, "")
			_ = cachedConn.quicConn.CloseWithError(0, "")
			_ = cachedConn.rawConn.Close()
		}
		c.cachedConns.Remove(elem)
	}
	c.cacheConnsLock.Unlock()
	return nil
}

type h2Conn struct {
	rawConn net.Conn
	h2Conn  *http2.ClientConn
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
	if config.Http3 {
		client.h3Transport = &http3.Transport{}
	} else {
		client.h2Transport = &http2.Transport{
			ReadIdleTimeout: time.Second * 15,
		}
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
		if mbuf, _ := reader.ReadMultiBufferTimeout(proxy.FirstPayloadTimeout); mbuf != nil {
			mlen := mbuf.Len()
			firstPayload = bytespool.Alloc(mlen)
			mbuf, _ = buf.SplitBytes(mbuf, firstPayload)
			firstPayload = firstPayload[:mlen]

			buf.ReleaseMulti(mbuf)
			defer bytespool.Free(firstPayload)
		}
	}

	conn, firstResp, err := c.setupHTTPTunnel(ctx, targetAddr, dialer, firstPayload)
	if err != nil {
		return newError("failed to find an available destination").Base(err)
	}
	defer conn.Close()
	_, isHTTP2Or3 := conn.(*httpConn)
	if !isHTTP2Or3 {
		if _, err := conn.Write(firstPayload); err != nil {
			conn.Close()
			return err
		}
	}
	if firstResp != nil {
		if err := link.Writer.WriteMultiBuffer(firstResp); err != nil {
			return err
		}
	}

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
			switch {
			case !isHTTP2Or3:
				userAgent = defaultH1UserAgent
			case !c.config.Http3:
				userAgent = defaultH2UserAgent
			case c.config.Http3:
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
func (c *Client) setupHTTPTunnel(ctx context.Context, target string, dialer internet.Dialer, firstPayload []byte) (net.Conn, buf.MultiBuffer, error) {
	handler, ok := dialer.(*outbound.Handler)
	if !ok {
		panic("dialer is not *outbound.Handler")
	}
	if handler.MuxEnabled() {
		return nil, nil, newError("mux enabled")
	}
	if handler.TransportLayerEnabled() {
		return nil, nil, newError("transport layer enabled")
	}
	streamSettings := handler.StreamSettings()
	if streamSettings == nil {
		return nil, nil, newError("tls not enabled")
	}
	if c.config.Http3 {
		if streamSettings.SecurityType != "v2ray.core.transport.internet.tls.Config" {
			return nil, nil, newError("tls not enabled")
		}
	} else {
		if streamSettings.SecurityType != "v2ray.core.transport.internet.tls.Config" && streamSettings.SecurityType != "v2ray.core.transport.internet.tls.utls.Config" {
			return nil, nil, newError("tls not enabled")
		}
	}

	req := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Host: target},
		Header: make(http.Header),
		Host:   target,
	}

	auth := c.config.Username + ":" + c.config.Password
	req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))

	connectHTTP1 := func(rawConn net.Conn) (net.Conn, buf.MultiBuffer, error) {
		req.Header.Set("Proxy-Connection", "Keep-Alive")
		err := req.Write(rawConn)
		if err != nil {
			rawConn.Close()
			return nil, nil, err
		}
		bufferedReader := bufio.NewReader(rawConn)
		resp, err := http.ReadResponse(bufferedReader, req) // nolint: bodyclose
		if err != nil {
			rawConn.Close()
			return nil, nil, err
		}

		if resp.StatusCode != http.StatusOK {
			rawConn.Close()
			return nil, nil, newError("Proxy responded with non 200 code: " + resp.Status)
		}
		if bufferedReader.Buffered() > 0 {
			payload, err := buf.ReadFrom(io.LimitReader(bufferedReader, int64(bufferedReader.Buffered())))
			if err != nil {
				rawConn.Close()
				return nil, nil, newError("unable to drain buffer: ").Base(err)
			}
			return rawConn, payload, nil
		}
		return rawConn, nil, nil
	}

	connectHTTP2 := func(conn *h2Conn, elem *list.Element) (net.Conn, error) {
		pr, pw := io.Pipe()
		req.Body = pr
		var pErr error
		var wg sync.WaitGroup
		wg.Go(func() {
			_, pErr = pw.Write(firstPayload)
		})
		resp, err := conn.h2Conn.RoundTrip(req) // nolint: bodyclose
		if err != nil {
			conn.h2Conn.Close()
			conn.rawConn.Close()
			if elem != nil {
				c.cacheConnsLock.Lock()
				c.cachedConns.Remove(elem)
				c.cacheConnsLock.Unlock()
			}
			return nil, err
		}
		wg.Wait()
		if pErr != nil {
			conn.h2Conn.Close()
			conn.rawConn.Close()
			if elem != nil {
				c.cacheConnsLock.Lock()
				c.cachedConns.Remove(elem)
				c.cacheConnsLock.Unlock()
			}
			return nil, pErr
		}
		if resp.StatusCode != http.StatusOK {
			return nil, newError("Proxy responded with non 200 code: " + resp.Status)
		}
		return &httpConn{
			Conn: conn.rawConn,
			in:   pw,
			out:  resp.Body,
		}, nil
	}

	connectHTTP3 := func(conn *h3Conn, elem *list.Element) (net.Conn, error) {
		pr, pw := io.Pipe()
		req.Body = pr
		var pErr error
		var wg sync.WaitGroup
		wg.Go(func() {
			_, pErr = pw.Write(firstPayload)
		})
		resp, err := conn.h3Conn.RoundTrip(req) // nolint: bodyclose
		if err != nil {
			conn.h3Conn.CloseWithError(0, "")
			conn.quicConn.CloseWithError(0, "")
			conn.rawConn.Close()
			if elem != nil {
				c.cacheConnsLock.Lock()
				c.cachedConns.Remove(elem)
				c.cacheConnsLock.Unlock()
			}
			return nil, err
		}
		wg.Wait()
		if pErr != nil {
			conn.h3Conn.CloseWithError(0, "")
			conn.quicConn.CloseWithError(0, "")
			conn.rawConn.Close()
			if elem != nil {
				c.cacheConnsLock.Lock()
				c.cachedConns.Remove(elem)
				c.cacheConnsLock.Unlock()
			}
			return nil, pErr
		}
		if resp.StatusCode != http.StatusOK {
			return nil, newError("Proxy responded with non 200 code: " + resp.Status)
		}
		return &httpConn{
			Conn:         conn.rawConn,
			in:           pw,
			out:          resp.Body,
			readCounter:  conn.readCounter,
			writeCounter: conn.writeCounter,
		}, nil
	}

	if !c.config.Http3 {
		c.cacheConnsLock.Lock()
		elem := c.cachedConns.Front()
		c.cacheConnsLock.Unlock()
		if elem != nil {
			conn := elem.Value.(*h2Conn)
			if conn.h2Conn.CanTakeNewRequest() {
				proxyConn, err := connectHTTP2(conn, elem)
				if err != nil {
					return nil, nil, err
				}
				return proxyConn, nil, nil
			}
		}
		rawConn, err := dialer.Dial(ctx, c.serverAddress)
		if err != nil {
			return nil, nil, err
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
				return nil, nil, err
			}
		}
		switch nextProto {
		case "", "http/1.1":
			return connectHTTP1(rawConn)
		case "h2":
			h2clientConn, err := c.h2Transport.NewClientConn(rawConn)
			if err != nil {
				rawConn.Close()
				return nil, nil, err
			}
			conn := &h2Conn{
				rawConn: rawConn,
				h2Conn:  h2clientConn,
			}
			proxyConn, err := connectHTTP2(conn, nil)
			if err != nil {
				return nil, nil, err
			}
			c.cacheConnsLock.Lock()
			c.cachedConns.PushFront(conn)
			c.cacheConnsLock.Unlock()
			return proxyConn, nil, err
		default:
			rawConn.Close()
			return nil, nil, newError("negotiated unsupported application layer protocol: " + nextProto)
		}
	} else {
		c.cacheConnsLock.Lock()
		elem := c.cachedConns.Front()
		c.cacheConnsLock.Unlock()
		if elem != nil {
			conn := elem.Value.(*h3Conn)
			select {
			case <-conn.h3Conn.Context().Done():
			default:
				proxyConn, err := connectHTTP3(conn, elem)
				if err != nil {
					return nil, nil, err
				}
				return proxyConn, nil, nil
			}
		}
		rawConn, err := dialer.Dial(ctx, c.serverAddress)
		if err != nil {
			return nil, nil, err
		}
		iConn := rawConn
		if trackedConn, ok := iConn.(*internet.TrackedConn); ok {
			iConn = trackedConn.NetConn()
		}
		var readCounter, writeCounter stats.Counter
		if statConn, ok := iConn.(*internet.StatCouterConnection); ok {
			iConn = statConn.Connection
			readCounter = statConn.ReadCounter
			writeCounter = statConn.WriteCounter
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
		tlsSettings := streamSettings.SecuritySettings.(*v2tls.Config)
		quicConn, err := quic.Dial(ctx, packetConn, rawConn.RemoteAddr(),
			tlsSettings.GetTLSConfigWithContext(ctx, v2tls.WithNextProto("h3"), v2tls.WithDestination(c.serverAddress)),
			&quic.Config{
				KeepAlivePeriod:      time.Second * 15,
				HandshakeIdleTimeout: time.Second * 8,
			})
		if err != nil {
			rawConn.Close()
			return nil, nil, err
		}
		h3clientConn := c.h3Transport.NewClientConn(quicConn)
		conn := &h3Conn{
			rawConn:      rawConn,
			quicConn:     quicConn,
			h3Conn:       h3clientConn,
			readCounter:  readCounter,
			writeCounter: writeCounter,
		}
		proxyConn, err := connectHTTP3(conn, nil)
		if err != nil {
			return nil, nil, err
		}
		c.cacheConnsLock.Lock()
		c.cachedConns.PushFront(conn)
		c.cacheConnsLock.Unlock()
		return proxyConn, nil, nil
	}
}

type httpConn struct {
	net.Conn
	in           *io.PipeWriter
	out          io.ReadCloser
	readCounter  stats.Counter
	writeCounter stats.Counter
}

func (c *httpConn) Read(p []byte) (n int, err error) {
	n, err = c.out.Read(p)
	if c.readCounter != nil {
		c.readCounter.Add(int64(n))
	}
	return n, err
}

func (c *httpConn) Write(p []byte) (n int, err error) {
	n, err = c.in.Write(p)
	if c.writeCounter != nil {
		c.writeCounter.Add(int64(n))
	}
	return n, err
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
