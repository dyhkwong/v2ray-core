package http3

import (
	"syscall"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/features/stats"
)

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
	return &syscallConn{
		setBufferConn:  setBufferConn,
		setWriteBuffer: setBufferFn.SetWriteBuffer,
		setReadBuffer:  setBufferFn.SetReadBuffer,
		syscallConn:    syscallConnFn.SyscallConn,
	}
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
