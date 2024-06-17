package splithttp

import (
	"context"
	"io"

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

func (c *BrowserDialerClient) OpenStream(ctx context.Context, url string, body io.Reader, uploadOnly bool) (io.ReadCloser, net.Addr, net.Addr, error) {
	if body != nil {
		return nil, nil, nil, newError("bidirectional streaming for browser dialer not implemented yet")
	}

	conn, err := c.dialer.DialGet(url, c.transportConfig.GetRequestHeader(url))
	if err != nil {
		return nil, nil, nil, err
	}

	return newConnection(conn), conn.RemoteAddr(), conn.LocalAddr(), nil
}

func (c *BrowserDialerClient) PostPacket(ctx context.Context, url string, body io.Reader, contentLength int64) error {
	bytes, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	err = c.dialer.DialPost(url, c.transportConfig.GetRequestHeader(url), bytes)
	if err != nil {
		return err
	}

	return nil
}
