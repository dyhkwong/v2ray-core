package splithttp

import (
	"context"
	"io"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/features/extension"
)

type browserDialerClient struct {
	dialer extension.BrowserDialer
}

func (c *browserDialerClient) OpenUpload(ctx context.Context, baseURL string) io.WriteCloser {
	panic("not implemented yet")
}

func (c *browserDialerClient) OpenDownload(ctx context.Context, baseURL string) (io.ReadCloser, net.Addr, net.Addr, error) {
	conn, err := c.dialer.DialGet(baseURL)
	if err != nil {
		return nil, nil, nil, err
	}
	return newConnection(conn), conn.RemoteAddr(), conn.LocalAddr(), nil
}

func (c *browserDialerClient) SendUploadRequest(ctx context.Context, url string, payload io.ReadWriteCloser, contentLength int64) error {
	bytes, err := io.ReadAll(payload)
	if err != nil {
		return err
	}
	err = c.dialer.DialPost(url, bytes)
	if err != nil {
		return err
	}
	return nil
}
