package anytls

import (
	"context"
	"strings"

	anytls "github.com/anytls/sing-anytls"
	"github.com/anytls/sing-anytls/padding"
	"github.com/sagernet/sing/common/bufio"
	"github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/common/uot"

	"github.com/v2fly/v2ray-core/v5/common"
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

var _ network.TCPConnectionHandlerEx = (*Inbound)(nil)

type Inbound struct {
	service *anytls.Service
	email   string
	level   int
}

func NewServer(ctx context.Context, config *ServerConfig) (*Inbound, error) {
	inbound := &Inbound{}
	paddingScheme := padding.DefaultPaddingScheme
	if len(config.PaddingScheme) > 0 {
		paddingScheme = []byte(strings.Join(config.PaddingScheme, "\n"))
	}
	serverConfig := anytls.ServiceConfig{
		PaddingScheme: paddingScheme,
		Handler:       inbound,
		Logger:        singbridge.NewLoggerWrapper(newError),
	}
	for _, user := range config.Users {
		serverConfig.Users = append(serverConfig.Users, anytls.User{
			Name:     user.Email,
			Password: user.Password,
		})
	}
	service, err := anytls.NewService(serverConfig)
	if err != nil {
		return nil, err
	}
	inbound.service = service
	return inbound, nil
}

func (i *Inbound) Network() []net.Network {
	return []net.Network{net.Network_TCP}
}

func (i *Inbound) Process(ctx context.Context, network net.Network, conn internet.Connection, dispatcher routing.Dispatcher) error {
	inbound := session.InboundFromContext(ctx)
	source := metadata.Socksaddr{}
	if inbound.Source.IsValid() {
		source = metadata.ParseSocksaddr(inbound.Source.NetAddr())
	}
	ctx = session.ContextWithDispatcher(ctx, dispatcher)
	return singbridge.ReturnError(i.service.NewConnection(ctx, conn, source, nil))
}

func (i *Inbound) NewConnectionEx(ctx context.Context, conn net.Conn, source metadata.Socksaddr, destination metadata.Socksaddr, onClose network.CloseHandlerFunc) {
	inbound := session.InboundFromContext(ctx)
	inbound.User = &protocol.MemoryUser{
		Email: i.email,
		Level: uint32(i.level),
	}

	if destination == uot.RequestDestination(uot.Version) || destination == uot.RequestDestination(uot.LegacyVersion) {
		request, err := uot.ReadRequest(conn)
		if err != nil {
			newError(err).WriteToLog(session.ExportIDToError(ctx))
			if onClose != nil {
				onClose(err)
			}
			return
		}
		if err := i.handleUDP(ctx, uot.NewConn(conn, *request), source, request.Destination); err != nil {
			newError(err).WriteToLog(session.ExportIDToError(ctx))
			if onClose != nil {
				onClose(err)
			}
			return
		}
	} else {
		if err := i.handleTCP(ctx, conn, source, destination); err != nil {
			newError(err).WriteToLog(session.ExportIDToError(ctx))
			if onClose != nil {
				onClose(err)
			}
			return
		}
	}
	if onClose != nil {
		onClose(nil)
	}
}

func (i *Inbound) handleTCP(ctx context.Context, conn net.Conn, source metadata.Socksaddr, destination metadata.Socksaddr) error {
	ctx = log.ContextWithAccessMessage(ctx, &log.AccessMessage{
		From:   source,
		To:     destination,
		Status: log.AccessAccepted,
		Email:  i.email,
	})
	newError("tunnelling request to tcp:", destination).WriteToLog(session.ExportIDToError(ctx))
	dispatcher := session.DispatcherFromContext(ctx)
	link, err := dispatcher.Dispatch(ctx, singbridge.ToDestination(destination, net.Network_TCP))
	if err != nil {
		return err
	}
	return singbridge.ReturnError(bufio.CopyConn(ctx, conn, singbridge.NewPipeConnWrapper(link)))
}

func (i *Inbound) handleUDP(ctx context.Context, conn network.PacketConn, source metadata.Socksaddr, destination metadata.Socksaddr) error {
	ctx = log.ContextWithAccessMessage(ctx, &log.AccessMessage{
		From:   source,
		To:     destination,
		Status: log.AccessAccepted,
		Email:  i.email,
	})
	newError("tunnelling request to udp:", destination).WriteToLog(session.ExportIDToError(ctx))
	dispatcher := session.DispatcherFromContext(ctx)
	link, err := dispatcher.Dispatch(ctx, singbridge.ToDestination(destination, net.Network_UDP))
	if err != nil {
		return err
	}
	return singbridge.ReturnError(bufio.CopyPacketConn(ctx, conn, singbridge.NewPacketConnWrapper(link, singbridge.ToDestination(destination, net.Network_UDP))))
}
