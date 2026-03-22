package splithttp

import (
	"context"
	"io"
	"net/http"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/features/extension"
)

type BrowserDialerClient struct {
	dialer          extension.BrowserDialer
	transportConfig *Config
}

func (c *BrowserDialerClient) IsClosed() bool {
	panic("not implemented yet")
}

func (c *BrowserDialerClient) OpenStream(ctx context.Context, url, sessionId string, body io.Reader, uploadOnly bool) (io.ReadCloser, net.Addr, net.Addr, error) {
	if body != nil {
		return nil, nil, nil, newError("bidirectional streaming for browser dialer not implemented yet")
	}

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	c.transportConfig.FillStreamRequest(request, sessionId, "")

	conn, err := c.dialer.DialGet(request.URL.String(), request.Header, request.Cookies())
	if err != nil {
		return nil, nil, nil, err
	}

	return newConnection(conn), conn.RemoteAddr(), conn.LocalAddr(), nil
}

func (c *BrowserDialerClient) PostPacket(ctx context.Context, url, sessionId, seqStr string, payload buf.MultiBuffer) error {
	method := c.transportConfig.GetNormalizedUplinkHTTPMethod()
	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		return err
	}

	err = c.transportConfig.FillPacketRequest(request, sessionId, seqStr, payload)
	if err != nil {
		return err
	}

	var bytes []byte
	if request.Body != nil {
		bytes, err = io.ReadAll(request.Body)
		if err != nil {
			return err
		}
	}

	return c.dialer.DialPost(method, request.URL.String(), request.Header, request.Cookies(), bytes)
}
