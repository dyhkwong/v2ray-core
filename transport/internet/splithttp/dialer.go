package splithttp

import (
	"context"
	gotls "crypto/tls"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal/semaphore"
	"github.com/v2fly/v2ray-core/v5/common/uuid"
	"github.com/v2fly/v2ray-core/v5/features/extension"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/reality"
	"github.com/v2fly/v2ray-core/v5/transport/internet/security"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls/utls"
	"github.com/v2fly/v2ray-core/v5/transport/internet/transportcommon"
	"github.com/v2fly/v2ray-core/v5/transport/pipe"
)

const (
	// defines the maximum time an idle TCP session can survive in the tunnel, so
	// it should be consistent across HTTP versions and with other transports.
	connIdleTimeout = 300 * time.Second
	// consistent with quic-go
	h3KeepalivePeriod = 10 * time.Second
	// consistent with chrome
	h2KeepalivePeriod = 45 * time.Second
)

type dialerConf struct {
	net.Destination
	*internet.MemoryStreamConfig
}

var (
	globalDialerMap    map[dialerConf]DialerClient
	globalDialerAccess sync.Mutex
)

func getHTTPClient(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (DialerClient, error) {
	shSettings := streamSettings.ProtocolSettings.(*Config)
	if reality.ConfigFromStreamSettings(streamSettings) == nil && shSettings.UseBrowserForwarding {
		newError("using browser dialer").WriteToLog(session.ExportIDToError(ctx))
		var dialer extension.BrowserDialer
		err := core.RequireFeatures(ctx, func(d extension.BrowserDialer) { dialer = d })
		if err != nil {
			return nil, err
		}
		if dialer == nil {
			return nil, newError("get browser dialer failed")
		}
		return &browserDialerClient{
			dialer: dialer,
		}, nil
	}

	globalDialerAccess.Lock()
	defer globalDialerAccess.Unlock()

	if globalDialerMap == nil {
		globalDialerMap = make(map[dialerConf]DialerClient)
	}

	key := dialerConf{dest, streamSettings}

	client, found := globalDialerMap[key]

	if !found {
		client = createHTTPClient(ctx, dest, streamSettings)
		globalDialerMap[key] = client
	}

	return client, nil
}

func createHTTPClient(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) DialerClient {
	var tlsConfig *tls.Config
	var realityConfig *reality.Config
	switch cfg := streamSettings.SecuritySettings.(type) {
	case *tls.Config:
		tlsConfig = cfg
	case *utls.Config:
		tlsConfig = cfg.GetTlsConfig()
	case *reality.Config:
		realityConfig = cfg
	}

	isH2 := false
	isH3 := false
	if realityConfig != nil {
		isH2 = true
		isH3 = false
	} else if tlsConfig != nil {
		isH3 = len(tlsConfig.NextProtocol) == 1 && tlsConfig.NextProtocol[0] == "h3"
		isH2 = !isH3 && !(len(tlsConfig.NextProtocol) == 1 && tlsConfig.NextProtocol[0] == "http/1.1")
	}

	dialContext := func(_ context.Context) (net.Conn, error) {
		ctx = core.ToBackgroundDetachedContext(ctx)
		if realityConfig != nil {
			conn, err := internet.DialSystem(ctx, dest, streamSettings.SocketSettings)
			if err != nil {
				return nil, err
			}
			return reality.UClient(conn, realityConfig, ctx, dest)
		}
		return transportcommon.DialWithSecuritySettings(ctx, dest, streamSettings)
	}

	var transport http.RoundTripper

	if isH3 {
		transport = &http3.Transport{
			QUICConfig: &quic.Config{
				MaxIdleTimeout: connIdleTimeout,
				// these two are defaults of quic-go/http3. the default of quic-go (no
				// http3) is different, so it is hardcoded here for clarity.
				// https://github.com/quic-go/quic-go/blob/b8ea5c798155950fb5bbfdd06cad1939c9355878/http3/client.go#L36-L39
				MaxIncomingStreams: -1,
				KeepAlivePeriod:    h3KeepalivePeriod,
			},
			TLSClientConfig: tlsConfig.GetTLSConfig(tls.WithDestination(dest)),
			Dial: func(_ context.Context, addr string, tlsCfg *gotls.Config, cfg *quic.Config) (quic.EarlyConnection, error) {
				ctx = core.ToBackgroundDetachedContext(ctx)
				rawConn, err := internet.DialSystem(ctx, dest, streamSettings.SocketSettings)
				if err != nil {
					return nil, err
				}
				return quic.DialEarly(ctx, internet.WrapPacketConn(rawConn), rawConn.RemoteAddr(), tlsCfg, cfg)
			},
		}
	} else if isH2 {
		transport = &http2.Transport{
			DialTLSContext: func(ctxInner context.Context, network string, addr string, cfg *gotls.Config) (net.Conn, error) {
				return dialContext(ctxInner)
			},
			IdleConnTimeout: connIdleTimeout,
			ReadIdleTimeout: h2KeepalivePeriod,
		}
	} else {
		httpDialContext := func(ctxInner context.Context, network string, addr string) (net.Conn, error) {
			return dialContext(ctxInner)
		}

		transport = &http.Transport{
			DialTLSContext:  httpDialContext,
			DialContext:     httpDialContext,
			IdleConnTimeout: connIdleTimeout,
			// chunked transfer download with keepalives is buggy with
			// http.Client and our custom dial context.
			DisableKeepAlives: true,
		}
	}

	client := &DefaultDialerClient{
		transportConfig: streamSettings.ProtocolSettings.(*Config),
		client: &http.Client{
			Transport: transport,
		},
		isH2:           isH2,
		isH3:           isH3,
		uploadRawPool:  &sync.Pool{},
		dialUploadConn: dialContext,
	}

	return client
}

func init() {
	common.Must(internet.RegisterTransportDialer(protocolName, Dial))
}

func Dial(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (internet.Connection, error) {
	var tlsConfig *tls.Config
	var realityConfig *reality.Config
	switch cfg := streamSettings.SecuritySettings.(type) {
	case *tls.Config:
		tlsConfig = cfg
	case *utls.Config:
		tlsConfig = cfg.GetTlsConfig()
	case *reality.Config:
		realityConfig = cfg
	}

	if realityConfig == nil && tlsConfig != nil && len(tlsConfig.NextProtocol) == 1 && tlsConfig.NextProtocol[0] == "h3" {
		dest.Network = net.Network_UDP
	}

	newError("dialing splithttp to ", dest).WriteToLog(session.ExportIDToError(ctx))

	var requestURL url.URL

	transportConfiguration := streamSettings.ProtocolSettings.(*Config)

	if securityEngine, _ := security.CreateSecurityEngineFromSettings(ctx, streamSettings); securityEngine != nil || realityConfig != nil {
		requestURL.Scheme = "https"
	} else {
		requestURL.Scheme = "http"
	}
	requestURL.Host = transportConfiguration.Host
	if requestURL.Host == "" {
		requestURL.Host = dest.NetAddr()
	}

	sessionIdUuid := uuid.New()
	requestURL.Path = transportConfiguration.GetNormalizedPath() + sessionIdUuid.String()
	requestURL.RawQuery = transportConfiguration.GetNormalizedQuery()

	httpClient, err := getHTTPClient(ctx, dest, streamSettings)
	if err != nil {
		return nil, err
	}

	reader, remoteAddr, localAddr, err := httpClient.OpenDownload(context.WithoutCancel(ctx), requestURL.String())
	if err != nil {
		return nil, err
	}

	closed := false

	conn := splitConn{
		writer:     nil,
		reader:     reader,
		remoteAddr: remoteAddr,
		localAddr:  localAddr,
		onClose: func() {
			if closed {
				return
			}
			closed = true
		},
	}

	mode := transportConfiguration.Mode
	if mode == "" || mode == "auto" {
		mode = "packet-up"
		if (tlsConfig != nil && len(tlsConfig.NextProtocol) != 1) || realityConfig != nil {
			mode = "stream-up"
		}
	}
	if mode == "stream-up" {
		conn.writer = httpClient.OpenUpload(ctx, requestURL.String())
		return internet.Connection(&conn), nil
	}

	uploadPipeReader, uploadPipeWriter := pipe.New(pipe.WithSizeLimit(scMaxEachPostBytes - buf.Size))

	conn.writer = uploadWriter{
		uploadPipeWriter,
		scMaxEachPostBytes,
	}

	go func() {
		requestsLimiter := semaphore.New(scMaxConcurrentPosts)
		var requestCounter int64

		lastWrite := time.Now()

		// by offloading the uploads into a buffered pipe, multiple conn.Write
		// calls get automatically batched together into larger POST requests.
		// without batching, bandwidth is extremely limited.
		for {
			chunk, err := uploadPipeReader.ReadMultiBuffer()
			if err != nil {
				break
			}

			<-requestsLimiter.Wait()

			seq := requestCounter
			requestCounter += 1

			go func() {
				defer requestsLimiter.Signal()

				// this intentionally makes a shallow-copy of the struct so we
				// can reassign Path (potentially concurrently)
				url := requestURL
				url.Path += "/" + strconv.FormatInt(seq, 10)
				// reassign query to get different padding
				url.RawQuery = transportConfiguration.GetNormalizedQuery()

				err := httpClient.SendUploadRequest(
					context.WithoutCancel(ctx),
					url.String(),
					&buf.MultiBufferContainer{MultiBuffer: chunk},
					int64(chunk.Len()),
				)
				if err != nil {
					newError("failed to send upload").Base(err).WriteToLog(session.ExportIDToError(ctx))
					uploadPipeReader.Interrupt()
				}
			}()

			if time.Since(lastWrite) < time.Duration(scMinPostsIntervalMs)*time.Millisecond {
				time.Sleep(time.Duration(scMinPostsIntervalMs) * time.Millisecond)
			}

			lastWrite = time.Now()
		}
	}()

	return internet.Connection(&conn), nil
}

// A wrapper around pipe that ensures the size limit is exactly honored.
//
// The MultiBuffer pipe accepts any single WriteMultiBuffer call even if that
// single MultiBuffer exceeds the size limit, and then starts blocking on the
// next WriteMultiBuffer call. This means that ReadMultiBuffer can return more
// bytes than the size limit. We work around this by splitting a potentially
// too large write up into multiple.
type uploadWriter struct {
	*pipe.Writer
	maxLen int32
}

func (w uploadWriter) Write(b []byte) (int, error) {
	buffer := buf.New()
	n, err := buffer.Write(b)
	if err != nil {
		return 0, err
	}

	err = w.WriteMultiBuffer([]*buf.Buffer{buffer})
	if err != nil {
		return 0, err
	}
	return n, nil
}
