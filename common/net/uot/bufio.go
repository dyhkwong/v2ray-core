package uot

import (
	"encoding/binary"
	"io"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
)

const (
	MagicAddress = "sp.v2.udp-over-tcp.arpa"
)

var (
	StreamAddrParser = protocol.NewAddressParser(
		protocol.AddressFamilyByte(0x00, net.AddressFamilyIPv4),
		protocol.AddressFamilyByte(0x01, net.AddressFamilyIPv6),
		protocol.AddressFamilyByte(0x02, net.AddressFamilyDomain),
	)
	RequestAddrParser = protocol.NewAddressParser(
		protocol.AddressFamilyByte(0x01, net.AddressFamilyIPv4),
		protocol.AddressFamilyByte(0x04, net.AddressFamilyIPv6),
		protocol.AddressFamilyByte(0x03, net.AddressFamilyDomain),
	)
)

type Reader struct {
	io.Reader
}

func NewReader(reader io.Reader) *Reader {
	return &Reader{Reader: reader}
}

func NewBufferedReader(reader buf.Reader) *Reader {
	return &Reader{Reader: &buf.BufferedReader{Reader: reader}}
}

func (r *Reader) ReadMultiBuffer() (buf.MultiBuffer, error) {
	addr, port, err := StreamAddrParser.ReadAddressPort(nil, r)
	if err != nil {
		return nil, err
	}
	var length uint16
	if err = binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	b := buf.NewWithSize(int32(length))
	if _, err = io.ReadFull(r, b.Extend(int32(length))); err != nil {
		b.Release()
		return nil, err
	}
	b.Endpoint = &net.Destination{
		Address: addr,
		Port:    port,
		Network: net.Network_UDP,
	}
	return buf.MultiBuffer{b}, nil
}

type Writer struct {
	io.Writer
	Flusher     buf.Flusher
	Destination *net.Destination
	requestSent bool
}

func NewWriter(writer io.Writer, request *net.Destination) *Writer {
	w := &Writer{
		Writer:      writer,
		Destination: request,
	}
	if flusher, ok := writer.(buf.Flusher); ok {
		w.Flusher = flusher
	}
	return w
}

func NewBufferedWriter(writer buf.Writer, destination *net.Destination) *Writer {
	bufferedWriter := buf.NewBufferedWriter(writer)
	return &Writer{
		Writer:      bufferedWriter,
		Flusher:     bufferedWriter,
		Destination: destination,
	}
}

func (w *Writer) WriteMultiBuffer(mb buf.MultiBuffer) error {
	defer buf.ReleaseMulti(mb)
	for _, b := range mb {
		destination := w.Destination
		if b.Endpoint != nil {
			destination = b.Endpoint
		}
		if !w.requestSent {
			w.requestSent = true
			w.Write([]byte{0x00}) // isConnect: false
			if err := RequestAddrParser.WriteAddressPort(w, destination.Address, destination.Port); err != nil {
				return err
			}
		}
		if err := StreamAddrParser.WriteAddressPort(w, destination.Address, destination.Port); err != nil {
			return err
		}
		if err := binary.Write(w, binary.BigEndian, uint16(b.Len())); err != nil {
			return err
		}
		if _, err := w.Write(b.Bytes()); err != nil {
			return err
		}
	}
	if w.Flusher != nil {
		return w.Flusher.Flush()
	}
	return nil
}
