package browserdialer

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/features/extension"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

func init() {
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewDialer(ctx, config.(*Config)), nil
	}))
}

type Dialer struct {
	ctx        context.Context
	config     *Config
	httpserver *http.Server
}

func NewDialer(ctx context.Context, config *Config) *Dialer {
	return &Dialer{
		ctx:    ctx,
		config: config,
	}
}

//go:embed dialer.html
var webpage []byte

type task struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Extra  any    `json:"extra,omitempty"`
}

var conns = make(chan *websocket.Conn, 256)

var upgrader = &websocket.Upgrader{
	ReadBufferSize:   0,
	WriteBufferSize:  0,
	HandshakeTimeout: time.Second * 4,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (d *Dialer) Type() interface{} {
	return extension.BrowserDialerType()
}

func (d *Dialer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path == "/websocket" {
		if conn, err := upgrader.Upgrade(writer, request, nil); err == nil {
			conns <- conn
		} else {
			newError("browser dialer http upgrade unexpected error")
		}
	} else {
		writer.Write(webpage)
	}
}

func (d *Dialer) Start() error {
	if d.config.ListenAddr != "" {
		d.httpserver = &http.Server{Handler: d}

		var listener net.Listener
		var err error
		address := net.ParseAddress(d.config.ListenAddr)
		switch {
		case address.Family().IsIP():
			listener, err = internet.ListenSystem(d.ctx, &net.TCPAddr{IP: address.IP(), Port: int(d.config.ListenPort)}, nil)
		case strings.EqualFold(address.Domain(), "localhost"):
			listener, err = internet.ListenSystem(d.ctx, &net.TCPAddr{IP: net.IP{127, 0, 0, 1}, Port: int(d.config.ListenPort)}, nil)
		default:
			return newError("browser dialer cannot listen on address: ", address)
		}
		if err != nil {
			return newError("browser dialer cannot listen on port ", d.config.ListenPort).Base(err)
		}

		go func() {
			if err := d.httpserver.Serve(listener); err != nil {
				newError("cannot serve http browser dialer server").Base(err).WriteToLog()
			}
		}()
	}
	return nil
}

func (d *Dialer) Close() error {
	if d.httpserver != nil {
		return d.httpserver.Close()
	}
	return nil
}

type webSocketExtra struct {
	Protocol string `json:"protocol,omitempty"`
}

func (d *Dialer) DialWS(uri string, ed []byte) (*websocket.Conn, error) {
	task := task{
		Method: "WS",
		URL:    uri,
	}
	if ed != nil {
		task.Extra = webSocketExtra{
			Protocol: base64.RawURLEncoding.EncodeToString(ed),
		}
	}
	return d.dialTask(task)
}

type httpExtra struct {
	Referrer string            `json:"referrer,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
}

func httpExtraFromHeaders(headers http.Header) *httpExtra {
	if len(headers) == 0 {
		return nil
	}

	extra := httpExtra{}
	if referrer := headers.Get("Referer"); referrer != "" {
		extra.Referrer = referrer
		headers.Del("Referer")
	}

	if len(headers) > 0 {
		extra.Headers = make(map[string]string)
		for header := range headers {
			extra.Headers[header] = headers.Get(header)
		}
	}

	return &extra
}

func (d *Dialer) DialGet(uri string, headers http.Header) (*websocket.Conn, error) {
	task := task{
		Method: "GET",
		URL:    uri,
		Extra:  httpExtraFromHeaders(headers),
	}

	return d.dialTask(task)
}

func (d *Dialer) DialPost(uri string, headers http.Header, payload []byte) error {
	task := task{
		Method: "POST",
		URL:    uri,
		Extra:  httpExtraFromHeaders(headers),
	}

	conn, err := d.dialTask(task)
	if err != nil {
		return err
	}
	err = conn.WriteMessage(websocket.BinaryMessage, payload)
	if err != nil {
		return err
	}
	err = d.checkOK(conn)
	if err != nil {
		return err
	}

	conn.Close()
	return nil
}

func (d *Dialer) dialTask(task task) (*websocket.Conn, error) {
	data, err := json.Marshal(task)
	if err != nil {
		return nil, err
	}
	var conn *websocket.Conn
	for {
		conn = <-conns
		if conn.WriteMessage(websocket.TextMessage, data) != nil {
			conn.Close()
		} else {
			break
		}
	}
	err = d.checkOK(conn)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (d *Dialer) checkOK(conn *websocket.Conn) error {
	if _, p, err := conn.ReadMessage(); err != nil {
		conn.Close()
		return err
	} else if s := string(p); s != "ok" {
		conn.Close()
		return newError(s)
	}

	return nil
}
