package trusttunnel

import (
	"syscall"

	"github.com/quic-go/quic-go"
	"golang.org/x/net/ipv4"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/features/stats"
)

var _ quic.OOBCapablePacketConn = (*oobConn)(nil)

func newStatCounterConn(packetConn net.PacketConn, readCounter, writeCounter stats.Counter) net.PacketConn {
	statCounterConn := &statCounterConn{
		PacketConn:   packetConn,
		readCounter:  readCounter,
		writeCounter: writeCounter,
	}
	setBufferFn, canSetBuffer := packetConn.(interface {
		SetWriteBuffer(bytes int) error
		SetReadBuffer(bytes int) error
	})
	if !canSetBuffer {
		return statCounterConn
	}
	setBufferConn := &setBufferConn{
		statCounterConn: statCounterConn,
		setWriteBuffer:  setBufferFn.SetWriteBuffer,
		setReadBuffer:   setBufferFn.SetReadBuffer,
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
		oobConn.readBatch = ipv4.NewPacketConn(newOOBConnWrapper(oobConn)).ReadBatch
	}
	return oobConn
}

type statCounterConn struct {
	net.PacketConn
	readCounter  stats.Counter
	writeCounter stats.Counter
}

func (c *statCounterConn) ReadFrom(p []byte) (int, net.Addr, error) {
	n, addr, err := c.PacketConn.ReadFrom(p)
	if c.readCounter != nil {
		c.readCounter.Add(int64(n))
	}
	return n, addr, err
}

func (c *statCounterConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	n, err := c.PacketConn.WriteTo(p, addr)
	if c.writeCounter != nil {
		c.writeCounter.Add(int64(n))
	}
	return n, err
}

type setBufferConn struct {
	*statCounterConn
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
	n, oobn, flags, addr, err := c.readMsgUDP(b, oob)
	if c.readCounter != nil {
		c.readCounter.Add(int64(n))
	}
	return n, oobn, flags, addr, err
}

func (c *oobConn) WriteMsgUDP(b, oob []byte, addr *net.UDPAddr) (int, int, error) {
	n, oobn, err := c.writeMsgUDP(b, oob, addr)
	if c.writeCounter != nil {
		c.writeCounter.Add(int64(n))
	}
	return n, oobn, err
}

func (c *oobConn) ReadBatch(ms []ipv4.Message, flags int) (int, error) {
	n, err := c.readBatch(ms, flags)
	if c.readCounter != nil {
		c.readCounter.Add(int64(n))
	}
	return n, err
}

var (
	_ net.Conn       = (*oobConnWrapper)(nil)
	_ net.PacketConn = (*oobConnWrapper)(nil)
)

func newOOBConnWrapper(oobConn *oobConn) net.PacketConn {
	return &oobConnWrapper{oobConn: oobConn}
}

type oobConnWrapper struct {
	*oobConn
}

func (c *oobConnWrapper) Read(b []byte) (n int, err error) {
	panic("placeholder")
}

func (c *oobConn) Write(b []byte) (n int, err error) {
	panic("placeholder")
}

func (c *oobConn) RemoteAddr() net.Addr {
	panic("placeholder")
}
