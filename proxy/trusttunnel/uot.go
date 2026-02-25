package trusttunnel

import (
	"bytes"
	"encoding/binary"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
)

var (
	_ buf.Writer = (*uotWriter)(nil)
	_ buf.Reader = (*uotReader)(nil)

	defaultH1UserAgent  = "Go-http-client/1.1" // net/http http.Transport
	defaultH2UserAgent  = "Go-http-client/2.0" // net/http http.Transport
	defaultH3UserAgent  = "quic-go HTTP/3"     // github.com/quic-go/quic-go/http3 http3.Transport
	uotMagicAddress     = "_udp2"
	ipv4Padding         = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	ipv6LoopBackAddress = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	zeroAddressPort     = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
)

type uotReader struct {
	net.Conn
	dest   net.Destination
	destIP net.Address
}

func newUoTReader(conn net.Conn, dest net.Destination, destIP net.Address) *uotReader {
	return &uotReader{
		Conn:   conn,
		dest:   dest,
		destIP: destIP,
	}
}

func (r *uotReader) ReadMultiBuffer() (buf.MultiBuffer, error) {
	// Length
	var length uint32
	err := binary.Read(r, binary.BigEndian, &length)
	if err != nil {
		return nil, err
	}
	if length < 36 {
		return nil, newError("length too short")
	}
	dest := net.Destination{
		Network: net.Network_UDP,
	}
	payload := buf.NewWithSize(int32(length))
	_, err = payload.ReadFullFrom(r, int32(length))
	if err != nil {
		payload.Release()
		return nil, err
	}
	// Source Address
	var addr []byte
	addr, err = payload.ReadBytes(16)
	if err != nil {
		payload.Release()
		return nil, err
	}
	if bytes.Equal(addr[:12], ipv4Padding) && !bytes.Equal(addr, ipv6LoopBackAddress) {
		// "An address is IPv4 if the first 12 bytes are zero AND it's not `::1` (IPv6 loopback)."
		dest.Address = net.IPAddress(addr[12:])
	} else {
		dest.Address = net.IPAddress(addr)
	}
	// Source Port
	err = binary.Read(payload, binary.BigEndian, &dest.Port)
	if err != nil {
		payload.Release()
		return nil, err
	}
	// Destination Address, Destination Port
	_, err = payload.ReadBytes(18)
	if err != nil {
		payload.Release()
		return nil, err
	}
	// Payload
	if dest.Address == r.destIP {
		dest.Address = r.dest.Address
	}
	payload.Endpoint = &dest
	return buf.MultiBuffer{payload}, nil
}

type uotWriter struct {
	net.Conn
	dest      net.Destination
	destIP    net.Address
	userAgent string
	resolver  func(domain string) (net.Address, error)
}

func newUoTWriter(conn net.Conn, dest net.Destination, destIP net.Address, userAgent string, resolver func(domain string) (net.Address, error)) *uotWriter {
	return &uotWriter{
		Conn:      conn,
		dest:      dest,
		destIP:    destIP,
		userAgent: userAgent,
		resolver:  resolver,
	}
}

func (w *uotWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	defer buf.ReleaseMulti(mb)
	if len(w.userAgent) > 255 {
		return newError("App Name too long")
	}
	for _, b := range mb {
		dest := w.dest
		if b.Endpoint != nil {
			dest = *b.Endpoint
		}
		if dest.Address.Family().IsDomain() {
			if dest.Address == w.dest.Address {
				dest.Address = w.destIP
			} else {
				if ip, err := w.resolver(dest.Address.Domain()); err != nil {
					newError(err).AtError().WriteToLog()
					continue
				} else {
					dest.Address = ip
				}
			}
		}
		length := uint32(b.Len()) + 37 + uint32(len(w.userAgent))
		payload := buf.NewWithSize(int32(length))
		// Length
		err := binary.Write(payload, binary.BigEndian, length)
		if err != nil {
			payload.Release()
			return err
		}
		// Source Address, Source Port
		_, err = payload.Write(zeroAddressPort)
		if err != nil {
			payload.Release()
			return err
		}
		// Destination Address
		if dest.Address.Family().IsIPv4() {
			_, err = payload.Write(ipv4Padding)
			if err != nil {
				payload.Release()
				return err
			}
		}
		_, err = payload.Write(dest.Address.IP())
		if err != nil {
			payload.Release()
			return err
		}
		// Destination Port
		err = binary.Write(payload, binary.BigEndian, dest.Port)
		if err != nil {
			payload.Release()
			return err
		}
		// App Name Length
		_, err = payload.Write([]byte{byte(len(w.userAgent))})
		if err != nil {
			payload.Release()
			return err
		}
		// App Name
		if len(w.userAgent) > 0 {
			_, err = payload.Write([]byte(w.userAgent))
			if err != nil {
				payload.Release()
				return err
			}
		}
		// Payload
		_, err = payload.Write(b.Bytes())
		if err != nil {
			payload.Release()
			return err
		}
		_, err = w.Write(payload.Bytes())
		payload.Release()
		if err != nil {
			return err
		}
	}
	return nil
}
