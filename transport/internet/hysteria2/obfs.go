package hysteria2

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"syscall"
	"time"

	"golang.org/x/crypto/blake2b"
)

const (
	smPSKMinLen = 4
	smSaltLen   = 8
	smKeyLen    = blake2b.Size256
)

const udpBufferSize = 2048 // QUIC packets are at most 1500 bytes long, so 2k should be more than enough

var _ Obfuscator = (*SalamanderObfuscator)(nil)

// Obfuscator is the interface that wraps the Obfuscate and Deobfuscate methods.
// Both methods return the number of bytes written to out.
// If a packet is not valid, the methods should return 0.
type Obfuscator interface {
	Obfuscate(in, out []byte) int
	Deobfuscate(in, out []byte) int
}

var _ net.PacketConn = (*obfsPacketConn)(nil)

type obfsPacketConn struct {
	Conn net.PacketConn
	Obfs Obfuscator

	readBuf    []byte
	readMutex  sync.Mutex
	writeBuf   []byte
	writeMutex sync.Mutex
}

// obfsPacketConnUDP is a special case of obfsPacketConn that uses a UDPConn
// as the underlying connection. We pass additional methods to quic-go to
// enable UDP-specific optimizations.
type obfsPacketConnUDP struct {
	*obfsPacketConn
	UDPConn *net.UDPConn
}

// WrapPacketConn enables obfuscation on a net.PacketConn.
// The obfuscation is transparent to the caller - the n bytes returned by
// ReadFrom and WriteTo are the number of original bytes, not after
// obfuscation/deobfuscation.
func WrapPacketConn(conn net.PacketConn, obfs Obfuscator) net.PacketConn {
	opc := &obfsPacketConn{
		Conn:     conn,
		Obfs:     obfs,
		readBuf:  make([]byte, udpBufferSize),
		writeBuf: make([]byte, udpBufferSize),
	}
	if udpConn, ok := conn.(*net.UDPConn); ok {
		return &obfsPacketConnUDP{
			obfsPacketConn: opc,
			UDPConn:        udpConn,
		}
	} else {
		return opc
	}
}

func (c *obfsPacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	for {
		c.readMutex.Lock()
		n, addr, err = c.Conn.ReadFrom(c.readBuf)
		if n <= 0 {
			c.readMutex.Unlock()
			return
		}
		n = c.Obfs.Deobfuscate(c.readBuf[:n], p)
		c.readMutex.Unlock()
		if n > 0 || err != nil {
			return
		}
		// Invalid packet, try again
	}
}

func (c *obfsPacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	c.writeMutex.Lock()
	nn := c.Obfs.Obfuscate(p, c.writeBuf)
	_, err = c.Conn.WriteTo(c.writeBuf[:nn], addr)
	c.writeMutex.Unlock()
	if err == nil {
		n = len(p)
	}
	return
}

func (c *obfsPacketConn) Close() error {
	return c.Conn.Close()
}

func (c *obfsPacketConn) LocalAddr() net.Addr {
	return c.Conn.LocalAddr()
}

func (c *obfsPacketConn) SetDeadline(t time.Time) error {
	return c.Conn.SetDeadline(t)
}

func (c *obfsPacketConn) SetReadDeadline(t time.Time) error {
	return c.Conn.SetReadDeadline(t)
}

func (c *obfsPacketConn) SetWriteDeadline(t time.Time) error {
	return c.Conn.SetWriteDeadline(t)
}

// UDP-specific methods below

func (c *obfsPacketConnUDP) SetReadBuffer(bytes int) error {
	return c.UDPConn.SetReadBuffer(bytes)
}

func (c *obfsPacketConnUDP) SetWriteBuffer(bytes int) error {
	return c.UDPConn.SetWriteBuffer(bytes)
}

func (c *obfsPacketConnUDP) SyscallConn() (syscall.RawConn, error) {
	return c.UDPConn.SyscallConn()
}

var ErrPSKTooShort = fmt.Errorf("PSK must be at least %d bytes", smPSKMinLen)

// SalamanderObfuscator is an obfuscator that obfuscates each packet with
// the BLAKE2b-256 hash of a pre-shared key combined with a random salt.
// Packet format: [8-byte salt][payload]
type SalamanderObfuscator struct {
	PSK     []byte
	RandSrc *rand.Rand

	lk sync.Mutex
}

func NewSalamanderObfuscator(psk []byte) (*SalamanderObfuscator, error) {
	if len(psk) < smPSKMinLen {
		return nil, ErrPSKTooShort
	}
	return &SalamanderObfuscator{
		PSK:     psk,
		RandSrc: rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

func (o *SalamanderObfuscator) Obfuscate(in, out []byte) int {
	outLen := len(in) + smSaltLen
	if len(out) < outLen {
		return 0
	}
	o.lk.Lock()
	_, _ = o.RandSrc.Read(out[:smSaltLen])
	o.lk.Unlock()
	key := o.key(out[:smSaltLen])
	for i, c := range in {
		out[i+smSaltLen] = c ^ key[i%smKeyLen]
	}
	return outLen
}

func (o *SalamanderObfuscator) Deobfuscate(in, out []byte) int {
	outLen := len(in) - smSaltLen
	if outLen <= 0 || len(out) < outLen {
		return 0
	}
	key := o.key(in[:smSaltLen])
	for i, c := range in[smSaltLen:] {
		out[i] = c ^ key[i%smKeyLen]
	}
	return outLen
}

func (o *SalamanderObfuscator) key(salt []byte) [smKeyLen]byte {
	return blake2b.Sum256(append(o.PSK, salt...))
}
