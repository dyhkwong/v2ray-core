package shadowtls

import (
	"context"
	"io"

	B "github.com/sagernet/sing/common/buf"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/logger"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/errors"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

var (
	_ N.Dialer             = (*dialerWrapper)(nil)
	_ net.Conn             = (*pipeConnWrapper)(nil)
	_ N.PacketConn         = (*packetConnWrapper)(nil)
	_ logger.ContextLogger = (*loggerWrapper)(nil)
)

func toNetwork(network string) net.Network {
	switch N.NetworkName(network) {
	case N.NetworkTCP:
		return net.Network_TCP
	case N.NetworkUDP:
		return net.Network_UDP
	default:
		return net.Network_Unknown
	}
}

func toDestination(socksaddr M.Socksaddr, network net.Network) net.Destination {
	if socksaddr.IsIP() {
		return net.Destination{
			Network: network,
			Address: net.IPAddress(socksaddr.Addr.AsSlice()),
			Port:    net.Port(socksaddr.Port),
		}
	} else {
		return net.Destination{
			Network: network,
			Address: net.DomainAddress(socksaddr.Fqdn),
			Port:    net.Port(socksaddr.Port),
		}
	}
}

func toSocksaddr(destination net.Destination) M.Socksaddr {
	var addr M.Socksaddr
	switch destination.Address.Family() {
	case net.AddressFamilyDomain:
		addr.Fqdn = destination.Address.Domain()
	default:
		addr.Addr = M.AddrFromIP(destination.Address.IP())
	}
	addr.Port = uint16(destination.Port)
	return addr
}

type pipeConnWrapper struct {
	R io.Reader
	W buf.Writer
	net.Conn
}

func (w *pipeConnWrapper) Close() error {
	return nil
}

func (w *pipeConnWrapper) Read(b []byte) (n int, err error) {
	return w.R.Read(b)
}

func (w *pipeConnWrapper) Write(p []byte) (n int, err error) {
	n = len(p)
	var mb buf.MultiBuffer
	pLen := len(p)
	for pLen > 0 {
		buffer := buf.New()
		if pLen > buf.Size {
			buffer.Write(p[:buf.Size])
			p = p[buf.Size:]
		} else {
			buffer.Write(p)
		}
		pLen -= int(buffer.Len())
		mb = append(mb, buffer)
	}
	err = w.W.WriteMultiBuffer(mb)
	if err != nil {
		n = 0
		buf.ReleaseMulti(mb)
	}
	return
}

type packetConnWrapper struct {
	buf.Reader
	buf.Writer
	net.Conn
	Dest   net.Destination
	cached buf.MultiBuffer
}

func (w *packetConnWrapper) ReadPacket(buffer *B.Buffer) (M.Socksaddr, error) {
	if w.cached != nil {
		mb, bb := buf.SplitFirst(w.cached)
		if bb == nil {
			w.cached = nil
		} else {
			buffer.Write(bb.Bytes())
			w.cached = mb
			var destination net.Destination
			if bb.Endpoint != nil {
				destination = *bb.Endpoint
			} else {
				destination = w.Dest
			}
			bb.Release()
			return toSocksaddr(destination), nil
		}
	}
	mb, err := w.ReadMultiBuffer()
	if err != nil {
		return M.Socksaddr{}, err
	}
	nb, bb := buf.SplitFirst(mb)
	if bb == nil {
		return M.Socksaddr{}, nil
	} else {
		buffer.Write(bb.Bytes())
		w.cached = nb
		var destination net.Destination
		if bb.Endpoint != nil {
			destination = *bb.Endpoint
		} else {
			destination = w.Dest
		}
		bb.Release()
		return toSocksaddr(destination), nil
	}
}

func (w *packetConnWrapper) WritePacket(buffer *B.Buffer, destination M.Socksaddr) error {
	vBuf := buf.New()
	vBuf.Write(buffer.Bytes())
	endpoint := toDestination(destination, net.Network_UDP)
	vBuf.Endpoint = &endpoint
	return w.Writer.WriteMultiBuffer(buf.MultiBuffer{vBuf})
}

func (w *packetConnWrapper) Close() error {
	buf.ReleaseMulti(w.cached)
	return nil
}

func returnError(err error) error {
	if E.IsClosed(err) {
		return nil
	}
	return err
}

type dialerWrapper struct {
	dialer internet.Dialer
}

func (d *dialerWrapper) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	return d.dialer.Dial(ctx, toDestination(destination, toNetwork(network)))
}

func (d *dialerWrapper) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	panic("invalid")
}

type loggerWrapper struct {
	newError func(values ...any) *errors.Error
}

func newLogger(newErrorFunc func(values ...any) *errors.Error) *loggerWrapper {
	return &loggerWrapper{
		newErrorFunc,
	}
}

func (l *loggerWrapper) Trace(args ...any) {
}

func (l *loggerWrapper) Debug(args ...any) {
	l.newError(args...).AtDebug().WriteToLog()
}

func (l *loggerWrapper) Info(args ...any) {
	l.newError(args...).AtInfo().WriteToLog()
}

func (l *loggerWrapper) Warn(args ...any) {
	l.newError(args...).AtWarning().WriteToLog()
}

func (l *loggerWrapper) Error(args ...any) {
	l.newError(args...).AtError().WriteToLog()
}

func (l *loggerWrapper) Fatal(args ...any) {
	l.newError(args...).AtError().WriteToLog()
}

func (l *loggerWrapper) Panic(args ...any) {
	l.newError(args...).AtError().WriteToLog()
}

func (l *loggerWrapper) TraceContext(ctx context.Context, args ...any) {
	l.newError(args...).AtError().WriteToLog(session.ExportIDToError(ctx))
}

func (l *loggerWrapper) DebugContext(ctx context.Context, args ...any) {
	l.newError(args...).AtDebug().WriteToLog(session.ExportIDToError(ctx))
}

func (l *loggerWrapper) InfoContext(ctx context.Context, args ...any) {
	l.newError(args...).AtInfo().WriteToLog(session.ExportIDToError(ctx))
}

func (l *loggerWrapper) WarnContext(ctx context.Context, args ...any) {
	l.newError(args...).AtWarning().WriteToLog(session.ExportIDToError(ctx))
}

func (l *loggerWrapper) ErrorContext(ctx context.Context, args ...any) {
	l.newError(args...).AtError().WriteToLog(session.ExportIDToError(ctx))
}

func (l *loggerWrapper) FatalContext(ctx context.Context, args ...any) {
	l.newError(args...).AtError().WriteToLog(session.ExportIDToError(ctx))
}

func (l *loggerWrapper) PanicContext(ctx context.Context, args ...any) {
	l.newError(args...).AtError().WriteToLog(session.ExportIDToError(ctx))
}
