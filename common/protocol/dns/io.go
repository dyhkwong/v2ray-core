package dns

import (
	"encoding/binary"
	"io"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/serial"
)

func PackMessage(msg *dnsmessage.Message) (*buf.Buffer, error) {
	buffer := buf.New()
	rawBytes := buffer.Extend(buf.Size)
	packed, err := msg.AppendPack(rawBytes[:0])
	if err != nil {
		buffer.Release()
		return nil, err
	}
	buffer.Resize(0, int32(len(packed)))
	return buffer, nil
}

type MessageReader interface {
	ReadMessage() (*buf.Buffer, error)
}

type UDPReader struct {
	io.Reader
}

// ReadMessage implements MessageReader.
func (r *UDPReader) ReadMessage() (*buf.Buffer, error) {
	b := buf.New()
	if _, err := b.ReadFrom(r.Reader); err != nil {
		b.Release()
		return nil, err
	}
	return b, nil
}

func (r *UDPReader) Interrupt() {
	common.Interrupt(r.Reader)
}

// Close implements common.Closable.
func (r *UDPReader) Close() error {
	return common.Close(r.Reader)
}

type TCPReader struct {
	io.Reader
}

func (r *TCPReader) ReadMessage() (*buf.Buffer, error) {
	size, err := serial.ReadUint16(r.Reader)
	if err != nil {
		return nil, err
	}
	b := buf.NewWithSize(int32(size))
	if _, err := b.ReadFullFrom(r.Reader, int32(size)); err != nil {
		b.Release()
		return nil, err
	}
	return b, nil
}

func (r *TCPReader) Interrupt() {
	common.Interrupt(r.Reader)
}

func (r *TCPReader) Close() error {
	return common.Close(r.Reader)
}

type MessageWriter interface {
	WriteMessage(msg *buf.Buffer) error
}

type UDPWriter struct {
	buf.Writer
}

func (w *UDPWriter) WriteMessage(b *buf.Buffer) error {
	return w.WriteMultiBuffer(buf.MultiBuffer{b})
}

type TCPWriter struct {
	buf.Writer
}

func (w *TCPWriter) WriteMessage(b *buf.Buffer) error {
	if b.IsEmpty() {
		return nil
	}

	mb := make(buf.MultiBuffer, 0, 2)

	size := buf.New()
	binary.BigEndian.PutUint16(size.Extend(2), uint16(b.Len()))
	mb = append(mb, size, b)
	return w.WriteMultiBuffer(mb)
}
