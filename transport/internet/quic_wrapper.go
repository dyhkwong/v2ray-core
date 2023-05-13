package internet

import (
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/uuid"
)

// https://github.com/quic-go/quic-go/commit/8189e75be6121fdc31dc1d6085f17015e9154667#diff-4c6aaadced390f3ce9bec0a9c9bb5203d5fa85df79023e3e0eec423dc9baa946R48-R62

func WrapPacketConn(rawConn net.Conn) net.PacketConn {
	switch c := rawConn.(type) {
	case *PacketConnWrapper:
		return newPacketConn(c.Conn)
	case net.PacketConn:
		return newPacketConn(c)
	default:
		return newPacketConnFromConn(c)
	}
}

func newPacketConn(pc net.PacketConn) net.PacketConn {
	uuid := uuid.New()
	if uc, ok := pc.(*net.UDPConn); ok {
		return &udpConn{UDPConn: uc, localAddr: &net.UnixAddr{Name: uuid.String()}}
	}
	return &packetConn{PacketConn: pc, localAddr: &net.UnixAddr{Name: uuid.String()}}
}

func newPacketConnFromConn(c net.Conn) net.PacketConn {
	uuid := uuid.New()
	return &conn{Conn: c, localAddr: &net.UnixAddr{Name: uuid.String()}}
}

type udpConn struct {
	*net.UDPConn
	localAddr net.Addr
}

func (c *udpConn) LocalAddr() net.Addr {
	return c.localAddr
}

type packetConn struct {
	net.PacketConn
	localAddr net.Addr
}

func (c *packetConn) LocalAddr() net.Addr {
	return c.localAddr
}

type conn struct {
	net.Conn
	localAddr net.Addr
}

func (c *conn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	n, err = c.Read(p)
	return n, c.RemoteAddr(), err
}

func (c *conn) WriteTo(p []byte, _ net.Addr) (n int, err error) {
	return c.Write(p)
}

func (c *conn) LocalAddr() net.Addr {
	return c.localAddr
}
