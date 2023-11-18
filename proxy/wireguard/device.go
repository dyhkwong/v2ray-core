package wireguard

import (
	"context"
	"fmt"
	"io"
	"net/netip"
	"sync"
	"time"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/checksum"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/log"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/proxy/wireguard/netstack"
)

type Tunnel interface {
	BuildDevice(ipc string, bind conn.Bind) error
	DialContextTCPAddrPort(ctx context.Context, addr netip.AddrPort) (net.Conn, error)
	DialUDPAddrPort(laddr, raddr netip.AddrPort) (net.Conn, error)
	Close() error
}

type tunnel struct {
	tun    tun.Device
	device *device.Device
	rw     sync.Mutex
}

func (t *tunnel) BuildDevice(ipc string, bind conn.Bind) (err error) {
	t.rw.Lock()
	defer t.rw.Unlock()

	if t.device != nil {
		return newError("device is already initialized")
	}

	logger := &device.Logger{
		Verbosef: func(format string, args ...any) {
			log.Record(&log.GeneralMessage{
				Severity: log.Severity_Debug,
				Content:  fmt.Sprintf(format, args...),
			})
		},
		Errorf: func(format string, args ...any) {
			log.Record(&log.GeneralMessage{
				Severity: log.Severity_Error,
				Content:  fmt.Sprintf(format, args...),
			})
		},
	}

	t.device = device.NewDevice(t.tun, bind, logger)
	if err = t.device.IpcSet(ipc); err != nil {
		return err
	}
	if err = t.device.Up(); err != nil {
		return err
	}
	return nil
}

func (t *tunnel) Close() (err error) {
	t.rw.Lock()
	defer t.rw.Unlock()

	if t.device == nil {
		return nil
	}

	t.device.Close()
	t.device = nil
	err = t.tun.Close()
	t.tun = nil
	return err
}

var _ Tunnel = (*wgNet)(nil)

type wgNet struct {
	tunnel
	net *netstack.Net
}

func (g *wgNet) Close() error {
	return g.tunnel.Close()
}

func (g *wgNet) DialContextTCPAddrPort(ctx context.Context, addr netip.AddrPort) (net.Conn, error) {
	return g.net.DialContextTCPAddrPort(ctx, addr)
}

func (g *wgNet) DialUDPAddrPort(laddr, raddr netip.AddrPort) (net.Conn, error) {
	return g.net.DialUDPAddrPort(laddr, raddr)
}

func createTun(localAddresses []netip.Addr, mtu int, handler func(dest net.Destination, conn net.Conn)) (Tunnel, error) {
	out := &wgNet{}
	tun, n, gstack, err := netstack.CreateNetTUN(localAddresses, mtu, handler != nil)
	if err != nil {
		return nil, err
	}

	if handler != nil {
		// handler is only used for promiscuous mode
		// capture all packets and send to handler

		tcpForwarder := tcp.NewForwarder(gstack, 0, 65535, func(r *tcp.ForwarderRequest) {
			go func(r *tcp.ForwarderRequest) {
				var wq waiter.Queue
				id := r.ID()

				ep, err := r.CreateEndpoint(&wq)
				if err != nil {
					newError(err.String()).AtError().WriteToLog()
					r.Complete(true)
					return
				}

				options := ep.SocketOptions()
				options.SetKeepAlive(false)
				options.SetReuseAddress(true)
				options.SetReusePort(true)

				handler(net.TCPDestination(net.IPAddress(id.LocalAddress.AsSlice()), net.Port(id.LocalPort)), gonet.NewTCPConn(&wq, ep))

				ep.Close()
				r.Complete(false)
			}(r)
		})
		gstack.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpForwarder.HandlePacket)

		/*udpForwarder := udp.NewForwarder(gstack, func(r *udp.ForwarderRequest) {
			go func(r *udp.ForwarderRequest) {
				var (
					wq waiter.Queue
					id = r.ID()
				)

				ep, err := r.CreateEndpoint(&wq)
				if err != nil {
					newError(err.String()).AtError().WriteToLog()
					return
				}
				defer ep.Close()

				// prevents hanging connections and ensure timely release
				ep.SocketOptions().SetLinger(tcpip.LingerOption{
					Enabled: true,
					Timeout: 15 * time.Second,
				})

				handler(net.UDPDestination(net.IPAddress(id.LocalAddress.AsSlice()), net.Port(id.LocalPort)), gonet.NewUDPConn(&wq, ep))
			}(r)
		})
		gstack.SetTransportProtocolHandler(udp.ProtocolNumber, udpForwarder.HandlePacket)*/

		manager := &udpManager{
			stack:   gstack,
			handler: handler,
			m:       make(map[string]*udpConn),
		}

		gstack.SetTransportProtocolHandler(udp.ProtocolNumber, func(id stack.TransportEndpointID, pkt *stack.PacketBuffer) bool {
			data := pkt.Clone().Data().AsRange().ToSlice()
			src := net.UDPDestination(net.IPAddress(id.RemoteAddress.AsSlice()), net.Port(id.RemotePort))
			dest := net.UDPDestination(net.IPAddress(id.LocalAddress.AsSlice()), net.Port(id.LocalPort))
			manager.feed(src, dest, data)
			return true
		})
	}

	out.tun, out.net = tun, n
	return out, nil
}

type udpManager struct {
	stack   *stack.Stack
	handler func(dest net.Destination, conn net.Conn)
	m       map[string]*udpConn
	mutex   sync.RWMutex
}

func (m *udpManager) feed(source net.Destination, dest net.Destination, data []byte) {
	m.mutex.RLock()
	uc, ok := m.m[source.NetAddr()]
	if ok {
		select {
		case uc.ch <- data:
		default:
		}
		m.mutex.RUnlock()
		return
	}
	m.mutex.RUnlock()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	uc, ok = m.m[source.NetAddr()]
	if !ok {
		uc = &udpConn{
			ch:     make(chan []byte, 1024),
			source: source,
			dest:   dest,
		}
		uc.writeFunc = m.writeRawUDPPacket
		uc.closeFunc = func() {
			m.mutex.Lock()
			m.close(uc)
			m.mutex.Unlock()
		}
		m.m[source.NetAddr()] = uc
		go m.handler(dest, uc)
	}

	select {
	case uc.ch <- data:
	default:
	}
}

func (m *udpManager) close(uc *udpConn) {
	if !uc.closed {
		uc.closed = true
		close(uc.ch)
		delete(m.m, uc.source.NetAddr())
	}
}

func (m *udpManager) writeRawUDPPacket(payload []byte, src net.Destination, dst net.Destination) error {
	udpLen := header.UDPMinimumSize + len(payload)
	srcIP := tcpip.AddrFromSlice(src.Address.IP())
	dstIP := tcpip.AddrFromSlice(dst.Address.IP())

	// build packet with appropriate IP header size
	isIPv4 := dst.Address.Family().IsIPv4()
	ipHdrSize := header.IPv6MinimumSize
	ipProtocol := header.IPv6ProtocolNumber
	if isIPv4 {
		ipHdrSize = header.IPv4MinimumSize
		ipProtocol = header.IPv4ProtocolNumber
	}

	pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{
		ReserveHeaderBytes: ipHdrSize + header.UDPMinimumSize,
		Payload:            buffer.MakeWithData(payload),
	})
	defer pkt.DecRef()

	// Build UDP header
	udpHdr := header.UDP(pkt.TransportHeader().Push(header.UDPMinimumSize))
	udpHdr.Encode(&header.UDPFields{
		SrcPort: uint16(src.Port),
		DstPort: uint16(dst.Port),
		Length:  uint16(udpLen),
	})

	// Calculate and set UDP checksum
	xsum := header.PseudoHeaderChecksum(header.UDPProtocolNumber, srcIP, dstIP, uint16(udpLen))
	udpHdr.SetChecksum(^udpHdr.CalculateChecksum(checksum.Checksum(payload, xsum)))

	// Build IP header
	if isIPv4 {
		ipHdr := header.IPv4(pkt.NetworkHeader().Push(header.IPv4MinimumSize))
		ipHdr.Encode(&header.IPv4Fields{
			TotalLength: uint16(header.IPv4MinimumSize + udpLen),
			TTL:         64,
			Protocol:    uint8(header.UDPProtocolNumber),
			SrcAddr:     srcIP,
			DstAddr:     dstIP,
		})
		ipHdr.SetChecksum(^ipHdr.CalculateChecksum())
	} else {
		ipHdr := header.IPv6(pkt.NetworkHeader().Push(header.IPv6MinimumSize))
		ipHdr.Encode(&header.IPv6Fields{
			PayloadLength:     uint16(udpLen),
			TransportProtocol: header.UDPProtocolNumber,
			HopLimit:          64,
			SrcAddr:           srcIP,
			DstAddr:           dstIP,
		})
	}

	// dispatch the packet
	err := m.stack.WriteRawPacket(1, ipProtocol, buffer.MakeWithView(pkt.ToView()))
	if err != nil {
		return newError(err)
	}

	return nil
}

type udpConn struct {
	ch        chan []byte
	source    net.Destination
	dest      net.Destination
	writeFunc func(payload []byte, src net.Destination, dst net.Destination) error
	closeFunc func()
	closed    bool
}

func (c *udpConn) Read(p []byte) (int, error) {
	b, ok := <-c.ch
	if !ok {
		return 0, io.EOF
	}
	n := copy(p, b)
	if n != len(b) {
		return 0, io.ErrShortBuffer
	}
	return n, nil
}

func (c *udpConn) WriteMultiBuffer(mb buf.MultiBuffer) error {
	defer buf.ReleaseMulti(mb)
	for _, b := range mb {
		dest := c.dest
		if b.Endpoint != nil {
			dest = *b.Endpoint
		}
		if err := c.writeFunc(b.Bytes(), dest, c.source); err != nil {
			return err
		}
	}
	return nil
}

func (c *udpConn) Write(p []byte) (int, error) {
	err := c.writeFunc(p, c.dest, c.source)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (c *udpConn) Close() error {
	c.closeFunc()
	return nil
}

func (c *udpConn) LocalAddr() net.Addr {
	// fake
	return &net.UDPAddr{
		IP:   c.source.Address.IP(),
		Port: int(c.source.Port),
	}
}

func (c *udpConn) RemoteAddr() net.Addr {
	return &net.UDPAddr{
		IP:   c.source.Address.IP(),
		Port: int(c.source.Port),
	}
}

func (c *udpConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *udpConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *udpConn) SetWriteDeadline(t time.Time) error {
	return nil
}
