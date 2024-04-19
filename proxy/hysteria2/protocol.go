package hysteria2

import (
	"io"
	gonet "net"

	hyProtocol "github.com/apernet/hysteria/core/international/protocol"
	"github.com/apernet/quic-go/quicvarint"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	hy2_transport "github.com/v2fly/v2ray-core/v5/transport/internet/hysteria2"
)

// ConnWriter is TCP Connection Writer Wrapper
type ConnWriter struct {
	io.Writer
	Target        net.Destination
	Account       *MemoryAccount
	TCPHeaderSent bool
}

// Write implements io.Writer
func (c *ConnWriter) Write(p []byte) (n int, err error) {
	if !c.TCPHeaderSent {
		if err := c.writeHeader(); err != nil {
			return 0, newError("failed to write request header").Base(err)
		}
	}

	return c.Writer.Write(p)
}

// WriteMultiBuffer implements buf.Writer
func (c *ConnWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	defer buf.ReleaseMulti(mb)

	for _, b := range mb {
		if !b.IsEmpty() {
			if _, err := c.Write(b.Bytes()); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *ConnWriter) WriteHeader() error {
	if !c.TCPHeaderSent {
		if err := c.writeHeader(); err != nil {
			return err
		}
	}
	return nil
}

func QuicLen(s int) int {
	return int(quicvarint.Len(uint64(s)))
}

func (c *ConnWriter) writeHeader() error {
	padding := "Jimmy Was Here"
	paddingLen := len(padding)
	addressAndPort := c.Target.Address.String() + ":" + c.Target.Port.String()
	addressLen := len(addressAndPort)
	size := QuicLen(addressLen) + addressLen + QuicLen(paddingLen) + paddingLen

	buf := make([]byte, size)
	i := hyProtocol.VarintPut(buf, uint64(addressLen))
	i += copy(buf[i:], addressAndPort)
	i += hyProtocol.VarintPut(buf[i:], uint64(paddingLen))
	copy(buf[i:], padding)

	_, err := c.Writer.Write(buf)
	if err == nil {
		c.TCPHeaderSent = true
	}
	return err
}

// PacketWriter UDP Connection Writer Wrapper
type PacketWriter struct {
	io.Writer
	HyConn *hy2_transport.HyConn
	Target net.Destination
}

// WriteMultiBuffer implements buf.Writer
func (w *PacketWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	for _, b := range mb {
		if b.IsEmpty() {
			continue
		}
		target := &w.Target
		if b.Endpoint != nil {
			target = b.Endpoint
		}
		if _, err := w.writePacket(b.Bytes(), *target); err != nil {
			buf.ReleaseMulti(mb)
			return err
		}
	}

	return nil
}

// WriteMultiBufferWithMetadata writes udp packet with destination specified
func (w *PacketWriter) WriteMultiBufferWithMetadata(mb buf.MultiBuffer, dest net.Destination) error {
	for _, b := range mb {
		if b.IsEmpty() {
			continue
		}
		if _, err := w.writePacket(b.Bytes(), dest); err != nil {
			buf.ReleaseMulti(mb)
			return err
		}
	}

	return nil
}

func (w *PacketWriter) WriteTo(payload []byte, addr gonet.Addr) (int, error) {
	dest := net.DestinationFromAddr(addr)

	return w.writePacket(payload, dest)
}

func (w *PacketWriter) writePacket(payload []byte, dest net.Destination) (int, error) {
	return w.HyConn.WritePacket(payload, dest)
}

// ConnReader is TCP Connection Reader Wrapper
type ConnReader struct {
	io.Reader
	Target net.Destination
}

// Read implements io.Reader
func (c *ConnReader) Read(p []byte) (int, error) {
	return c.Reader.Read(p)
}

// ReadMultiBuffer implements buf.Reader
func (c *ConnReader) ReadMultiBuffer() (buf.MultiBuffer, error) {
	b := buf.New()
	_, err := b.ReadFrom(c)
	return buf.MultiBuffer{b}, err
}

// PacketPayload combines udp payload and destination
type PacketPayload struct {
	Target net.Destination
	Buffer buf.MultiBuffer
}

// PacketReader is UDP Connection Reader Wrapper
type PacketReader struct {
	io.Reader
	HyConn *hy2_transport.HyConn
}

// ReadMultiBuffer implements buf.Reader
func (r *PacketReader) ReadMultiBuffer() (buf.MultiBuffer, error) {
	p, err := r.ReadMultiBufferWithMetadata()
	if p != nil {
		return p.Buffer, err
	}
	return nil, err
}

// ReadMultiBufferWithMetadata reads udp packet with destination
func (r *PacketReader) ReadMultiBufferWithMetadata() (*PacketPayload, error) {
	_, data, dest, err := r.HyConn.ReadPacket()
	if err != nil {
		return nil, err
	}
	b := buf.FromBytes(data)
	b.Endpoint = dest
	return &PacketPayload{Target: *dest, Buffer: buf.MultiBuffer{b}}, nil
}

type PacketConnectionReader struct {
	reader  *PacketReader
	payload *PacketPayload
}

func (r *PacketConnectionReader) ReadFrom(p []byte) (n int, addr gonet.Addr, err error) {
	if r.payload == nil || r.payload.Buffer.IsEmpty() {
		r.payload, err = r.reader.ReadMultiBufferWithMetadata()
		if err != nil {
			return
		}
	}

	addr = &gonet.UDPAddr{
		IP:   r.payload.Target.Address.IP(),
		Port: int(r.payload.Target.Port),
	}

	r.payload.Buffer, n = buf.SplitFirstBytes(r.payload.Buffer, p)

	return
}
