package hysteria2

import (
	gonet "net"
	"net/netip"
	"sync"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

const (
	protocolName = "hysteria2"
)

var (
	localAddrMutex sync.Mutex
	localIP        netip.Addr = netip.IPv6Unspecified()
	localPort      uint16
)

// https://github.com/quic-go/quic-go/commit/8189e75be6121fdc31dc1d6085f17015e9154667#diff-4c6aaadced390f3ce9bec0a9c9bb5203d5fa85df79023e3e0eec423dc9baa946R48-R62
func generateAddr() *net.UDPAddr {
	localAddrMutex.Lock()
	defer func() {
		if localPort == 65535 {
			localPort = 0
			localIP = localIP.Next()
		} else {
			localPort++
		}
		localAddrMutex.Unlock()
	}()
	return gonet.UDPAddrFromAddrPort(netip.AddrPortFrom(localIP, localPort))
}

func wrapPacketConn(pc net.PacketConn) net.PacketConn {
	if uc, ok := pc.(*net.UDPConn); ok {
		return &udpConn{
			UDPConn:   uc,
			localAddr: generateAddr(),
		}
	}
	return &packetConn{
		PacketConn: pc,
		localAddr:  generateAddr(),
	}
}

type packetConn struct {
	net.PacketConn
	localAddr net.Addr
}

func (c *packetConn) LocalAddr() net.Addr {
	return c.localAddr
}

type udpConn struct {
	*net.UDPConn
	localAddr net.Addr
}

func (c *udpConn) LocalAddr() net.Addr {
	return c.localAddr
}

func init() {
	common.Must(internet.RegisterProtocolConfigCreator(protocolName, func() interface{} {
		return new(Config)
	}))
}
