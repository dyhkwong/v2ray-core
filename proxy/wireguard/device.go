package wireguard

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"sync"
	"time"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"

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
		return errors.New("device is already initialized")
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
	tun, n, stack, err := netstack.CreateNetTUN(localAddresses, mtu, handler != nil)
	if err != nil {
		return nil, err
	}

	if handler != nil {
		// handler is only used for promiscuous mode
		// capture all packets and send to handler

		tcpForwarder := tcp.NewForwarder(stack, 0, 65535, func(r *tcp.ForwarderRequest) {
			go func(r *tcp.ForwarderRequest) {
				var (
					wq waiter.Queue
					id = r.ID()
				)

				// Perform a TCP three-way handshake.
				ep, err := r.CreateEndpoint(&wq)
				if err != nil {
					newError(err.String()).AtError().WriteToLog()
					r.Complete(true)
					return
				}
				r.Complete(false)
				defer ep.Close()

				// enable tcp keep-alive to prevent hanging connections
				ep.SocketOptions().SetKeepAlive(true)

				// local address is actually destination
				handler(net.TCPDestination(net.IPAddress(id.LocalAddress.AsSlice()), net.Port(id.LocalPort)), gonet.NewTCPConn(&wq, ep))
			}(r)
		})
		stack.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpForwarder.HandlePacket)

		udpForwarder := udp.NewForwarder(stack, func(r *udp.ForwarderRequest) {
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
		stack.SetTransportProtocolHandler(udp.ProtocolNumber, udpForwarder.HandlePacket)
	}

	out.tun, out.net = tun, n
	return out, nil
}
