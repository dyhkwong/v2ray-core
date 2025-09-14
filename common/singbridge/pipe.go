package singbridge

import (
	"io"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/transport"
)

var _ net.Conn = (*pipeConnWrapper)(nil)

type pipeConnWrapper struct {
	reader io.Reader
	writer buf.Writer
	net.Conn
}

func NewPipeConnWrapper(link *transport.Link) *pipeConnWrapper {
	conn := &pipeConnWrapper{
		writer: link.Writer,
	}
	if ir, ok := link.Reader.(io.Reader); ok {
		conn.reader = ir
	} else {
		conn.reader = &buf.BufferedReader{Reader: link.Reader}
	}
	return conn
}

func (w *pipeConnWrapper) Close() error {
	return nil
}

func (w *pipeConnWrapper) Read(b []byte) (n int, err error) {
	return w.reader.Read(b)
}

func (w *pipeConnWrapper) Write(p []byte) (n int, err error) {
	n = len(p)
	var mb buf.MultiBuffer
	pLen := len(p)
	for pLen > 0 {
		buffer := buf.New()
		if pLen > buf.Size {
			buffer.Write(p[:buf.Size])
			p = p[buf.Size:]
		} else {
			buffer.Write(p)
		}
		pLen -= int(buffer.Len())
		mb = append(mb, buffer)
	}
	err = w.writer.WriteMultiBuffer(mb)
	if err != nil {
		n = 0
		buf.ReleaseMulti(mb)
	}
	return
}
