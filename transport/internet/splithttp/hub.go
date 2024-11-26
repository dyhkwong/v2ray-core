package splithttp

import (
	"context"
	gotls "crypto/tls"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	goreality "github.com/xtls/reality"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/net"
	http_proto "github.com/v2fly/v2ray-core/v5/common/protocol/http"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal/done"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/reality"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

type requestHandler struct {
	config    *Config
	host      string
	path      string
	ln        *Listener
	sessionMu *sync.Mutex
	sessions  sync.Map
}

type httpSession struct {
	uploadQueue *uploadQueue
	// for as long as the GET request is not opened by the client, this will be
	// open ("undone"), and the session may be expired within a certain TTL.
	// after the client connects, this becomes "done" and the session lives as
	// long as the GET request.
	isFullyConnected *done.Instance
}

func (h *requestHandler) maybeReapSession(isFullyConnected *done.Instance, sessionId string) {
	shouldReap := done.New()
	go func() {
		time.Sleep(30 * time.Second)
		shouldReap.Close()
	}()

	select {
	case <-isFullyConnected.Wait():
		return
	case <-shouldReap.Wait():
		h.sessions.Delete(sessionId)
	}
}

func (h *requestHandler) upsertSession(sessionId string) *httpSession {
	// fast path
	currentSessionAny, ok := h.sessions.Load(sessionId)
	if ok {
		return currentSessionAny.(*httpSession)
	}

	// slow path
	h.sessionMu.Lock()
	defer h.sessionMu.Unlock()

	currentSessionAny, ok = h.sessions.Load(sessionId)
	if ok {
		return currentSessionAny.(*httpSession)
	}

	s := &httpSession{
		uploadQueue:      NewUploadQueue(),
		isFullyConnected: done.New(),
	}

	h.sessions.Store(sessionId, s)
	go h.maybeReapSession(s.isFullyConnected, sessionId)
	return s
}

func (h *requestHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if !strings.HasPrefix(request.URL.Path, h.path) {
		newError("failed to validate path, request:", request.URL.Path, ", config:", h.path).WriteToLog()
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	h.config.WriteResponseHeader(writer)

	sessionId := ""
	subpath := strings.Split(request.URL.Path[len(h.path):], "/")
	if len(subpath) > 0 {
		sessionId = subpath[0]
	}

	if sessionId == "" {
		newError("no sessionid on request:", request.URL.Path).WriteToLog()
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	forwardedAddrs := http_proto.ParseXForwardedFor(request.Header)
	remoteAddr, err := net.ResolveTCPAddr("tcp", request.RemoteAddr)
	if err != nil {
		remoteAddr = &net.TCPAddr{}
	}
	if len(forwardedAddrs) > 0 && forwardedAddrs[0].Family().IsIP() {
		remoteAddr = &net.TCPAddr{
			IP:   forwardedAddrs[0].IP(),
			Port: int(0),
		}
	}

	currentSession := h.upsertSession(sessionId)

	if request.Method == "POST" {
		seq := ""
		if len(subpath) > 1 {
			seq = subpath[1]
		}

		if seq == "" {
			err = currentSession.uploadQueue.Push(Packet{
				Reader: request.Body,
			})
			if err != nil {
				newError("failed to upload (PushReader)").Base(err).AtInfo().WriteToLog()
				writer.WriteHeader(http.StatusConflict)
			} else {
				writer.WriteHeader(http.StatusOK)
				<-request.Context().Done()
			}
			return
		}

		payload, err := io.ReadAll(request.Body)

		if err != nil {
			newError("failed to upload (ReadAll)").Base(err).WriteToLog()
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		seqInt, err := strconv.ParseUint(seq, 10, 64)
		if err != nil {
			newError("failed to upload (ParseUint)").Base(err).WriteToLog()
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = currentSession.uploadQueue.Push(Packet{
			Payload: payload,
			Seq:     seqInt,
		})

		if err != nil {
			newError("failed to upload (PushPayload)").Base(err).WriteToLog()
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		writer.WriteHeader(http.StatusOK)
	} else if request.Method == "GET" {
		responseFlusher, ok := writer.(http.Flusher)
		if !ok {
			panic("expected http.ResponseWriter to be an http.Flusher")
		}

		// after GET is done, the connection is finished. disable automatic
		// session reaping, and handle it in defer
		currentSession.isFullyConnected.Close()
		defer h.sessions.Delete(sessionId)

		// magic header instructs nginx + apache to not buffer response body
		writer.Header().Set("X-Accel-Buffering", "no")
		// A web-compliant header telling all middleboxes to disable caching.
		// Should be able to prevent overloading the cache, or stop CDNs from
		// teeing the response stream into their cache, causing slowdowns.
		writer.Header().Set("Cache-Control", "no-store")
		// magic header to make the HTTP middle box consider this as SSE to disable buffer
		writer.Header().Set("Content-Type", "text/event-stream")

		writer.WriteHeader(http.StatusOK)

		responseFlusher.Flush()

		downloadDone := done.New()

		conn := splitConn{
			writer: &httpResponseBodyWriter{
				responseWriter:  writer,
				downloadDone:    downloadDone,
				responseFlusher: responseFlusher,
			},
			reader:     currentSession.uploadQueue,
			remoteAddr: remoteAddr,
		}

		h.ln.addConn(internet.Connection(&conn))

		// "A ResponseWriter may not be used after [Handler.ServeHTTP] has returned."
		select {
		case <-request.Context().Done():
		case <-downloadDone.Wait():
		}

		conn.Close()
	} else {
		newError("unsupported method: ", request.Method).AtInfo().WriteToLog()
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

type httpResponseBodyWriter struct {
	sync.Mutex
	responseWriter  http.ResponseWriter
	responseFlusher http.Flusher
	downloadDone    *done.Instance
}

func (c *httpResponseBodyWriter) Write(b []byte) (int, error) {
	c.Lock()
	defer c.Unlock()
	if c.downloadDone.Done() {
		return 0, io.ErrClosedPipe
	}
	n, err := c.responseWriter.Write(b)
	if err == nil {
		c.responseFlusher.Flush()
	}
	return n, err
}

func (c *httpResponseBodyWriter) Close() error {
	c.Lock()
	defer c.Unlock()
	c.downloadDone.Close()
	return nil
}

type Listener struct {
	sync.Mutex
	server     http.Server
	h3server   *http3.Server
	listener   net.Listener
	h3listener *quic.EarlyListener
	config     *Config
	addConn    internet.ConnHandler
}

func ListenSH(ctx context.Context, address net.Address, port net.Port, streamSettings *internet.MemoryStreamConfig, addConn internet.ConnHandler) (internet.Listener, error) {
	l := &Listener{
		addConn: addConn,
	}
	shSettings := streamSettings.ProtocolSettings.(*Config)
	l.config = shSettings
	var listener net.Listener
	var err error
	handler := &requestHandler{
		config:    shSettings,
		host:      shSettings.Host,
		path:      shSettings.GetNormalizedPath(),
		ln:        l,
		sessionMu: &sync.Mutex{},
		sessions:  sync.Map{},
	}

	tlsConfig := tls.ConfigFromStreamSettings(streamSettings)
	realityConfig := reality.ConfigFromStreamSettings(streamSettings)

	if tlsConfig != nil && len(tlsConfig.NextProtocol) == 1 && tlsConfig.NextProtocol[0] == "h3" { // quic
		conn, err := internet.ListenSystemPacket(context.Background(), &net.UDPAddr{
			IP:   address.IP(),
			Port: int(port),
		}, streamSettings.SocketSettings)
		if err != nil {
			return nil, newError("failed to listen UDP (for SH3) on ", address).Base(err)
		}
		l.h3listener, err = quic.ListenEarly(conn, tlsConfig.GetTLSConfig(), nil)
		if err != nil {
			return nil, newError("failed to listen QUIC (for SH3) on ", address, ":", port).Base(err)
		}
		l.h3server = &http3.Server{
			Handler: handler,
		}
		go func() {
			if err := l.h3server.ServeListener(l.h3listener); err != nil {
				newError("failed to serve http3 for splithttp").Base(err).AtWarning().WriteToLog(session.ExportIDToError(ctx))
			}
		}()
		newError("listening QUIC (for SH3) on ", address, ":", port).WriteToLog(session.ExportIDToError(ctx))
		return l, err
	}

	if port == net.Port(0) { // unix
		listener, err = internet.ListenSystem(ctx, &net.UnixAddr{
			Name: address.Domain(),
			Net:  "unix",
		}, streamSettings.SocketSettings)
		if err != nil {
			return nil, newError("failed to listen unix domain socket (for SH) on ", address).Base(err)
		}
		newError("listening unix domain socket (for SH) on ", address).WriteToLog(session.ExportIDToError(ctx))
	} else { // tcp
		listener, err = internet.ListenSystem(ctx, &net.TCPAddr{
			IP:   address.IP(),
			Port: int(port),
		}, streamSettings.SocketSettings)
		if err != nil {
			return nil, newError("failed to listen TCP (for SH) on ", address, ":", port).Base(err)
		}
		newError("listening TCP (for SH) on ", address, ":", port).WriteToLog(session.ExportIDToError(ctx))
	}

	if tlsConfig != nil {
		listener = gotls.NewListener(listener, tlsConfig.GetTLSConfig())
	}
	if realityConfig != nil {
		listener = goreality.NewListener(listener, realityConfig.GetREALITYConfig())
	}

	// h2cHandler can handle both plaintext HTTP/1.1 and h2c
	h2cHandler := h2c.NewHandler(handler, &http2.Server{})
	l.listener = listener
	l.server = http.Server{
		Handler:           h2cHandler,
		ReadHeaderTimeout: time.Second * 4,
		MaxHeaderBytes:    8192,
	}

	go func() {
		if err := l.server.Serve(l.listener); err != nil {
			newError("failed to serve http for splithttp").Base(err).AtWarning().WriteToLog(session.ExportIDToError(ctx))
		}
	}()

	return l, err
}

// Addr implements net.Listener.Addr().
func (ln *Listener) Addr() net.Addr {
	if ln.h3listener != nil {
		return ln.h3listener.Addr()
	}
	return ln.listener.Addr()
}

// Close implements net.Listener.Close().
func (ln *Listener) Close() error {
	if ln.h3server != nil {
		return ln.h3server.Close()
	}
	return ln.listener.Close()
}

func init() {
	common.Must(internet.RegisterTransportListener(protocolName, ListenSH))
}
