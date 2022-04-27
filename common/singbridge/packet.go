package singbridge

import (
	singbuf "github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/common/network"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/transport"
)

var _ network.PacketConn = (*packetConnWrapper)(nil)

func NewPacketConnWrapper(link *transport.Link, dest net.Destination) *packetConnWrapper {
	conn := &packetConnWrapper{
		Reader: link.Reader,
		Writer: link.Writer,
		dest:   dest,
	}
	return conn
}

type packetConnWrapper struct {
	buf.Reader
	buf.Writer
	dest   net.Destination
	cached buf.MultiBuffer
	net.Conn
}

func (w *packetConnWrapper) ReadPacket(buffer *singbuf.Buffer) (metadata.Socksaddr, error) {
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
				destination = w.dest
			}
			bb.Release()
			return ToSocksAddr(destination), nil
		}
	}
	mb, err := w.ReadMultiBuffer()
	if err != nil {
		return metadata.Socksaddr{}, err
	}
	nb, bb := buf.SplitFirst(mb)
	if bb == nil {
		return metadata.Socksaddr{}, nil
	} else {
		buffer.Write(bb.Bytes())
		w.cached = nb
		var destination net.Destination
		if bb.Endpoint != nil {
			destination = *bb.Endpoint
		} else {
			destination = w.dest
		}
		bb.Release()
		return ToSocksAddr(destination), nil
	}
}

func (w *packetConnWrapper) WritePacket(buffer *singbuf.Buffer, destination metadata.Socksaddr) error {
	vBuf := buf.New()
	vBuf.Write(buffer.Bytes())
	endpoint := ToDestination(destination, net.Network_UDP)
	vBuf.Endpoint = &endpoint
	return w.WriteMultiBuffer(buf.MultiBuffer{vBuf})
}

func (w *packetConnWrapper) Close() error {
	buf.ReleaseMulti(w.cached)
	return nil
}
