package shadowsocks_2022 //nolint:stylecheck

import (
	"context"

	shadowsocks "github.com/sagernet/sing-shadowsocks"
	"github.com/sagernet/sing-shadowsocks/shadowaead_2022"
	B "github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/bufio"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/log"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/singbridge"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

func init() {
	common.Must(common.RegisterConfig((*ServerConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewServer(ctx, config.(*ServerConfig))
	}))
}

const (
	udpTimeout = 500
)

type Inbound struct {
	networks []net.Network
	service  shadowsocks.Service
	email    string
	level    int
}

func NewServer(ctx context.Context, config *ServerConfig) (*Inbound, error) {
	networks := config.Network
	if len(networks) == 0 {
		networks = []net.Network{
			net.Network_TCP,
			net.Network_UDP,
		}
	}
	inbound := &Inbound{
		networks: networks,
		email:    config.Email,
		level:    int(config.Level),
	}
	service, err := shadowaead_2022.NewServiceWithPassword(config.Method, config.Key, udpTimeout, inbound, nil)
	if err != nil {
		return nil, newError("create service").Base(err)
	}
	inbound.service = service
	return inbound, nil
}

func (i *Inbound) Network() []net.Network {
	return i.networks
}

func (i *Inbound) Process(ctx context.Context, network net.Network, connection internet.Connection, dispatcher routing.Dispatcher) error {
	inbound := session.InboundFromContext(ctx)

	var metadata M.Metadata
	if inbound.Source.IsValid() {
		metadata.Source = M.ParseSocksaddr(inbound.Source.NetAddr())
	}

	ctx = session.ContextWithDispatcher(ctx, dispatcher)

	if network == net.Network_TCP {
		return singbridge.ReturnError(i.service.NewConnection(ctx, connection, metadata))
	} else {
		reader := buf.NewReader(connection)
		pc := &natPacketConn{connection}
		for {
			mb, err := reader.ReadMultiBuffer()
			if err != nil {
				buf.ReleaseMulti(mb)
				return singbridge.ReturnError(err)
			}
			for _, buffer := range mb {
				packet := B.As(buffer.Bytes()).ToOwned()
				buffer.Release()
				err = i.service.NewPacket(ctx, pc, packet, metadata)
				if err != nil {
					packet.Release()
					buf.ReleaseMulti(mb)
					return err
				}
			}
		}
	}
}

func (i *Inbound) NewConnection(ctx context.Context, conn net.Conn, metadata M.Metadata) error {
	inbound := session.InboundFromContext(ctx)
	inbound.User = &protocol.MemoryUser{
		Email: i.email,
		Level: uint32(i.level),
	}
	ctx = log.ContextWithAccessMessage(ctx, &log.AccessMessage{
		From:   metadata.Source,
		To:     metadata.Destination,
		Status: log.AccessAccepted,
		Email:  i.email,
	})
	newError("tunnelling request to tcp:", metadata.Destination).WriteToLog(session.ExportIDToError(ctx))
	dispatcher := session.DispatcherFromContext(ctx)
	link, err := dispatcher.Dispatch(ctx, singbridge.ToDestination(metadata.Destination, net.Network_TCP))
	if err != nil {
		return err
	}
	return singbridge.ReturnError(bufio.CopyConn(ctx, conn, singbridge.NewPipeConnWrapper(link)))
}

func (i *Inbound) NewPacketConnection(ctx context.Context, conn N.PacketConn, metadata M.Metadata) error {
	inbound := session.InboundFromContext(ctx)
	inbound.User = &protocol.MemoryUser{
		Email: i.email,
		Level: uint32(i.level),
	}
	ctx = log.ContextWithAccessMessage(ctx, &log.AccessMessage{
		From:   metadata.Source,
		To:     metadata.Destination,
		Status: log.AccessAccepted,
		Email:  i.email,
	})
	newError("tunnelling request to udp:", metadata.Destination).WriteToLog(session.ExportIDToError(ctx))
	dispatcher := session.DispatcherFromContext(ctx)
	destination := singbridge.ToDestination(metadata.Destination, net.Network_UDP)
	link, err := dispatcher.Dispatch(ctx, destination)
	if err != nil {
		return err
	}
	return singbridge.ReturnError(bufio.CopyPacketConn(ctx, conn, singbridge.NewPacketConnWrapper(link, destination)))
}

func (i *Inbound) NewError(ctx context.Context, err error) {
	if singbridge.ReturnError(err) != nil {
		newError(err).AtWarning().WriteToLog(session.ExportIDToError(ctx))
	}
}

type natPacketConn struct {
	net.Conn
}

func (c *natPacketConn) ReadPacket(buffer *B.Buffer) (addr M.Socksaddr, err error) {
	_, err = buffer.ReadFrom(c)
	return addr, err
}

func (c *natPacketConn) WritePacket(buffer *B.Buffer, addr M.Socksaddr) error {
	_, err := buffer.WriteTo(c)
	return err
}
