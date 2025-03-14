package internet

import (
	"container/list"
	"net"
	"syscall"

	"golang.org/x/net/ipv4"

	"github.com/v2fly/v2ray-core/v5/common/track"
)

var (
	_ net.Conn       = (*trackedConn)(nil)
	_ net.Conn       = (*trackedPacketConn)(nil)
	_ net.PacketConn = (*trackedPacketConn)(nil)
)

func newTrackedConn(conn net.Conn, pool *track.ConnectionPool) net.Conn {
	if _, ok := conn.(*trackedConn); ok {
		panic("already a trackedConn")
	}
	if _, ok := conn.(*trackedPacketConn); ok {
		panic("already a trackedPacketConn")
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
		return &trackedConn{
			Conn: conn,
			pool: pool,
			elem: elem,
		}
	}
	trackedPacketConn := &trackedPacketConn{
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
		trackedPacketConn: trackedPacketConn,
		setWriteBuffer:    setBufferFn.SetWriteBuffer,
		setReadBuffer:     setBufferFn.SetReadBuffer,
	}
	syscallConnFn, isSyscallConn := packetConn.(interface {
		SyscallConn() (syscall.RawConn, error)
	})
	if !isSyscallConn {
		return setBufferConn
	}
	syscallConn := &syscallConn{
		setBufferConn: setBufferConn,
		syscallConn:   syscallConnFn.SyscallConn,
	}
	oobFn, oobCapable := packetConn.(interface {
		ReadMsgUDP(b, oob []byte) (int, int, int, *net.UDPAddr, error)
		WriteMsgUDP(b, oob []byte, addr *net.UDPAddr) (int, int, error)
	})
	if !oobCapable {
		return syscallConn
	}
	oobConn := &oobConn{
		syscallConn: syscallConn,
		readMsgUDP:  oobFn.ReadMsgUDP,
		writeMsgUDP: oobFn.WriteMsgUDP,
	}
	readBatchFn, canReadBatch := packetConn.(interface {
		ReadBatch(ms []ipv4.Message, flags int) (int, error)
	})
	if canReadBatch {
		oobConn.readBatch = readBatchFn.ReadBatch
	} else {
		oobConn.readBatch = ipv4.NewPacketConn(oobConn).ReadBatch
	}
	return oobConn
}

type trackedConn struct {
	net.Conn
	pool *track.ConnectionPool
	elem *list.Element
}

func (c *trackedConn) Close() error {
	c.pool.Lock()
	c.pool.Remove(c.elem)
	c.pool.Unlock()
	return c.Conn.Close()
}

type trackedPacketConn struct {
	net.PacketConn
	pool       *track.ConnectionPool
	elem       *list.Element
	read       func(b []byte) (int, error)
	write      func(b []byte) (int, error)
	remoteAddr func() net.Addr
}

func (c *trackedPacketConn) Read(b []byte) (int, error) {
	return c.read(b)
}

func (c *trackedPacketConn) Write(b []byte) (int, error) {
	return c.write(b)
}

func (c *trackedPacketConn) RemoteAddr() net.Addr {
	return c.remoteAddr()
}

func (c *trackedPacketConn) Close() error {
	c.pool.Lock()
	c.pool.Remove(c.elem)
	c.pool.Unlock()
	return c.PacketConn.Close()
}

type setBufferConn struct {
	*trackedPacketConn
	setWriteBuffer func(bytes int) error
	setReadBuffer  func(bytes int) error
}

type syscallConn struct {
	*setBufferConn
	syscallConn func() (syscall.RawConn, error)
}

type oobConn struct {
	*syscallConn
	readMsgUDP  func(b, oob []byte) (int, int, int, *net.UDPAddr, error)
	writeMsgUDP func(b, oob []byte, addr *net.UDPAddr) (int, int, error)
	readBatch   func(ms []ipv4.Message, flags int) (int, error)
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

func (c *oobConn) ReadMsgUDP(b, oob []byte) (int, int, int, *net.UDPAddr, error) {
	return c.readMsgUDP(b, oob)
}

func (c *oobConn) WriteMsgUDP(b, oob []byte, addr *net.UDPAddr) (int, int, error) {
	return c.writeMsgUDP(b, oob, addr)
}

func (c *oobConn) ReadBatch(ms []ipv4.Message, flags int) (int, error) {
	return c.readBatch(ms, flags)
}
