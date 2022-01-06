package packetaddr

import (
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
)

var _ buf.Reader = (*BufReader)(nil)

type BufReader struct {
	buf.Reader
	dest    net.Destination
	reverse bool
}

func (r *BufReader) ReadMultiBuffer() (buf.MultiBuffer, error) {
	mb, err := r.Reader.ReadMultiBuffer()
	if err != nil {
		return nil, err
	}
	for index, buffer := range mb {
		if r.reverse {
			endpoint := buffer.Endpoint
			if endpoint == nil {
				endpoint = &r.dest
			}
			if endpoint.Address.Family().IsDomain() {
				buf.ReleaseMulti(mb)
				return nil, newError("PacketAddr does not support domain name").AtError()
			}
			packet, err := AttachAddressToPacket(buffer, &net.UDPAddr{
				IP:   endpoint.Address.IP(),
				Port: int(endpoint.Port),
			})
			if err != nil {
				buf.ReleaseMulti(mb)
				return nil, newError("failed to attach address to packet").Base(err)
			}
			buffer = packet
		} else {
			extracted, addr, err := ExtractAddressFromPacket(buffer)
			if err != nil {
				buf.ReleaseMulti(mb)
				return nil, newError("failed to extract address from packet").Base(err)
			}
			udpAddr := addr.(*net.UDPAddr)
			extracted.Endpoint = &net.Destination{
				Network: net.Network_UDP,
				Address: net.IPAddress(udpAddr.IP),
				Port:    net.Port(udpAddr.Port),
			}
			buffer = extracted
		}
		mb[index] = buffer
	}
	return mb, nil
}

func NewPacketReader(reader buf.Reader) *BufReader {
	return &BufReader{
		Reader: reader,
	}
}

func NewReversePacketReader(reader buf.Reader, dest net.Destination) *BufReader {
	return &BufReader{
		Reader:  reader,
		dest:    dest,
		reverse: true,
	}
}

var _ buf.Writer = (*BufWriter)(nil)

type BufWriter struct {
	buf.Writer
	dest    net.Destination
	reverse bool
}

func (w *BufWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	for index, buffer := range mb {
		if w.reverse {
			extracted, addr, err := ExtractAddressFromPacket(buffer)
			if err != nil {
				buf.ReleaseMulti(mb)
				return newError("failed to extract address from packet").Base(err)
			}
			udpAddr := addr.(*net.UDPAddr)
			extracted.Endpoint = &net.Destination{
				Network: net.Network_UDP,
				Address: net.IPAddress(udpAddr.IP),
				Port:    net.Port(udpAddr.Port),
			}
			buffer = extracted
		} else {
			dest := &w.dest
			if buffer.Endpoint != nil {
				dest = buffer.Endpoint
			}
			if dest.Address.Family().IsDomain() {
				buf.ReleaseMulti(mb)
				return newError("PacketAddr does not support domain name").AtError()
			}
			packet, err := AttachAddressToPacket(buffer, &net.UDPAddr{
				IP:   dest.Address.IP(),
				Port: int(dest.Port),
			})
			if err != nil {
				buf.ReleaseMulti(mb)
				return newError("failed to attach address to packet").Base(err)
			}
			buffer = packet
		}
		mb[index] = buffer
	}
	return w.Writer.WriteMultiBuffer(mb)
}

func NewPacketWriter(writer buf.Writer, dest net.Destination) *BufWriter {
	return &BufWriter{
		Writer: writer,
		dest:   dest,
	}
}

func NewReversePacketWriter(writer buf.Writer) *BufWriter {
	return &BufWriter{
		Writer:  writer,
		reverse: true,
	}
}
