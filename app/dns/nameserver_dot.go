package dns

import (
	"context"
	"crypto/tls"
	"net/url"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/net/cnc"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

// NewDoTNameServer creates DOT server object for remote resolving.
func NewDoTNameServer(url *url.URL, dispatcher routing.Dispatcher) (*TCPNameServer, error) {
	s, err := baseTCPNameServer(url, "DOT", net.Port(853), "tls")
	if err != nil {
		return nil, err
	}

	s.dial = func(ctx context.Context) (net.Conn, error) {
		link, err := dispatcher.Dispatch(ctx, s.destination)
		if err != nil {
			return nil, err
		}

		conn := cnc.NewConnection(
			cnc.ConnectionInputMulti(link.Writer),
			cnc.ConnectionOutputMulti(link.Reader),
		)

		return tls.Client(conn, &tls.Config{
			NextProtos: []string{"dot"},
			ServerName: func() string {
				switch s.destination.Address.Family() {
				case net.AddressFamilyIPv4, net.AddressFamilyIPv6:
					return s.destination.Address.IP().String()
				case net.AddressFamilyDomain:
					return s.destination.Address.Domain()
				default:
					panic("unknown address family")
				}
			}(),
		}), nil
	}

	return s, nil
}

// NewDoTLocalNameServer creates DOT client object for local resolving
func NewDoTLocalNameServer(url *url.URL) (*TCPNameServer, error) {
	s, err := baseTCPNameServer(url, "DOTL", net.Port(853), "tls")
	if err != nil {
		return nil, err
	}

	s.dial = func(ctx context.Context) (net.Conn, error) {
		conn, err := internet.DialSystem(ctx, s.destination, nil)
		if err != nil {
			return nil, err
		}

		return tls.Client(conn, &tls.Config{
			NextProtos: []string{"dot"},
			ServerName: func() string {
				switch s.destination.Address.Family() {
				case net.AddressFamilyIPv4, net.AddressFamilyIPv6:
					return s.destination.Address.IP().String()
				case net.AddressFamilyDomain:
					return s.destination.Address.Domain()
				default:
					panic("unknown address family")
				}
			}(),
		}), nil
	}

	return s, nil
}
