package internet

import (
	"container/list"
	"net"
	"syscall"

	"github.com/v2fly/v2ray-core/v5/common/track"
)

var (
	_ net.Conn       = (*TrackedConn)(nil)
	_ net.Conn       = (*TrackedPacketConn)(nil)
	_ net.PacketConn = (*TrackedPacketConn)(nil)
)

func newTrackedConn(conn net.Conn, pool *track.ConnectionPool) net.Conn {
	if _, ok := conn.(*TrackedConn); ok {
		panic("already a TrackedConn")
	}
	if _, ok := conn.(*TrackedPacketConn); ok {
		panic("already a TrackedPacketConn")
	}
	pool.Lock()
	elem := pool.PushBack(conn)
	pool.Unlock()

	var packetConn net.PacketConn
	switch conn := conn.(type) {
	case *PacketConnWrapper:
		packetConn = conn.Conn
	case net.PacketConn:
		packetConn = conn
	default:
		return &TrackedConn{
			Conn: conn,
			pool: pool,
			elem: elem,
		}
	}
	trackedPacketConn := &TrackedPacketConn{
		PacketConn: packetConn,
		pool:       pool,
		elem:       elem,
		read:       conn.Read,
		write:      conn.Write,
		remoteAddr: conn.RemoteAddr,
	}
	setBufferFn, canSetBuffer := packetConn.(interface {
		SetWriteBuffer(bytes int) error
		SetReadBuffer(bytes int) error
	})
	if !canSetBuffer {
		return trackedPacketConn
	}
	setBufferConn := &setBufferConn{
		TrackedPacketConn: trackedPacketConn,
		setWriteBuffer:    setBufferFn.SetWriteBuffer,
		setReadBuffer:     setBufferFn.SetReadBuffer,
	}
	syscallConnFn, isSyscallConn := packetConn.(interface {
		SyscallConn() (syscall.RawConn, error)
	})
	if !isSyscallConn {
		return setBufferConn
	}
	return &syscallConn{
		setBufferConn:  setBufferConn,
		setWriteBuffer: setBufferFn.SetWriteBuffer,
		setReadBuffer:  setBufferFn.SetReadBuffer,
		syscallConn:    syscallConnFn.SyscallConn,
	}
}

type TrackedConn struct {
	net.Conn
	pool *track.ConnectionPool
	elem *list.Element
}

func (c *TrackedConn) Close() error {
	c.pool.Lock()
	c.pool.Remove(c.elem)
	c.pool.Unlock()
	return c.Conn.Close()
}

type TrackedPacketConn struct {
	net.PacketConn
	pool       *track.ConnectionPool
	elem       *list.Element
	read       func(b []byte) (n int, err error)
	write      func(b []byte) (n int, err error)
	remoteAddr func() net.Addr
}

func (c *TrackedPacketConn) Read(b []byte) (n int, err error) {
	return c.read(b)
}

func (c *TrackedPacketConn) Write(b []byte) (n int, err error) {
	return c.write(b)
}

func (c *TrackedPacketConn) RemoteAddr() net.Addr {
	return c.remoteAddr()
}

func (c *TrackedPacketConn) Close() error {
	c.pool.Lock()
	c.pool.Remove(c.elem)
	c.pool.Unlock()
	return c.PacketConn.Close()
}

type setBufferConn struct {
	*TrackedPacketConn
	setWriteBuffer func(bytes int) error
	setReadBuffer  func(bytes int) error
}

type syscallConn struct {
	*setBufferConn
	setWriteBuffer func(bytes int) error
	setReadBuffer  func(bytes int) error
	syscallConn    func() (syscall.RawConn, error)
}

func (c *setBufferConn) SetReadBuffer(bytes int) error {
	return c.setReadBuffer(bytes)
}

func (c *setBufferConn) SetWriteBuffer(bytes int) error {
	return c.setWriteBuffer(bytes)
}

func (c *syscallConn) SyscallConn() (syscall.RawConn, error) {
	return c.syscallConn()
}
