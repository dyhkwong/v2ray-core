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

func (c *BrowserDialerClient) OpenStream(ctx context.Context, url, _ string, body io.Reader, uploadOnly bool) (io.ReadCloser, net.Addr, net.Addr, error) {
	if body != nil {
		return nil, nil, nil, newError("bidirectional streaming for browser dialer not implemented yet")
	}

	header, err := c.transportConfig.GetRequestHeader()
	if err != nil {
		return nil, nil, nil, err
	}

	xPaddingConfig := &XPaddingConfig{
		Length: int(c.transportConfig.GetNormalizedXPaddingBytes().rand()),
	}

	if c.transportConfig.XPaddingObfsMode {
		xPaddingConfig.Placement = XPaddingPlacement{
			Placement: c.transportConfig.XPaddingPlacement,
			Key:       c.transportConfig.XPaddingKey,
			Header:    c.transportConfig.XPaddingHeader,
			RawURL:    url,
		}
		xPaddingConfig.Method = PaddingMethod(c.transportConfig.XPaddingMethod)
	} else {
		xPaddingConfig.Placement = XPaddingPlacement{
			Placement: PlacementQueryInHeader,
			Key:       "x_padding",
			Header:    "Referer",
			RawURL:    url,
		}
	}

	c.transportConfig.ApplyXPaddingToHeader(header, xPaddingConfig)

	conn, err := c.dialer.DialGet(url, header)
	if err != nil {
		return nil, nil, nil, err
	}

	return newConnection(conn), conn.RemoteAddr(), conn.LocalAddr(), nil
}

func (c *BrowserDialerClient) PostPacket(ctx context.Context, url, _, _ string, body io.Reader, contentLength int64) error {
	bytes, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	header, err := c.transportConfig.GetRequestHeader()
	if err != nil {
		return err
	}

	xPaddingConfig := &XPaddingConfig{
		Length: int(c.transportConfig.GetNormalizedXPaddingBytes().rand()),
	}

	if c.transportConfig.XPaddingObfsMode {
		xPaddingConfig.Placement = XPaddingPlacement{
			Placement: c.transportConfig.XPaddingPlacement,
			Key:       c.transportConfig.XPaddingKey,
			Header:    c.transportConfig.XPaddingHeader,
			RawURL:    url,
		}
		xPaddingConfig.Method = PaddingMethod(c.transportConfig.XPaddingMethod)
	} else {
		xPaddingConfig.Placement = XPaddingPlacement{
			Placement: PlacementQueryInHeader,
			Key:       "x_padding",
			Header:    "Referer",
			RawURL:    url,
		}
	}

	c.transportConfig.ApplyXPaddingToHeader(header, xPaddingConfig)

	return c.dialer.DialPost(url, header, bytes)
}
