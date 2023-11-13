package wireguard

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"sync"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"

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
	return nil
}

var _ Tunnel = (*wgNet)(nil)

type wgNet struct {
	tunnel
	net *netstack.Net
}

func (g *wgNet) Close() error {
	g.tunnel.rw.Lock()
	defer g.tunnel.rw.Unlock()
	return g.tunnel.Close()
}

func (g *wgNet) DialContextTCPAddrPort(ctx context.Context, addr netip.AddrPort) (net.Conn, error) {
	return g.net.DialContextTCPAddrPort(ctx, addr)
}

func (g *wgNet) DialUDPAddrPort(laddr, raddr netip.AddrPort) (net.Conn, error) {
	return g.net.DialUDPAddrPort(laddr, raddr)
}

func CreateTun(localAddresses []netip.Addr, mtu int) (Tunnel, error) {
	out := &wgNet{}
	tun, n, err := netstack.CreateNetTUN(localAddresses, mtu)
	if err != nil {
		return nil, err
	}
	out.tun, out.net = tun, n
	return out, nil
}
