package ipc

import (
	"encoding/binary"
	"io"
	"os"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
)

var addrParser = protocol.NewAddressParser(
	protocol.AddressFamilyByte(0x01, net.AddressFamilyIPv4),
	protocol.AddressFamilyByte(0x04, net.AddressFamilyIPv6),
	protocol.AddressFamilyByte(0x03, net.AddressFamilyDomain),
)

const (
	commandTCP byte = 1
	commandUDP byte = 3
)

type packetWriter struct {
	io.Writer
}

func (w *packetWriter) writeMultiBufferWithMetadata(mb buf.MultiBuffer, dest net.Destination) error {
	defer buf.ReleaseMulti(mb)
	for _, b := range mb {
		if b == nil {
			continue
		}
		_, err := w.writePacket(b.Bytes(), dest)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *packetWriter) writePacket(payload []byte, dest net.Destination) (int, error) {
	var addrPortLen int32
	switch dest.Address.Family() {
	case net.AddressFamilyDomain:
		if len(dest.Address.Domain()) > 255 {
			return 0, os.ErrInvalid
		}
		addrPortLen = 1 + 1 + int32(len(dest.Address.Domain())) + 2
	case net.AddressFamilyIPv4:
		addrPortLen = 1 + 4 + 2
	case net.AddressFamilyIPv6:
		addrPortLen = 1 + 16 + 2
	default:
		panic(os.ErrInvalid)
	}
	length := len(payload)
	lengthBuf := [2]byte{}
	binary.BigEndian.PutUint16(lengthBuf[:], uint16(length))
	b := buf.NewWithSize(addrPortLen + 2 + 2 + int32(length))
	defer b.Release()
	err := addrParser.WriteAddressPort(b, dest.Address, dest.Port)
	if err != nil {
		return 0, err
	}
	if _, err := b.Write(lengthBuf[:]); err != nil {
		return 0, err
	}
	if _, err := b.Write(payload); err != nil {
		return 0, err
	}
	_, err = w.Write(b.Bytes())
	if err != nil {
		return 0, err
	}
	return length, nil
}

type connReader struct {
	io.Reader
	target       net.Destination
	headerParsed bool
}

func (c *connReader) parseHeader() error {
	var command [1]byte
	if _, err := io.ReadFull(c.Reader, command[:]); err != nil {
		return err
	}
	var network net.Network
	switch command[0] {
	case commandTCP:
		network = net.Network_TCP
	case commandUDP:
		network = net.Network_UDP
	default:
		return os.ErrInvalid
	}
	addr, port, err := addrParser.ReadAddressPort(nil, c.Reader)
	if err != nil {
		return err
	}
	c.target = net.Destination{Network: network, Address: addr, Port: port}
	c.headerParsed = true
	return nil
}

func (c *connReader) Read(p []byte) (int, error) {
	if !c.headerParsed {
		err := c.parseHeader()
		if err != nil {
			return 0, err
		}
	}
	return c.Reader.Read(p)
}

func (c *connReader) ReadMultiBuffer() (buf.MultiBuffer, error) {
	b := buf.New()
	_, err := b.ReadFrom(c)
	if err != nil {
		b.Release()
		return nil, err
	}
	return buf.MultiBuffer{b}, err
}

type packetPayload struct {
	target net.Destination
	buffer buf.MultiBuffer
}

type packetReader struct {
	io.Reader
}

func (r *packetReader) readMultiBufferWithMetadata() (*packetPayload, error) {
	addr, port, err := addrParser.ReadAddressPort(nil, r)
	if err != nil {
		return nil, err
	}
	var lengthBuf [2]byte
	_, err = io.ReadFull(r, lengthBuf[:])
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint16(lengthBuf[:])
	b := buf.NewWithSize(int32(length))
	dest := net.UDPDestination(addr, port)
	b.Endpoint = &dest
	_, err = b.ReadFullFrom(r, int32(length))
	if err != nil {
		return nil, err
	}
	return &packetPayload{target: dest, buffer: buf.MultiBuffer{b}}, nil
}
