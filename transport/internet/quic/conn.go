package quic

import (
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"syscall"
	"time"

	"github.com/quic-go/quic-go"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

type sysConn struct {
	net.PacketConn
	header internet.PacketHeader
	auth   cipher.AEAD
}

type setBufferConn struct {
	*sysConn
	setWriteBufferFn interface{ SetWriteBuffer(int) error }
	setReadBufferFn  interface{ SetReadBuffer(int) error }
}

type syscallConn struct {
	*setBufferConn
	syscallConnFn interface {
		SyscallConn() (syscall.RawConn, error)
	}
}

func wrapSysConn(rawConn net.PacketConn, config *Config) (net.PacketConn, error) {
	header, err := getHeader(config)
	if err != nil {
		return nil, err
	}
	auth, err := getAuth(config)
	if err != nil {
		return nil, err
	}

	setWriteBufferFn, canSetWriteBuffer := rawConn.(interface{ SetWriteBuffer(int) error })
	setReadBufferFn, canSetReadBuffer := rawConn.(interface{ SetReadBuffer(int) error })
	syscallConnFn, isSyscallConn := rawConn.(interface {
		SyscallConn() (syscall.RawConn, error)
	})

	sysConn := &sysConn{
		PacketConn: rawConn,
		header:     header,
		auth:       auth,
	}

	if canSetWriteBuffer && canSetReadBuffer {
		setBufferConn := &setBufferConn{
			sysConn:          sysConn,
			setWriteBufferFn: setWriteBufferFn,
			setReadBufferFn:  setReadBufferFn,
		}
		if isSyscallConn {
			return &syscallConn{
				setBufferConn: setBufferConn,
				syscallConnFn: syscallConnFn,
			}, nil
		}
		return setBufferConn, nil
	}
	return sysConn, nil
}

var errInvalidPacket = errors.New("invalid packet")

func (c *sysConn) readFromInternal(p []byte) (int, net.Addr, error) {
	buffer := getBuffer()
	defer putBuffer(buffer)

	nBytes, addr, err := c.PacketConn.ReadFrom(buffer)
	if err != nil {
		return 0, nil, err
	}

	payload := buffer[:nBytes]
	if c.header != nil {
		if len(payload) <= int(c.header.Size()) {
			return 0, nil, errInvalidPacket
		}
		payload = payload[c.header.Size():]
	}

	if c.auth == nil {
		n := copy(p, payload)
		return n, addr, nil
	}

	if len(payload) <= c.auth.NonceSize() {
		return 0, nil, errInvalidPacket
	}

	nonce := payload[:c.auth.NonceSize()]
	payload = payload[c.auth.NonceSize():]

	p, err = c.auth.Open(p[:0], nonce, payload, nil)
	if err != nil {
		return 0, nil, errInvalidPacket
	}

	return len(p), addr, nil
}

func (c *sysConn) ReadFrom(p []byte) (int, net.Addr, error) {
	if c.header == nil && c.auth == nil {
		return c.PacketConn.ReadFrom(p)
	}

	for {
		n, addr, err := c.readFromInternal(p)
		if err != nil && err != errInvalidPacket {
			return 0, nil, err
		}
		if err == nil {
			return n, addr, nil
		}
	}
}

func (c *sysConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	if c.header == nil && c.auth == nil {
		return c.PacketConn.WriteTo(p, addr)
	}

	buffer := getBuffer()
	defer putBuffer(buffer)

	payload := buffer
	n := 0
	if c.header != nil {
		c.header.Serialize(payload)
		n = int(c.header.Size())
	}

	if c.auth == nil {
		nBytes := copy(payload[n:], p)
		n += nBytes
	} else {
		nounce := payload[n : n+c.auth.NonceSize()]
		common.Must2(rand.Read(nounce))
		n += c.auth.NonceSize()
		pp := c.auth.Seal(payload[:n], nounce, p, nil)
		n = len(pp)
	}

	return c.PacketConn.WriteTo(payload[:n], addr)
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

type interConn struct {
	stream *quic.Stream
	local  net.Addr
	remote net.Addr
}

func (c *interConn) Read(b []byte) (int, error) {
	return c.stream.Read(b)
}

func (c *interConn) WriteMultiBuffer(mb buf.MultiBuffer) error {
	mb = buf.Compact(mb)
	mb, err := buf.WriteMultiBuffer(c, mb)
	buf.ReleaseMulti(mb)
	return err
}

func (c *interConn) Write(b []byte) (int, error) {
	return c.stream.Write(b)
}

func (c *interConn) Close() error {
	return c.stream.Close()
}

func (c *interConn) LocalAddr() net.Addr {
	return c.local
}

func (c *interConn) RemoteAddr() net.Addr {
	return c.remote
}

func (c *interConn) SetDeadline(t time.Time) error {
	return c.stream.SetDeadline(t)
}

func (c *interConn) SetReadDeadline(t time.Time) error {
	return c.stream.SetReadDeadline(t)
}

func (c *interConn) SetWriteDeadline(t time.Time) error {
	return c.stream.SetWriteDeadline(t)
}
