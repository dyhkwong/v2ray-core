package wireguard

import (
	"context"
	"errors"
	"io"
	"net/netip"
	"sync"

	"golang.zx2c4.com/wireguard/conn"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

type netReadInfo struct {
	// status
	waiter sync.WaitGroup
	// param
	buff []byte
	// result
	bytes    int
	endpoint conn.Endpoint
	err      error
}

// reduce duplicated code
type netBind struct {
	workers   int
	readQueue chan *netReadInfo
	closeOnce sync.Once
}

// SetMark implements conn.Bind
func (bind *netBind) SetMark(mark uint32) error {
	return nil
}

// ParseEndpoint implements conn.Bind
func (n *netBind) ParseEndpoint(s string) (conn.Endpoint, error) {
	host, portStr, err := net.SplitHostPort(s)
	if err != nil {
		return nil, err
	}
	port, err := net.PortFromString(portStr)
	if err != nil {
		return nil, err
	}
	dest := net.Destination{
		Address: net.ParseAddress(host),
		Port:    port,
		Network: net.Network_UDP,
	}
	if dest.Address.Family().IsDomain() {
		destAddr, err := net.ResolveUDPAddr("udp", dest.NetAddr())
		if err != nil {
			return nil, err
		}
		dest.Address = net.IPAddress(destAddr.IP)
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
	bind.readQueue = make(chan *netReadInfo)

	fn := func(bufs [][]byte, sizes []int, eps []conn.Endpoint) (n int, err error) {
		r := &netReadInfo{
			buff: bufs[0],
		}
		r.waiter.Add(1)
		bind.readQueue <- r
		r.waiter.Wait()
		sizes[0], eps[0] = r.bytes, r.endpoint
		return 1, r.err
	}
	workers := bind.workers
	if workers <= 0 {
		workers = 1
	}
	arr := make([]conn.ReceiveFunc, workers)
	for i := 0; i < workers; i++ {
		arr[i] = fn
	}

	return arr, uint16(uport), nil
}

// Close implements conn.Bind
func (bind *netBind) Close() error {
	bind.closeOnce.Do(func() {
		if bind.readQueue != nil {
			close(bind.readQueue)
		}
	})
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

	go func(readQueue <-chan *netReadInfo, endpoint *netEndpoint) {
		for {
			v, ok := <-readQueue
			if !ok {
				return
			}
			i, err := c.Read(v.buff)

			if i > 3 {
				v.buff[1] = 0
				v.buff[2] = 0
				v.buff[3] = 0
			}

			v.bytes = i
			v.endpoint = endpoint
			v.err = err
			v.waiter.Done()
			if err != nil && errors.Is(err, io.EOF) {
				endpoint.conn = nil
				return
			}
		}
	}(bind.readQueue, endpoint)

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

type netBindServer struct {
	netBind
}

func (bind *netBindServer) Send(buff [][]byte, endpoint conn.Endpoint) error {
	var err error

	nend, ok := endpoint.(*netEndpoint)
	if !ok {
		return conn.ErrWrongEndpointType
	}

	if nend.conn == nil {
		return newError("connection not open yet")
	}

	for _, buff := range buff {
		if _, err = nend.conn.Write(buff); err != nil {
			return err
		}
	}

	return err
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
