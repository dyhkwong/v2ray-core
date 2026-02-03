package splithttp

import (
	"context"
	"io"
	"net/url"

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

func (c *BrowserDialerClient) OpenStream(ctx context.Context, rawURL, sessionId string, body io.Reader, uploadOnly bool) (io.ReadCloser, net.Addr, net.Addr, error) {
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
			RawURL:    rawURL,
		}
		xPaddingConfig.Method = PaddingMethod(c.transportConfig.XPaddingMethod)
	} else {
		xPaddingConfig.Placement = XPaddingPlacement{
			Placement: PlacementQueryInHeader,
			Key:       "x_padding",
			Header:    "Referer",
			RawURL:    rawURL,
		}
	}

	c.transportConfig.ApplyXPaddingToHeader(header, xPaddingConfig)

	url, _ := url.Parse(rawURL)

	sessionPlacement := c.transportConfig.GetNormalizedSessionPlacement()
	sessionKey := c.transportConfig.GetNormalizedSessionKey()

	if sessionId != "" {
		switch sessionPlacement {
		case PlacementPath:
			url.Path = appendToPath(url.Path, sessionId)
		case PlacementQuery:
			q := url.Query()
			q.Set(sessionKey, sessionId)
			url.RawQuery = q.Encode()
		case PlacementHeader:
			header.Set(sessionKey, sessionId)
		}
	}

	conn, err := c.dialer.DialGet(url.String(), header)
	if err != nil {
		return nil, nil, nil, err
	}

	return newConnection(conn), conn.RemoteAddr(), conn.LocalAddr(), nil
}

func (c *BrowserDialerClient) PostPacket(ctx context.Context, rawURL, sessionId, seqStr string, body io.Reader, contentLength int64) error {
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
			RawURL:    rawURL,
		}
		xPaddingConfig.Method = PaddingMethod(c.transportConfig.XPaddingMethod)
	} else {
		xPaddingConfig.Placement = XPaddingPlacement{
			Placement: PlacementQueryInHeader,
			Key:       "x_padding",
			Header:    "Referer",
			RawURL:    rawURL,
		}
	}

	c.transportConfig.ApplyXPaddingToHeader(header, xPaddingConfig)

	url, _ := url.Parse(rawURL)

	sessionPlacement := c.transportConfig.GetNormalizedSessionPlacement()
	seqPlacement := c.transportConfig.GetNormalizedSeqPlacement()
	sessionKey := c.transportConfig.GetNormalizedSessionKey()
	seqKey := c.transportConfig.GetNormalizedSeqKey()
	if sessionId != "" {
		switch sessionPlacement {
		case PlacementPath:
			url.Path = appendToPath(url.Path, sessionId)
		case PlacementQuery:
			q := url.Query()
			q.Set(sessionKey, sessionId)
			url.RawQuery = q.Encode()
		case PlacementHeader:
			header.Set(sessionKey, sessionId)
		}
	}
	if seqStr != "" {
		switch seqPlacement {
		case PlacementPath:
			url.Path = appendToPath(url.Path, seqStr)
		case PlacementQuery:
			q := url.Query()
			q.Set(seqKey, seqStr)
			url.RawQuery = q.Encode()
		case PlacementHeader:
			header.Set(seqKey, seqStr)
		}
	}

	return c.dialer.DialPost(url.String(), header, bytes)
}
