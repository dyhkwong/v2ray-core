package splithttp

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	gonet "net"
	"net/http"
	"net/http/httptrace"
	"sync"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal/done"
)

type DialerClient interface {
	IsClosed() bool

	// ctx, url, sessionId, body, uploadOnly
	OpenStream(context.Context, string, string, io.Reader, bool) (io.ReadCloser, net.Addr, net.Addr, error)

	// ctx, url, sessionId, seqStr, body, contentLength
	PostPacket(context.Context, string, string, string, io.Reader, int64) error
}

// implements splithttp.DialerClient in terms of direct network connections
type DefaultDialerClient struct {
	transportConfig *Config
	client          *http.Client
	closed          bool
	httpVersion     string
	// pool of net.Conn, created using dialUploadConn
	uploadRawPool  *sync.Pool
	dialUploadConn func(ctxInner context.Context) (net.Conn, error)
}

func (c *DefaultDialerClient) IsClosed() bool {
	return c.closed
}

func (c *DefaultDialerClient) OpenStream(ctx context.Context, url, sessionId string, body io.Reader, uploadOnly bool) (wrc io.ReadCloser, remoteAddr, localAddr gonet.Addr, err error) {
	// this is done when the TCP/UDP connection to the server was established,
	// and we can unblock the Dial function and print correct net addresses in
	// logs
	gotConn := done.New()
	ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GotConn: func(connInfo httptrace.GotConnInfo) {
			remoteAddr = connInfo.Conn.RemoteAddr()
			localAddr = connInfo.Conn.LocalAddr()
			gotConn.Close()
		},
	})

	method := "GET" // stream-down
	if body != nil {
		method = c.transportConfig.GetNormalizedUplinkHTTPMethod() // stream-up/one
	}
	req, err := http.NewRequestWithContext(context.WithoutCancel(ctx), method, url, body)
	if err != nil {
		return wrc, remoteAddr, localAddr, err
	}
	req.Header, err = c.transportConfig.GetRequestHeader()
	if err != nil {
		return wrc, remoteAddr, localAddr, err
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

	c.transportConfig.ApplyXPaddingToRequest(req, xPaddingConfig)
	c.transportConfig.ApplyMetaToRequest(req, sessionId, "")

	if method == c.transportConfig.GetNormalizedUplinkHTTPMethod() && !c.transportConfig.NoGRPCHeader {
		req.Header.Set("Content-Type", "application/grpc")
	}

	wrc = &WaitReadCloser{Wait: make(chan struct{})}
	go func() {
		var resp *http.Response
		resp, err = c.client.Do(req)
		if err != nil {
			if !uploadOnly { // stream-down is enough
				c.client.CloseIdleConnections()
				c.closed = true
				newError("failed to " + method + " " + url).Base(err).AtInfo().WriteToLog(session.ExportIDToError(ctx))
			}
			gotConn.Close()
			wrc.Close()
			return
		}
		if resp.StatusCode != 200 && !uploadOnly {
			newError("unexpected status ", resp.StatusCode).AtInfo().WriteToLog(session.ExportIDToError(ctx))
		}
		if resp.StatusCode != 200 || uploadOnly { // stream-up
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close() // if it is called immediately, the upload will be interrupted also
			wrc.Close()
			err = newError("unexpected status ", resp.StatusCode)
			return
		}
		wrc.(*WaitReadCloser).Set(resp.Body)
	}()

	<-gotConn.Wait()
	return wrc, remoteAddr, localAddr, err
}

func (c *DefaultDialerClient) PostPacket(ctx context.Context, url, sessionId, seqStr string, body io.Reader, contentLength int64) error {
	var encodedData string
	dataPlacement := c.transportConfig.GetNormalizedUplinkDataPlacement()

	if dataPlacement != PlacementBody {
		data, err := io.ReadAll(body)
		if err != nil {
			return err
		}
		encodedData = base64.RawURLEncoding.EncodeToString(data)
		body = nil
		contentLength = 0
	}

	method := c.transportConfig.GetNormalizedUplinkHTTPMethod()
	req, err := http.NewRequestWithContext(context.WithoutCancel(ctx), method, url, body)
	if err != nil {
		return err
	}
	req.ContentLength = contentLength
	req.Header, err = c.transportConfig.GetRequestHeader()
	if err != nil {
		return err
	}

	if dataPlacement != PlacementBody {
		key := c.transportConfig.UplinkDataKey
		chunkSize := int(c.transportConfig.UplinkChunkSize)

		switch dataPlacement {
		case PlacementHeader:
			for i := 0; i < len(encodedData); i += chunkSize {
				end := i + chunkSize
				if end > len(encodedData) {
					end = len(encodedData)
				}
				chunk := encodedData[i:end]
				headerKey := fmt.Sprintf("%s-%d", key, i/chunkSize)
				req.Header.Set(headerKey, chunk)
			}

			req.Header.Set(key+"-Length", fmt.Sprintf("%d", len(encodedData)))
			req.Header.Set(key+"-Upstream", "1")
		case PlacementCookie:
			for i := 0; i < len(encodedData); i += chunkSize {
				end := i + chunkSize
				if end > len(encodedData) {
					end = len(encodedData)
				}
				chunk := encodedData[i:end]
				cookieName := fmt.Sprintf("%s_%d", key, i/chunkSize)
				req.AddCookie(&http.Cookie{Name: cookieName, Value: chunk})
			}

			req.AddCookie(&http.Cookie{Name: key + "_upstream", Value: "1"})
		}
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

	c.transportConfig.ApplyXPaddingToRequest(req, xPaddingConfig)
	c.transportConfig.ApplyMetaToRequest(req, sessionId, seqStr)

	if c.httpVersion != "1.1" {
		resp, err := c.client.Do(req)
		if err != nil {
			c.client.CloseIdleConnections()
			c.closed = true
			return err
		}

		io.Copy(io.Discard, resp.Body)
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return newError("bad status code:", resp.Status)
		}
	} else {
		// stringify the entire HTTP/1.1 request so it can be
		// safely retried. if instead req.Write is called multiple
		// times, the body is already drained after the first
		// request
		requestBuff := new(bytes.Buffer)
		common.Must(req.Write(requestBuff))

		var uploadConn any
		var h1UploadConn *H1Conn

		for {
			uploadConn = c.uploadRawPool.Get()
			newConnection := uploadConn == nil
			if newConnection {
				newConn, err := c.dialUploadConn(context.WithoutCancel(ctx))
				if err != nil {
					return err
				}
				h1UploadConn = NewH1Conn(newConn)
				uploadConn = h1UploadConn
			} else {
				h1UploadConn = uploadConn.(*H1Conn)

				// TODO: Replace 0 here with a config value later
				// Or add some other condition for optimization purposes
				if h1UploadConn.UnreadedResponsesCount > 0 {
					resp, err := http.ReadResponse(h1UploadConn.RespBufReader, req)
					if err != nil {
						c.client.CloseIdleConnections()
						c.closed = true
						return fmt.Errorf("error while reading response: %s", err.Error())
					}
					io.Copy(io.Discard, resp.Body)
					defer resp.Body.Close()
					if resp.StatusCode != 200 {
						return fmt.Errorf("got non-200 error response code: %d", resp.StatusCode)
					}
				}
			}

			_, err := h1UploadConn.Write(requestBuff.Bytes())
			// if the write failed, we try another connection from
			// the pool, until the write on a new connection fails.
			// failed writes to a pooled connection are normal when
			// the connection has been closed in the meantime.
			if err == nil {
				break
			} else if newConnection {
				return err
			}
		}

		c.uploadRawPool.Put(uploadConn)
	}

	return nil
}

type WaitReadCloser struct {
	Wait chan struct{}
	io.ReadCloser
}

func (w *WaitReadCloser) Set(rc io.ReadCloser) {
	w.ReadCloser = rc
	defer func() {
		if recover() != nil {
			rc.Close()
		}
	}()
	close(w.Wait)
}

func (w *WaitReadCloser) Read(b []byte) (int, error) {
	if w.ReadCloser == nil {
		if <-w.Wait; w.ReadCloser == nil {
			return 0, io.ErrClosedPipe
		}
	}
	return w.ReadCloser.Read(b)
}

func (w *WaitReadCloser) Close() error {
	if w.ReadCloser != nil {
		return w.ReadCloser.Close()
	}
	defer func() {
		if recover() != nil && w.ReadCloser != nil {
			w.ReadCloser.Close()
		}
	}()
	close(w.Wait)
	return nil
}
