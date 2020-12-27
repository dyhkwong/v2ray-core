package udp

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/protocol/udp"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal"
	"github.com/v2fly/v2ray-core/v5/common/signal/done"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/transport"
)

type ResponseCallback func(ctx context.Context, packet *udp.Packet)

type connEntry struct {
	link   *transport.Link
	ctx    context.Context
	timer  signal.ActivityUpdater
	cancel context.CancelFunc
}

type Dispatcher struct {
	sync.RWMutex
	conn       *connEntry
	dispatcher routing.Dispatcher
	callback   ResponseCallback
}

func (v *Dispatcher) Close() error {
	return nil
}

func NewSplitDispatcher(dispatcher routing.Dispatcher, callback ResponseCallback) DispatcherI {
	return &Dispatcher{
		dispatcher: dispatcher,
		callback:   callback,
	}
}

func (v *Dispatcher) getInboundRay(ctx context.Context, dest net.Destination) (*connEntry, error) {
	v.Lock()
	defer v.Unlock()

	if v.conn != nil {
		select {
		case <-v.conn.ctx.Done():
			common.Interrupt(v.conn.link.Reader)
			common.Close(v.conn.link.Writer)
			v.conn = nil
		default:
			return v.conn, nil
		}
	}

	newError("establishing new connection for ", dest).WriteToLog()

	ctx, cancel := context.WithCancel(ctx)
	timer := signal.CancelAfterInactivity(ctx, cancel, time.Second*300)
	link, err := v.dispatcher.Dispatch(ctx, dest)
	if err != nil {
		return nil, newError("failed to dispatch request to: ", dest).Base(err)
	}
	entry := &connEntry{
		link:   link,
		ctx:    ctx,
		timer:  timer,
		cancel: cancel,
	}
	v.conn = entry
	go handleInput(ctx, entry, dest, v.callback)
	return entry, nil
}

func (v *Dispatcher) Dispatch(ctx context.Context, destination net.Destination, payload *buf.Buffer) {
	// TODO: Add user to destString
	newError("dispatch request to: ", destination).AtDebug().WriteToLog(session.ExportIDToError(ctx))

	conn, err := v.getInboundRay(ctx, destination)
	if err != nil {
		payload.Release()
		return
	}
	outputStream := conn.link.Writer
	if outputStream != nil {
		payload.Endpoint = &destination
		if err := outputStream.WriteMultiBuffer(buf.MultiBuffer{payload}); err != nil {
			newError("failed to write first UDP payload").Base(err).WriteToLog(session.ExportIDToError(ctx))
			conn.cancel()
			return
		}
		conn.timer.Update()
	}
}

func handleInput(ctx context.Context, conn *connEntry, dest net.Destination, callback ResponseCallback) {
	defer conn.cancel()

	input := conn.link.Reader
	timer := conn.timer

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		mb, err := input.ReadMultiBuffer()
		if err != nil {
			newError("failed to handle UDP input").Base(err).WriteToLog(session.ExportIDToError(ctx))
			return
		}
		timer.Update()
		for _, b := range mb {
			if b.Endpoint != nil {
				dest = *b.Endpoint
			}
			callback(ctx, &udp.Packet{
				Payload: b,
				Source:  dest,
			})
		}
	}
}

type dispatcherConn struct {
	ctx        context.Context
	dispatcher *Dispatcher
	cache      chan *udp.Packet
	done       *done.Instance
}

func DialDispatcher(ctx context.Context, dispatcher routing.Dispatcher) (net.PacketConn, error) {
	c := &dispatcherConn{
		ctx:   ctx,
		cache: make(chan *udp.Packet, 16),
		done:  done.New(),
	}

	d := NewSplitDispatcher(dispatcher, c.callback)
	c.dispatcher = d.(*Dispatcher)
	return c, nil
}

func (c *dispatcherConn) callback(ctx context.Context, packet *udp.Packet) {
	select {
	case <-c.done.Wait():
		packet.Payload.Release()
		return
	case c.cache <- packet:
	default:
		packet.Payload.Release()
		return
	}
}

func (c *dispatcherConn) ReadFrom(p []byte) (int, net.Addr, error) {
	select {
	case <-c.done.Wait():
		return 0, nil, io.EOF
	case packet := <-c.cache:
		n := copy(p, packet.Payload.Bytes())
		return n, &net.UDPAddr{
			IP:   packet.Source.Address.IP(),
			Port: int(packet.Source.Port),
		}, nil
	}
}

func (c *dispatcherConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	buffer := buf.New()
	raw := buffer.Extend(buf.Size)
	n := copy(raw, p)
	buffer.Resize(0, int32(n))

	dest := net.DestinationFromAddr(addr)
	buffer.Endpoint = &dest
	c.dispatcher.Dispatch(c.ctx, dest, buffer)
	return n, nil
}

func (c *dispatcherConn) Close() error {
	return c.done.Close()
}

func (c *dispatcherConn) LocalAddr() net.Addr {
	return &net.UDPAddr{
		IP:   []byte{0, 0, 0, 0},
		Port: 0,
	}
}

func (c *dispatcherConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *dispatcherConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *dispatcherConn) SetWriteDeadline(t time.Time) error {
	return nil
}
