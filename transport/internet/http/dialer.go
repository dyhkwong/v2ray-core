package http

import (
	"context"
	gotls "crypto/tls"
	gonet "net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/net/cnc"
	"github.com/v2fly/v2ray-core/v5/common/uuid"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/reality"
	"github.com/v2fly/v2ray-core/v5/transport/internet/security"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls/utls"
	"github.com/v2fly/v2ray-core/v5/transport/pipe"
)

type dialerConf struct {
	net.Destination
	*internet.MemoryStreamConfig
}

var (
	globalDialerMap    map[dialerConf]*http.Client
	globalDialerAccess sync.Mutex
)

type dialerCanceller func()

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

func (c *connWrapper) LocalAddr() net.Addr {
	return c.localAddr
}

func NewConnWrapper(conn net.Conn) net.PacketConn {
	// https://github.com/quic-go/quic-go/commit/8189e75be6121fdc31dc1d6085f17015e9154667#diff-4c6aaadced390f3ce9bec0a9c9bb5203d5fa85df79023e3e0eec423dc9baa946R48-R62
	uuid := uuid.New()
	return &connWrapper{
		Conn:      conn,
		localAddr: &net.UnixAddr{Name: uuid.String()},
	}
}

func getHTTPClient(ctx context.Context, dest net.Destination, securityEngine *security.Engine, streamSettings *internet.MemoryStreamConfig) (*http.Client, dialerCanceller) {
	globalDialerAccess.Lock()
	defer globalDialerAccess.Unlock()

	canceller := func() {
		globalDialerAccess.Lock()
		defer globalDialerAccess.Unlock()
		delete(globalDialerMap, dialerConf{dest, streamSettings})
	}

	if globalDialerMap == nil {
		globalDialerMap = make(map[dialerConf]*http.Client)
	}

	if client, found := globalDialerMap[dialerConf{dest, streamSettings}]; found {
		return client, canceller
	}

	var tlsConfig *tls.Config
	switch cfg := streamSettings.SecuritySettings.(type) {
	case *tls.Config:
		tlsConfig = cfg
	case *utls.Config:
		tlsConfig = cfg.GetTlsConfig()
	}
	isH3 := tlsConfig != nil && len(tlsConfig.NextProtocol) == 1 && tlsConfig.NextProtocol[0] == "h3"

	var transport http.RoundTripper
	transport = &http2.Transport{
		DialTLSContext: func(_ context.Context, network, addr string, tlsConfig *gotls.Config) (gonet.Conn, error) {
			rawHost, rawPort, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			if len(rawPort) == 0 {
				rawPort = "443"
			}
			port, err := net.PortFromString(rawPort)
			if err != nil {
				return nil, err
			}
			address := net.ParseAddress(rawHost)

			detachedContext := core.ToBackgroundDetachedContext(ctx)
			pconn, err := internet.DialSystem(detachedContext, net.TCPDestination(address, port), streamSettings.SocketSettings)
			if err != nil {
				return nil, err
			}

			if realitySettings := reality.ConfigFromStreamSettings(streamSettings); realitySettings != nil {
				return reality.UClient(pconn, realitySettings, detachedContext, dest)
			}

			cn, err := (*securityEngine).Client(pconn,
				security.OptionWithDestination{Dest: dest})
			if err != nil {
				return nil, err
			}

			protocol := ""
			if connAPLNGetter, ok := cn.(security.ConnectionApplicationProtocol); ok {
				connectionALPN, err := connAPLNGetter.GetConnectionApplicationProtocol()
				if err != nil {
					return nil, newError("failed to get connection ALPN").Base(err).AtWarning()
				}
				protocol = connectionALPN
			}

			if protocol != http2.NextProtoTLS {
				return nil, newError("http2: unexpected ALPN protocol " + protocol + "; want q" + http2.NextProtoTLS).AtError()
			}
			return cn, nil
		},
	}

	if isH3 {
		transport = &http3.RoundTripper{
			QUICConfig: &quic.Config{
				MaxIdleTimeout:     300 * time.Second,
				MaxIncomingStreams: -1,
				KeepAlivePeriod:    10 * time.Second,
			},
			TLSClientConfig: tlsConfig.GetTLSConfig(tls.WithDestination(dest)),
			Dial: func(_ context.Context, addr string, tlsCfg *gotls.Config, cfg *quic.Config) (quic.EarlyConnection, error) {
				detachedContext := core.ToBackgroundDetachedContext(ctx)
				conn, err := internet.DialSystem(detachedContext, dest, streamSettings.SocketSettings)
				if err != nil {
					return nil, err
				}
				var packetConn net.PacketConn
				switch c := conn.(type) {
				case *internet.PacketConnWrapper:
					packetConn = c.Conn
				case net.PacketConn:
					packetConn = c
				default:
					packetConn = NewConnWrapper(conn)
				}
				tr := quic.Transport{
					Conn: packetConn,
				}
				return tr.DialEarly(detachedContext, conn.RemoteAddr(), tlsCfg, cfg)
			},
		}
	}

	client := &http.Client{
		Transport: transport,
	}

	globalDialerMap[dialerConf{dest, streamSettings}] = client
	return client, canceller
}

// Dial dials a new TCP connection to the given destination.
func Dial(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (internet.Connection, error) {
	httpSettings := streamSettings.ProtocolSettings.(*Config)

	var tlsConfig *tls.Config
	switch cfg := streamSettings.SecuritySettings.(type) {
	case *tls.Config:
		tlsConfig = cfg
	case *utls.Config:
		tlsConfig = cfg.GetTlsConfig()
	}
	isH3 := tlsConfig != nil && len(tlsConfig.NextProtocol) == 1 && tlsConfig.NextProtocol[0] == "h3"
	if isH3 {
		dest.Network = net.Network_UDP
	}

	securityEngine, _ := security.CreateSecurityEngineFromSettings(ctx, streamSettings)
	realityConfig := reality.ConfigFromStreamSettings(streamSettings)
	if securityEngine == nil && realityConfig == nil {
		return nil, newError("TLS or REALITY must be enabled for http transport.").AtWarning()
	}
	client, canceller := getHTTPClient(ctx, dest, &securityEngine, streamSettings)

	opts := pipe.OptionsFromContext(ctx)
	preader, pwriter := pipe.New(opts...)
	breader := &buf.BufferedReader{Reader: preader}

	httpMethod := "PUT"
	if httpSettings.Method != "" {
		httpMethod = httpSettings.Method
	}

	httpHeaders := make(http.Header)

	for _, httpHeader := range httpSettings.Header {
		for _, httpHeaderValue := range httpHeader.Value {
			httpHeaders.Set(httpHeader.Name, httpHeaderValue)
		}
	}

	request := &http.Request{
		Method: httpMethod,
		Host:   httpSettings.getRandomHost(),
		Body:   breader,
		URL: &url.URL{
			Scheme: "https",
			Host:   dest.NetAddr(),
			Path:   httpSettings.getNormalizedPath(),
		},
		Header: httpHeaders,
	}

	if !isH3 {
		request.Proto = "HTTP/2"
		request.ProtoMajor = 2
		request.ProtoMinor = 0
	}

	// Disable any compression method from server.
	request.Header.Set("Accept-Encoding", "identity")

	response, err := client.Do(request) // nolint: bodyclose
	if err != nil {
		canceller()
		return nil, newError("failed to dial to ", dest).Base(err).AtWarning()
	}
	if response.StatusCode != 200 {
		return nil, newError("unexpected status", response.StatusCode).AtWarning()
	}

	bwriter := buf.NewBufferedWriter(pwriter)
	common.Must(bwriter.SetBuffered(false))
	return cnc.NewConnection(
		cnc.ConnectionOutput(response.Body),
		cnc.ConnectionInput(bwriter),
		cnc.ConnectionOnClose(common.ChainedClosable{breader, bwriter, response.Body}),
	), nil
}

func init() {
	common.Must(internet.RegisterTransportDialer(protocolName, Dial))
}
