//go:build !confonly
// +build !confonly

package tun

import (
	"context"

	"github.com/v2fly/v2ray-core/v4/common/buf"
	"github.com/v2fly/v2ray-core/v4/common/net"
	"github.com/v2fly/v2ray-core/v4/common/net/packetaddr"
	udp_proto "github.com/v2fly/v2ray-core/v4/common/protocol/udp"
	"github.com/v2fly/v2ray-core/v4/common/session"
	"github.com/v2fly/v2ray-core/v4/features/policy"
	"github.com/v2fly/v2ray-core/v4/features/routing"
	"github.com/v2fly/v2ray-core/v4/transport/internet/udp"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	gvisor_udp "gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
)

type UDPHandler struct {
	ctx           context.Context
	dispatcher    routing.Dispatcher
	policyManager policy.Manager
	config        *Config

	stack *stack.Stack
}

type udpConn struct {
	*gonet.UDPConn
	id stack.TransportEndpointID
}

func (c *udpConn) ID() *stack.TransportEndpointID {
	return &c.id
}

func HandleUDP(handle func(UDPConn)) StackOption {
	return func(s *stack.Stack) error {
		udpForwarder := gvisor_udp.NewForwarder(s, func(r *gvisor_udp.ForwarderRequest) {
			wg := new(waiter.Queue)
			linkedEndpoint, err := r.CreateEndpoint(wg)
			if err != nil {
				// TODO: log
				return
			}

			udpConn := &udpConn{
				UDPConn: gonet.NewUDPConn(s, wg, linkedEndpoint),
				id:      r.ID(),
			}

			handle(udpConn)
		})
		s.SetTransportProtocolHandler(gvisor_udp.ProtocolNumber, udpForwarder.HandlePacket)
		return nil
	}
}

func (h *UDPHandler) HandleQueue(ch chan UDPConn) {
	for {
		select {
		case <-h.ctx.Done():
			return
		case conn := <-ch:
			if err := h.Handle(conn); err != nil {
				newError(err).AtError().WriteToLog(session.ExportIDToError(h.ctx))
			}
		}
	}
}

func (h *UDPHandler) Handle(conn UDPConn) error {
	ctx := session.ContextWithInbound(h.ctx, &session.Inbound{Tag: h.config.Tag})
	packetConn := conn.(net.PacketConn)

	udpDispatcherConstructor := udp.NewSplitDispatcher
	switch h.config.PacketEncoding {
	case packetaddr.PacketAddrType_None:
		break
	case packetaddr.PacketAddrType_Packet:
		packetAddrDispatcherFactory := udp.NewPacketAddrDispatcherCreator(ctx)
		udpDispatcherConstructor = packetAddrDispatcherFactory.NewPacketAddrDispatcher
	}

	udpServer := udpDispatcherConstructor(h.dispatcher, func(ctx context.Context, packet *udp_proto.Packet) {
		if _, err := packetConn.WriteTo(packet.Payload.Bytes(), &net.UDPAddr{
			IP:   packet.Source.Address.IP(),
			Port: int(packet.Source.Port),
		}); err != nil {
			newError("failed to write UDP packet").Base(err).WriteToLog()
		}
	})

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			var buffer [2048]byte
			n, addr, err := packetConn.ReadFrom(buffer[:])
			if err != nil {
				return newError("failed to read UDP packet").Base(err)
			}
			currentPacketCtx := ctx

			udpServer.Dispatch(currentPacketCtx, net.DestinationFromAddr(addr), buf.FromBytes(buffer[:n]))
		}
	}
}
