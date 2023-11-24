package udp

import (
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

type MonoDestUDPAddr struct {
	Address net.Address
	Port    net.Port
}

func (*MonoDestUDPAddr) Network() string {
	return "udp"
}

func (a *MonoDestUDPAddr) String() string {
	return a.Address.String() + ":" + a.Port.String()
}

func NewMonoDestUDPConn(conn internet.AbstractPacketConn, addr net.Addr) *MonoDestUDPConn {
	return &MonoDestUDPConn{
		AbstractPacketConn: conn,
		dest:               addr,
	}
}

type MonoDestUDPConn struct {
	internet.AbstractPacketConn
	dest net.Addr
}

func (m *MonoDestUDPConn) ReadMultiBuffer() (buf.MultiBuffer, error) {
	buffer := buf.New()
	buffer.Extend(buf.Size)
	nBytes, addr, err := m.ReadFrom(buffer.Bytes())
	if err != nil {
		buffer.Release()
		return nil, err
	}
	buffer.Resize(0, int32(nBytes))
	if udpAddr, ok := addr.(*net.UDPAddr); ok {
		buffer.Endpoint = &net.Destination{
			Address: net.IPAddress(udpAddr.IP),
			Port:    net.Port(udpAddr.Port),
			Network: net.Network_UDP,
		}
	} else if monoDestUDPAddr, ok := addr.(*MonoDestUDPAddr); ok {
		buffer.Endpoint = &net.Destination{
			Address: monoDestUDPAddr.Address,
			Port:    monoDestUDPAddr.Port,
			Network: net.Network_UDP,
		}
	} else {
		dest, err := net.ParseDestination(addr.Network() + ":" + addr.String())
		if err != nil {
			buffer.Release()
			return nil, newError("unable to parse destination").Base(err)
		}
		buffer.Endpoint = &net.Destination{
			Address: dest.Address,
			Port:    dest.Port,
			Network: net.Network_UDP,
		}
	}
	return buf.MultiBuffer{buffer}, nil
}

func (m *MonoDestUDPConn) WriteMultiBuffer(buffer buf.MultiBuffer) error {
	defer buf.ReleaseMulti(buffer)
	for _, b := range buffer {
		dest := m.dest
		if b.Endpoint != nil {
			if !b.Endpoint.Address.Family().IsDomain() {
				dest = &net.UDPAddr{IP: b.Endpoint.Address.IP(), Port: int(b.Endpoint.Port)}
			} else {
				dest = &MonoDestUDPAddr{Address: b.Endpoint.Address, Port: b.Endpoint.Port}
			}
		}
		_, err := m.WriteTo(b.Bytes(), dest)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MonoDestUDPConn) Read(p []byte) (n int, err error) {
	n, _, err = m.ReadFrom(p)
	return n, err
}

func (m *MonoDestUDPConn) Write(p []byte) (n int, err error) {
	return m.WriteTo(p, m.dest)
}
