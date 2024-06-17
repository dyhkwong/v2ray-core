package splithttp

import (
	"context"
	gotls "crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xtls/quic-go"
	"github.com/xtls/quic-go/http3"
	"golang.org/x/net/http2"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal/done"
	"github.com/v2fly/v2ray-core/v5/common/uuid"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/reality"
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

func decideHTTPVersion(tlsConfig *tls.Config, realityConfig *reality.Config) string {
	if realityConfig != nil {
		return "2"
	}
	if tlsConfig == nil {
		return "1.1"
	}
	if len(tlsConfig.NextProtocol) != 1 {
		return "2"
	}
	if tlsConfig.NextProtocol[0] == "http/1.1" {
		return "1.1"
	}
	if tlsConfig.NextProtocol[0] == "h3" {
		return "3"
	}
	return "2"
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

	httpVersion := decideHTTPVersion(tlsConfig, realityConfig)
	if httpVersion == "3" {
		dest.Network = net.Network_UDP // better to keep this line
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

	if httpVersion == "3" {
		transport = &http3.RoundTripper{
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
	} else if httpVersion == "2" {
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
		httpVersion:    httpVersion,
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

	httpVersion := decideHTTPVersion(tlsConfig, realityConfig)
	if httpVersion == "3" {
		dest.Network = net.Network_UDP
	}

	transportConfiguration := streamSettings.ProtocolSettings.(*Config)
	var requestURL url.URL

	if tlsConfig != nil || realityConfig != nil {
		requestURL.Scheme = "https"
	} else {
		requestURL.Scheme = "http"
	}
	requestURL.Host = transportConfiguration.Host
	if requestURL.Host == "" && tlsConfig != nil {
		requestURL.Host = tlsConfig.ServerName
	}
	if requestURL.Host == "" && realityConfig != nil {
		requestURL.Host = realityConfig.ServerName
	}
	if requestURL.Host == "" {
		requestURL.Host = dest.Address.String()
	}

	sessionIdUuid := uuid.New()
	requestURL.Path = transportConfiguration.GetNormalizedPath() + sessionIdUuid.String()
	requestURL.RawQuery = transportConfiguration.GetNormalizedQuery()

	httpClient, err := getHTTPClient(ctx, dest, streamSettings)
	if err != nil {
		return nil, err
	}

	mode := transportConfiguration.Mode
	if mode == "" || mode == "auto" {
		mode = "packet-up"
		if httpVersion == "2" {
			mode = "stream-up"
		}
		if realityConfig != nil {
			mode = "stream-one"
		}
	}

	newError(fmt.Sprintf("XHTTP is dialing to %s, mode %s, HTTP version %s, host %s", dest, mode, httpVersion, requestURL.Host)).AtInfo().WriteToLog(session.ExportIDToError(ctx))

	var closed atomic.Int32

	reader, writer := io.Pipe()
	conn := splitConn{
		writer: writer,
		onClose: func() {
			if closed.Add(1) > 1 {
				return
			}
		},
	}

	if mode == "stream-one" {
		requestURL.Path = transportConfiguration.GetNormalizedPath()
		conn.reader, conn.remoteAddr, conn.localAddr, _ = httpClient.OpenStream(context.WithoutCancel(ctx), requestURL.String(), reader, false)
		return internet.Connection(&conn), nil
	} else { // stream-down
		var err error
		conn.reader, conn.remoteAddr, conn.localAddr, err = httpClient.OpenStream(context.WithoutCancel(ctx), requestURL.String(), nil, false)
		if err != nil { // browser dialer only
			return nil, err
		}
	}
	if mode == "stream-up" {
		httpClient.OpenStream(ctx, requestURL.String(), reader, true)
		return internet.Connection(&conn), nil
	}

	scMaxEachPostBytes := transportConfiguration.GetNormalizedScMaxEachPostBytes()
	scMinPostsIntervalMs := transportConfiguration.GetNormalizedScMinPostsIntervalMs()

	maxUploadSize := int32(scMaxEachPostBytes.rand())
	uploadPipeReader, uploadPipeWriter := pipe.New(pipe.WithSizeLimit(maxUploadSize - buf.Size))

	conn.writer = uploadWriter{
		uploadPipeWriter,
		maxUploadSize,
	}

	go func() {
		var seq int64
		var lastWrite time.Time

		for {
			wroteRequest := done.New()

			ctx := httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
				WroteRequest: func(httptrace.WroteRequestInfo) {
					wroteRequest.Close()
				},
			})

			// this intentionally makes a shallow-copy of the struct so we
			// can reassign Path (potentially concurrently)
			url := requestURL
			url.Path += "/" + strconv.FormatInt(seq, 10)
			// reassign query to get different padding
			url.RawQuery = transportConfiguration.GetNormalizedQuery()

			seq += 1

			if scMinPostsIntervalMs.From > 0 {
				time.Sleep(time.Duration(scMinPostsIntervalMs.rand())*time.Millisecond - time.Since(lastWrite))
			}

			// by offloading the uploads into a buffered pipe, multiple conn.Write
			// calls get automatically batched together into larger POST requests.
			// without batching, bandwidth is extremely limited.
			chunk, err := uploadPipeReader.ReadMultiBuffer()
			if err != nil {
				break
			}

			lastWrite = time.Now()

			go func() {
				err := httpClient.PostPacket(
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
			wroteRequest.Close()
			if _, ok := httpClient.(*DefaultDialerClient); ok {
				<-wroteRequest.Wait()
			}
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
