package trusttunnel

import (
	"syscall"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/features/stats"
)

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

type conn struct {
	net.PacketConn
	readCounter  stats.Counter
	writeCounter stats.Counter
}

type setBufferConn struct {
	*conn
	setWriteBufferFn interface{ SetWriteBuffer(int) error }
	setReadBufferFn  interface{ SetReadBuffer(int) error }
}

type syscallConn struct {
	*setBufferConn
	syscallConnFn interface {
		SyscallConn() (syscall.RawConn, error)
	}
}

func wrapPacketConn(rawConn net.PacketConn, readCounter, writeCounter stats.Counter) net.PacketConn {
	setWriteBufferFn, canSetWriteBuffer := rawConn.(interface{ SetWriteBuffer(int) error })
	setReadBufferFn, canSetReadBuffer := rawConn.(interface{ SetReadBuffer(int) error })
	syscallConnFn, isSyscallConn := rawConn.(interface {
		SyscallConn() (syscall.RawConn, error)
	})

	conn := &conn{
		PacketConn:   rawConn,
		readCounter:  readCounter,
		writeCounter: writeCounter,
	}

	if canSetWriteBuffer && canSetReadBuffer {
		setBufferConn := &setBufferConn{
			conn:             conn,
			setWriteBufferFn: setWriteBufferFn,
			setReadBufferFn:  setReadBufferFn,
		}
		if isSyscallConn {
			return &syscallConn{
				setBufferConn: setBufferConn,
				syscallConnFn: syscallConnFn,
			}
		}
		return setBufferConn
	}
	return conn
}

func (c *setBufferConn) SetReadBuffer(bytes int) error {
	return c.setReadBufferFn.SetReadBuffer(bytes)
}

func (c *setBufferConn) SetWriteBuffer(bytes int) error {
	return c.setWriteBufferFn.SetWriteBuffer(bytes)
}

func (c *syscallConn) SyscallConn() (syscall.RawConn, error) {
	return c.syscallConnFn.SyscallConn()
}

func (c *conn) ReadFrom(p []byte) (int, net.Addr, error) {
	n, addr, err := c.PacketConn.ReadFrom(p)
	if c.readCounter != nil {
		c.readCounter.Add(int64(n))
	}
	return n, addr, err
}

func (c *conn) WriteTo(p []byte, addr net.Addr) (int, error) {
	n, err := c.PacketConn.WriteTo(p, addr)
	if c.writeCounter != nil {
		c.writeCounter.Add(int64(n))
	}
	return n, err
}
