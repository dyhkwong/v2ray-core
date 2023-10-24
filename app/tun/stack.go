//go:build !confonly
// +build !confonly

package tun

import (
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
)

type StackOption func(*stack.Stack) error

func (t *TUN) CreateStack(linkedEndpoint stack.LinkEndpoint) (*stack.Stack, error) {
	s := stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			ipv4.NewProtocol,
			ipv6.NewProtocol,
		},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
			udp.NewProtocol,
			icmp.NewProtocol4,
			icmp.NewProtocol6,
		},
	})

	nicID := tcpip.NICID(s.UniqueID())

	opts := []StackOption{
		HandleTCP(handleTCP),
		HandleUDP(handleUDP),

		CreateNIC(nicID, linkedEndpoint),
		AddProtocolAddress(nicID, t.config.Ips),
		SetRouteTable(nicID, t.config.Routes),
		SetPromiscuousMode(nicID, t.config.EnablePromiscuousMode),
		SetSpoofing(nicID, t.config.EnableSpoofing),
	}

	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}

	return s, nil
}
