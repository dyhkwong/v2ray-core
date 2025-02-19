package anytls

import (
	"context"
	"slices"
	"strings"
	"sync"

	anytls "github.com/anytls/sing-anytls"
	"github.com/anytls/sing-anytls/padding"
	"github.com/sagernet/sing/common/auth"
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
	sync.Mutex
	users   []*User
	service *anytls.Service
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
	inbound.users = config.Users
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
	inbound := session.InboundFromContext(ctx)
	userInt, _ := auth.UserFromContext[int](ctx)
	user := i.users[userInt]
	inbound.User = &protocol.MemoryUser{
		Email: user.Email,
		Level: uint32(user.Level),
	}
	ctx = log.ContextWithAccessMessage(ctx, &log.AccessMessage{
		From:   source,
		To:     destination,
		Status: log.AccessAccepted,
		Email:  user.Email,
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
	inbound := session.InboundFromContext(ctx)
	idx, _ := auth.UserFromContext[int](ctx)
	user := i.users[idx]
	inbound.User = &protocol.MemoryUser{
		Email: user.Email,
		Level: uint32(user.Level),
	}
	ctx = log.ContextWithAccessMessage(ctx, &log.AccessMessage{
		From:   source,
		To:     destination,
		Status: log.AccessAccepted,
		Email:  user.Email,
	})
	newError("tunnelling request to udp:", destination).WriteToLog(session.ExportIDToError(ctx))
	dispatcher := session.DispatcherFromContext(ctx)
	link, err := dispatcher.Dispatch(ctx, singbridge.ToDestination(destination, net.Network_UDP))
	if err != nil {
		return err
	}
	return singbridge.ReturnError(bufio.CopyPacketConn(ctx, conn, singbridge.NewPacketConnWrapper(link, singbridge.ToDestination(destination, net.Network_UDP))))
}

// AddUser implements proxy.UserManager.AddUser().
func (i *Inbound) AddUser(ctx context.Context, user *protocol.MemoryUser) error {
	account := user.Account.(*MemoryAccount)
	if account.Email == "" {
		return newError("Email must not be empty.")
	}
	i.Lock()
	defer i.Unlock()
	if idx := slices.IndexFunc(i.users, func(u *User) bool {
		return u.Email == account.Email
	}); idx >= 0 {
		return newError("User ", account.Email, " already exists.")
	}
	i.users = append(i.users, &User{
		Password: account.Password,
		Email:    account.Email,
		Level:    account.Level,
	})
	var users []anytls.User
	for _, u := range i.users {
		users = append(users, anytls.User{
			Name:     u.Email,
			Password: u.Password,
		})
	}
	i.service.UpdateUsers(users)
	return nil
}

// RemoveUser implements proxy.UserManager.RemoveUser().
func (i *Inbound) RemoveUser(ctx context.Context, email string) error {
	if email == "" {
		return newError("Email must not be empty.")
	}
	i.Lock()
	defer i.Unlock()
	i.users = slices.DeleteFunc(i.users, func(u *User) bool {
		return u.Email == email
	})
	var users []anytls.User
	for _, user := range i.users {
		users = append(users, anytls.User{
			Name:     user.Email,
			Password: user.Password,
		})
	}
	i.service.UpdateUsers(users)
	return nil
}
