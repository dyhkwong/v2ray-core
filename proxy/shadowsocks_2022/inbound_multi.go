//go:build !confonly

package shadowsocks_2022 //nolint:stylecheck

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"sync"

	shadowsocks "github.com/sagernet/sing-shadowsocks"
	"github.com/sagernet/sing-shadowsocks/shadowaead_2022"
	C "github.com/sagernet/sing/common"
	A "github.com/sagernet/sing/common/auth"
	B "github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/bufio"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"

	"github.com/v2fly/v2ray-core/v4/common"
	"github.com/v2fly/v2ray-core/v4/common/buf"
	"github.com/v2fly/v2ray-core/v4/common/log"
	"github.com/v2fly/v2ray-core/v4/common/net"
	"github.com/v2fly/v2ray-core/v4/common/protocol"
	"github.com/v2fly/v2ray-core/v4/common/session"
	"github.com/v2fly/v2ray-core/v4/common/singbridge"
	"github.com/v2fly/v2ray-core/v4/common/uuid"
	"github.com/v2fly/v2ray-core/v4/features/routing"
	"github.com/v2fly/v2ray-core/v4/transport/internet"
)

func init() {
	common.Must(common.RegisterConfig((*MultiUserServerConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewMultiServer(ctx, config.(*MultiUserServerConfig))
	}))
}

type MultiUserInbound struct {
	sync.Mutex
	networks []net.Network
	users    []*User
	service  shadowsocks.MultiService[int]
}

func NewMultiServer(ctx context.Context, config *MultiUserServerConfig) (*MultiUserInbound, error) {
	networks := config.Network
	if len(networks) == 0 {
		networks = []net.Network{
			net.Network_TCP,
			net.Network_UDP,
		}
	}
	inbound := &MultiUserInbound{
		networks: networks,
		users:    config.Users,
	}
	service, err := shadowaead_2022.NewMultiServiceWithPassword[int](config.Method, config.Key, udpTimeout, inbound, nil)
	if err != nil {
		return nil, newError("create service").Base(err)
	}

	for i, user := range config.Users {
		user.Email = strings.ToLower(user.Email)
		if len(user.Email) == 0 {
			u := uuid.New()
			user.Email = "unnamed-user-" + strconv.Itoa(i) + "-" + u.String()
		}
	}
	err = service.UpdateUsersWithPasswords(
		C.MapIndexed(config.Users, func(index int, it *User) int { return index }),
		C.Map(config.Users, func(it *User) string { return it.Key }),
	)
	if err != nil {
		return nil, newError("create service").Base(err)
	}

	inbound.service = service
	return inbound, nil
}

// AddUser implements proxy.UserManager.AddUser().
func (i *MultiUserInbound) AddUser(ctx context.Context, u *protocol.MemoryUser) error {
	account := u.Account.(*MemoryAccount)
	email := strings.ToLower(account.Email)
	if len(email) == 0 {
		u := uuid.New()
		email = "unnamed-user-" + strconv.Itoa(len(i.users)) + "-" + u.String()
		return newError("Email must not be empty.")
	}
	i.Lock()
	defer i.Unlock()
	if slices.ContainsFunc(i.users, func(u *User) bool {
		return u.Email == email
	}) {
		return newError("User ", account.Email, " already exists.")
	}
	i.users = append(i.users, &User{
		Key:   account.Key,
		Email: email,
		Level: account.Level,
	})
	i.service.UpdateUsersWithPasswords(
		C.MapIndexed(i.users, func(index int, it *User) int { return index }),
		C.Map(i.users, func(it *User) string { return it.Key }),
	)

	return nil
}

// RemoveUser implements proxy.UserManager.RemoveUser().
func (i *MultiUserInbound) RemoveUser(ctx context.Context, email string) error {
	email = strings.ToLower(email)
	if len(email) == 0 {
		return newError("Email must not be empty.")
	}
	i.Lock()
	defer i.Unlock()
	if !slices.ContainsFunc(i.users, func(u *User) bool {
		return u.Email == email
	}) {
		return newError("User ", email, " does not exist.")
	}
	i.users = slices.DeleteFunc(i.users, func(u *User) bool {
		return u.Email == email
	})
	i.service.UpdateUsersWithPasswords(
		C.MapIndexed(i.users, func(index int, it *User) int { return index }),
		C.Map(i.users, func(it *User) string { return it.Key }),
	)
	return nil
}

func (i *MultiUserInbound) Network() []net.Network {
	return i.networks
}

func (i *MultiUserInbound) Process(ctx context.Context, network net.Network, connection internet.Connection, dispatcher routing.Dispatcher) error {
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
		pc := bufio.NewUnbindPacketConn(connection)
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

func (i *MultiUserInbound) NewConnection(ctx context.Context, conn net.Conn, metadata M.Metadata) error {
	inbound := session.InboundFromContext(ctx)
	userInt, _ := A.UserFromContext[int](ctx)
	user := i.users[userInt]
	inbound.User = &protocol.MemoryUser{
		Email: user.Email,
		Level: uint32(user.Level),
	}
	ctx = log.ContextWithAccessMessage(ctx, &log.AccessMessage{
		From:   metadata.Source,
		To:     metadata.Destination,
		Status: log.AccessAccepted,
		Email:  user.Email,
	})
	newError("tunnelling request to tcp:", metadata.Destination).WriteToLog(session.ExportIDToError(ctx))
	dispatcher := session.DispatcherFromContext(ctx)
	link, err := dispatcher.Dispatch(ctx, singbridge.ToDestination(metadata.Destination, net.Network_TCP))
	if err != nil {
		return err
	}
	return singbridge.ReturnError(bufio.CopyConn(ctx, conn, singbridge.NewPipeConnWrapper(link)))
}

func (i *MultiUserInbound) NewPacketConnection(ctx context.Context, conn N.PacketConn, metadata M.Metadata) error {
	inbound := session.InboundFromContext(ctx)
	userInt, _ := A.UserFromContext[int](ctx)
	user := i.users[userInt]
	inbound.User = &protocol.MemoryUser{
		Email: user.Email,
		Level: uint32(user.Level),
	}
	ctx = log.ContextWithAccessMessage(ctx, &log.AccessMessage{
		From:   metadata.Source,
		To:     metadata.Destination,
		Status: log.AccessAccepted,
		Email:  user.Email,
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

func (i *MultiUserInbound) NewError(ctx context.Context, err error) {
	if E.IsClosed(err) {
		return
	}
	newError(err).AtWarning().WriteToLog()
}
