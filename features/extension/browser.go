package extension

import (
	"io"
	"net/http"

	"github.com/gorilla/websocket"
)

type BrowserForwarder interface {
	DialWebsocket(url string, header http.Header) (io.ReadWriteCloser, error)
}

func BrowserForwarderType() interface{} {
	return (*BrowserForwarder)(nil)
}

type BrowserDialer interface {
	DialWS(uri string, earlydata []byte) (*websocket.Conn, error)
	DialGet(uri string, headers http.Header) (*websocket.Conn, error)
	DialPost(uri string, headers http.Header, payload []byte) error
}

func BrowserDialerType() interface{} {
	return (*BrowserDialer)(nil)
}
