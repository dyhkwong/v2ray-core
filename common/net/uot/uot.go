// Package uot implements sing-uot (UDP over TCP) client.
package uot

import (
	"encoding/binary"
	"io"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
)

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

var (
	_ buf.Reader = (*reader)(nil)
	_ buf.Writer = (*writer)(nil)
)

const (
	version1 = iota
	version2
	version2Connect
)

const (
	MagicAddressV1 = "sp.udp-over-tcp.arpa"
	MagicAddressV2 = "sp.v2.udp-over-tcp.arpa"
)

const (
	typeBind    = byte(0x00) // isConnect: false
	typeConnect = byte(0x01) // isConnect: true
)

var (
	streamAddrParser = protocol.NewAddressParser(
		protocol.AddressFamilyByte(0x00, net.AddressFamilyIPv4),
		protocol.AddressFamilyByte(0x01, net.AddressFamilyIPv6),
		protocol.AddressFamilyByte(0x02, net.AddressFamilyDomain),
	)
	requestAddrParser = protocol.NewAddressParser(
		protocol.AddressFamilyByte(0x01, net.AddressFamilyIPv4),
		protocol.AddressFamilyByte(0x04, net.AddressFamilyIPv6),
		protocol.AddressFamilyByte(0x03, net.AddressFamilyDomain),
	)
)

type reader struct {
	io.Reader
	version int
	dest    net.Destination // version2Connect
}

func newReader(r buf.Reader, version int) *reader {
	var bufferedReader *buf.BufferedReader
	switch r := r.(type) {
	case *buf.BufferedReader:
		bufferedReader = r
	default:
		bufferedReader = &buf.BufferedReader{Reader: r}
	}
	return &reader{
		Reader:  bufferedReader,
		version: version,
	}
}

func NewReaderV1(r buf.Reader) buf.Reader {
	return newReader(r, version1)
}

func NewReaderV2(reader buf.Reader) buf.Reader {
	return newReader(reader, version2)
}

func NewReaderV2Connect(reader buf.Reader, dest net.Destination) buf.Reader {
	r := newReader(reader, version2Connect)
	r.dest = dest
	return r
}

func (r *reader) ReadMultiBuffer() (buf.MultiBuffer, error) {
	var (
		addr net.Address
		port net.Port
	)
	switch r.version {
	case version1, version2:
		var err error
		addr, port, err = streamAddrParser.ReadAddressPort(nil, r)
		if err != nil {
			return nil, err
		}
	case version2Connect:
		addr, port = r.dest.Address, r.dest.Port
	default:
		panic("undefined")
	}
	var length uint16
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	b := buf.NewWithSize(int32(length))
	if _, err := io.ReadFull(r, b.Extend(int32(length))); err != nil {
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

type writer struct {
	io.Writer
	flusher    buf.Flusher
	version    int
	dest       net.Destination
	headerSent bool // version2 or version2Connect
}

func newWriter(w buf.Writer, version int, dest net.Destination) *writer {
	var bufferedWriter *buf.BufferedWriter
	switch w := w.(type) {
	case *buf.BufferedWriter:
		bufferedWriter = w
	default:
		bufferedWriter = buf.NewBufferedWriter(w)
	}
	return &writer{
		Writer:  bufferedWriter,
		flusher: bufferedWriter,
		version: version,
		dest:    dest,
	}
}

func NewWriterV1(writer buf.Writer, dest net.Destination) buf.Writer {
	return newWriter(writer, version1, dest)
}

func NewWriterV2(writer buf.Writer, dest net.Destination) buf.Writer {
	return newWriter(writer, version2, dest)
}

func NewWriterV2Connect(writer buf.Writer, dest net.Destination) buf.Writer {
	return newWriter(writer, version2Connect, dest)
}

func (w *writer) WriteMultiBuffer(mb buf.MultiBuffer) error {
	defer buf.ReleaseMulti(mb)
	for _, b := range mb {
		dest := w.dest
		if b.Endpoint != nil {
			dest = *b.Endpoint
		}
		switch w.version {
		case version1:
		case version2, version2Connect:
			if w.version == version2Connect && dest != w.dest {
				return newError("bind-like behavior is unsupported in Connect")
			}
			if !w.headerSent {
				w.headerSent = true
				typeByte := typeBind
				if w.version == version2Connect {
					typeByte = typeConnect
				}
				if _, err := w.Write([]byte{typeByte}); err != nil {
					return err
				}
				if err := requestAddrParser.WriteAddressPort(w, w.dest.Address, w.dest.Port); err != nil {
					return err
				}
			}
		default:
			panic("undefined")
		}
		if err := streamAddrParser.WriteAddressPort(w, dest.Address, dest.Port); err != nil {
			return err
		}
		if err := binary.Write(w, binary.BigEndian, uint16(b.Len())); err != nil {
			return err
		}
		if _, err := w.Write(b.Bytes()); err != nil {
			return err
		}
		if err := w.flusher.Flush(); err != nil {
			return err
		}
	}
	return nil
}
