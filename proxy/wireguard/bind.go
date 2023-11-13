package wireguard

import (
	"context"
	gonet "net"
	"net/netip"
	"runtime"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/features/dns/localdns"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

type netReadInfo struct {
	buff     []byte
	endpoint conn.Endpoint
}

type netBind struct {
	workers   int
	readQueue chan *netReadInfo
	closedCh  chan struct{}
	resolver  func(ctx context.Context, domain string) net.Address
}

// SetMark implements conn.Bind
func (bind *netBind) SetMark(mark uint32) error {
	return nil
}

// ParseEndpoint implements conn.Bind
func (n *netBind) ParseEndpoint(s string) (conn.Endpoint, error) {
	dest, err := net.ParseDestination(s)
	if err != nil {
		return nil, err
	}
	dest.Network = net.Network_UDP
	if dest.Address.Family().IsDomain() {
		if n.resolver != nil {
			addr := n.resolver(context.TODO(), dest.Address.Domain())
			if addr == nil {
				return nil, newError("failed to resolve domain ", dest.Address.Domain())
			}
			dest.Address = addr
		} else {
			addr, err := localdns.New().LookupIP(dest.Address.Domain())
			if err != nil {
				return nil, err
			}
			dest.Address = net.IPAddress(addr[0])
		}
	}
	return &netEndpoint{
		dest: dest,
	}, nil
}

// BatchSize implements conn.Bind
func (bind *netBind) BatchSize() int {
	return 1
}

// Open implements conn.Bind
func (bind *netBind) Open(uport uint16) ([]conn.ReceiveFunc, uint16, error) {
	bind.closedCh = make(chan struct{})
	fn := func(bufs [][]byte, sizes []int, eps []conn.Endpoint) (n int, err error) {
		select {
		case r := <-bind.readQueue:
			sizes[0], eps[0] = copy(bufs[0], r.buff), r.endpoint
			return 1, nil
		case <-bind.closedCh:
			return 0, gonet.ErrClosed
		}
	}
	workers := bind.workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	arr := make([]conn.ReceiveFunc, workers)
	for i := 0; i < workers; i++ {
		arr[i] = fn
	}
	return arr, uint16(uport), nil
}

// Close implements conn.Bind
func (bind *netBind) Close() error {
	if bind.closedCh != nil {
		close(bind.closedCh)
	}
	return nil
}

type netBindClient struct {
	netBind
	ctx      context.Context
	dialer   internet.Dialer
	reserved []byte
}

func (bind *netBindClient) connectTo(endpoint *netEndpoint) error {
	c, err := bind.dialer.Dial(bind.ctx, endpoint.dest)
	if err != nil {
		return err
	}
	endpoint.conn = c
	go func() {
		for {
			buff := make([]byte, device.MaxMessageSize)
			n, err := c.Read(buff)
			if err != nil {
				endpoint.conn = nil
				c.Close()
				return
			}
			if n > 3 {
				buff[1] = 0
				buff[2] = 0
				buff[3] = 0
			}
			select {
			case bind.readQueue <- &netReadInfo{
				buff:     buff[:n],
				endpoint: endpoint,
			}:
			case <-bind.closedCh:
				endpoint.conn = nil
				c.Close()
				return
			}
		}
	}()
	return nil
}

func (bind *netBindClient) Send(buff [][]byte, endpoint conn.Endpoint) error {
	var err error
	nend, ok := endpoint.(*netEndpoint)
	if !ok {
		return conn.ErrWrongEndpointType
	}
	if nend.conn == nil {
		err = bind.connectTo(nend)
		if err != nil {
			return err
		}
	}
	for _, buff := range buff {
		if len(buff) > 3 && len(bind.reserved) == 3 {
			copy(buff[1:], bind.reserved)
		}
		if _, err = nend.conn.Write(buff); err != nil {
			return err
		}
	}
	return nil
}

type netEndpoint struct {
	dest net.Destination
	conn net.Conn
}

func (netEndpoint) ClearSrc() {}

func (e netEndpoint) DstIP() netip.Addr {
	return netip.Addr{}
}

func (e netEndpoint) SrcIP() netip.Addr {
	return netip.Addr{}
}

func (e netEndpoint) DstToBytes() []byte {
	var b []byte
	if e.dest.Address.Family().IsIPv4() {
		b = e.dest.Address.IP()
	} else {
		b = e.dest.Address.IP()
	}
	b = append(b, byte(e.dest.Port), byte(e.dest.Port>>8))
	return b
}

func (e netEndpoint) DstToString() string {
	return e.dest.NetAddr()
}

func (e netEndpoint) SrcToString() string {
	return ""
}
