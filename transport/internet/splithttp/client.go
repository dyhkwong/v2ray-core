package splithttp

import (
	"bytes"
	"context"
	"io"
	gonet "net"
	"net/http"
	"net/http/httptrace"
	"sync"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/signal/done"
)

type DialerClient interface {
	// (ctx, baseURL, payload) -> err
	// baseURL already contains sessionId and seq
	SendUploadRequest(context.Context, string, io.ReadWriteCloser, int64) error

	// (ctx, baseURL) -> (downloadReader, remoteAddr, localAddr)
	// baseURL already contains sessionId
	OpenDownload(context.Context, string) (io.ReadCloser, net.Addr, net.Addr, error)

	// (ctx, baseURL) -> uploadWriter
	// baseURL already contains sessionId
	OpenUpload(context.Context, string) io.WriteCloser
}

// implements splithttp.DialerClient in terms of direct network connections
type DefaultDialerClient struct {
	transportConfig *Config
	client          *http.Client
	isH2            bool
	isH3            bool
	// pool of net.Conn, created using dialUploadConn
	uploadRawPool  *sync.Pool
	dialUploadConn func(ctxInner context.Context) (net.Conn, error)
}

func (c *DefaultDialerClient) OpenUpload(ctx context.Context, baseURL string) io.WriteCloser {
	reader, writer := io.Pipe()
	req, _ := http.NewRequestWithContext(ctx, "POST", baseURL, reader)
	req.Header = c.transportConfig.GetRequestHeader()
	go c.client.Do(req)
	return writer
}

func (c *DefaultDialerClient) OpenDownload(ctx context.Context, baseURL string) (io.ReadCloser, gonet.Addr, gonet.Addr, error) {
	var remoteAddr gonet.Addr
	var localAddr gonet.Addr
	// this is done when the TCP/UDP connection to the server was established,
	// and we can unblock the Dial function and print correct net addresses in
	// logs
	gotConn := done.New()

	var downResponse io.ReadCloser
	gotDownResponse := done.New()

	ctx, ctxCancel := context.WithCancel(ctx)

	go func() {
		trace := &httptrace.ClientTrace{
			GotConn: func(connInfo httptrace.GotConnInfo) {
				remoteAddr = connInfo.Conn.RemoteAddr()
				localAddr = connInfo.Conn.LocalAddr()
				gotConn.Close()
			},
		}

		// in case we hit an error, we want to unblock this part
		defer gotConn.Close()

		ctx = httptrace.WithClientTrace(ctx, trace)

		req, err := http.NewRequestWithContext(
			ctx,
			"GET",
			baseURL,
			nil,
		)
		if err != nil {
			newError("failed to construct download http request").Base(err).WriteToLog()
			gotDownResponse.Close()
			return
		}

		req.Header = c.transportConfig.GetRequestHeader()

		response, err := c.client.Do(req)
		gotConn.Close()
		if err != nil {
			newError("failed to send download http request").Base(err).WriteToLog()
			gotDownResponse.Close()
			return
		}

		if response.StatusCode != 200 {
			response.Body.Close()
			newError("invalid status code on download: ", response.Status).Base(err).WriteToLog()
			gotDownResponse.Close()
			return
		}

		downResponse = response.Body
		gotDownResponse.Close()
	}()

	if !c.isH3 {
		// in quic-go, sometimes gotConn is never closed for the lifetime of
		// the entire connection, and the download locks up
		// https://github.com/quic-go/quic-go/issues/3342
		// for other HTTP versions, we want to block Dial until we know the
		// remote address of the server, for logging purposes
		<-gotConn.Wait()
	}

	lazyDownload := &LazyReader{
		CreateReader: func() (io.Reader, error) {
			<-gotDownResponse.Wait()
			if downResponse == nil {
				return nil, newError("downResponse failed")
			}
			return downResponse, nil
		},
	}

	// workaround for https://github.com/quic-go/quic-go/issues/2143 --
	// always cancel request context so that Close cancels any Read.
	// Should then match the behavior of http2 and http1.
	reader := downloadBody{
		lazyDownload,
		ctxCancel,
	}

	return reader, remoteAddr, localAddr, nil
}

func (c *DefaultDialerClient) SendUploadRequest(ctx context.Context, url string, payload io.ReadWriteCloser, contentLength int64) error {
	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return err
	}
	req.ContentLength = contentLength
	req.Header = c.transportConfig.GetRequestHeader()

	if c.isH2 || c.isH3 {
		resp, err := c.client.Do(req)
		if err != nil {
			return err
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return newError("bad status code: ", resp.Status)
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
						return newError("error while reading response: ", err.Error())
					}
					if resp.StatusCode != 200 {
						return newError("got non-200 error response code: ", resp.StatusCode)
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

type downloadBody struct {
	io.Reader
	cancel context.CancelFunc
}

func (c downloadBody) Close() error {
	c.cancel()
	return nil
}
