package shadowsocks_2022 //nolint:stylecheck

import (
	"context"
	"io"
	"strconv"

	"github.com/sagernet/sing-shadowsocks/shadowaead_2022"
	C "github.com/sagernet/sing/common"
	A "github.com/sagernet/sing/common/auth"
	B "github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/bufio"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/app/proxyman"
	app_inbound "github.com/v2fly/v2ray-core/v5/app/proxyman/inbound"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/log"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/singbridge"
	"github.com/v2fly/v2ray-core/v5/common/task"
	"github.com/v2fly/v2ray-core/v5/common/uuid"
	features_inbound "github.com/v2fly/v2ray-core/v5/features/inbound"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/proxy/sip003"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

func init() {
	common.Must(common.RegisterConfig((*RelayServerConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewRelayServer(ctx, config.(*RelayServerConfig))
	}))
}

type RelayInbound struct {
	networks     []net.Network
	destinations []*RelayDestination
	service      *shadowaead_2022.RelayService[int]

	tag            string
	pluginTag      string
	plugin         sip003.Plugin
	pluginOverride net.Destination
	receiverPort   int
}

func (i *RelayInbound) Initialize(self features_inbound.Handler) {
	i.tag = self.Tag()
}

func (i *RelayInbound) Close() error {
	if i.plugin != nil {
		return i.plugin.Close()
	}
	return nil
}

func NewRelayServer(ctx context.Context, config *RelayServerConfig) (*RelayInbound, error) {
	networks := config.Network
	if len(networks) == 0 {
		networks = []net.Network{
			net.Network_TCP,
			net.Network_UDP,
		}
	}
	inbound := &RelayInbound{
		networks:     networks,
		destinations: config.Destinations,
	}
	service, err := shadowaead_2022.NewRelayServiceWithPassword[int](config.Method, config.Key, udpTimeout, inbound)
	if err != nil {
		return nil, newError("create service").Base(err)
	}

	for i, destination := range config.Destinations {
		if destination.Email == "" {
			u := uuid.New()
			destination.Email = "unnamed-destination-" + strconv.Itoa(i) + "-" + u.String()
		}
	}
	err = service.UpdateUsersWithPasswords(
		C.MapIndexed(config.Destinations, func(index int, it *RelayDestination) int { return index }),
		C.Map(config.Destinations, func(it *RelayDestination) string { return it.Key }),
		C.Map(config.Destinations, func(it *RelayDestination) M.Socksaddr {
			return singbridge.ToSocksAddr(net.Destination{
				Address: it.Address.AsAddress(),
				Port:    net.Port(it.Port),
			})
		}),
	)
	if err != nil {
		return nil, newError("create service").Base(err)
	}
	inbound.service = service

	if config.Plugin != "" {
		var plugin sip003.Plugin
		if pc := sip003.Plugins[config.Plugin]; pc != nil {
			plugin = pc()
		} else if sip003.PluginLoader == nil {
			return nil, newError("plugin loader not registered")
		} else {
			plugin = sip003.PluginLoader(config.Plugin)
		}
		port, err := net.GetFreePort()
		if err != nil {
			return nil, newError("failed to get free port for sip003 plugin").Base(err)
		}
		inbound.receiverPort, err = net.GetFreePort()
		if err != nil {
			return nil, newError("failed to get free port for sip003 plugin receiver").Base(err)
		}
		u := uuid.New()
		tag := "v2ray.system.shadowsocks-inbound-plugin-receiver." + u.String()
		inbound.pluginTag = tag
		handler, err := app_inbound.NewAlwaysOnInboundHandlerWithProxy(ctx, tag, &proxyman.ReceiverConfig{
			Listen:    net.NewIPOrDomain(net.LocalHostIP),
			PortRange: net.SinglePortRange(net.Port(inbound.receiverPort)),
		}, inbound, true, false)
		if err != nil {
			return nil, newError("failed to create sip003 plugin inbound").Base(err)
		}
		v := core.MustFromContext(ctx)
		inboundManager := v.GetFeature(features_inbound.ManagerType()).(features_inbound.Manager)
		if err := inboundManager.AddHandler(ctx, handler); err != nil {
			return nil, newError("failed to add sip003 plugin inbound").Base(err)
		}
		inbound.pluginOverride = net.Destination{
			Network: net.Network_TCP,
			Address: net.LocalHostIP,
			Port:    net.Port(port),
		}
		if err := plugin.Init(net.LocalHostIP.String(), strconv.Itoa(inbound.receiverPort), net.LocalHostIP.String(), strconv.Itoa(port), config.PluginOpts, config.PluginArgs, config.PluginWorkingDir); err != nil {
			return nil, newError("failed to start plugin").Base(err)
		}
		inbound.plugin = plugin
	}

	return inbound, nil
}

func (i *RelayInbound) Network() []net.Network {
	return i.networks
}

func (i *RelayInbound) Process(ctx context.Context, network net.Network, connection internet.Connection, dispatcher routing.Dispatcher) error {
	inbound := session.InboundFromContext(ctx)

	if network == net.Network_TCP && i.plugin != nil {
		if inbound.Tag != i.pluginTag {
			dest, err := internet.Dial(ctx, i.pluginOverride, nil)
			if err != nil {
				return newError("failed to handle request to shadowsocks SIP003 plugin").Base(err)
			}
			if err := task.Run(ctx, func() error {
				_, err := io.Copy(connection, dest)
				return err
			}, func() error {
				_, err := io.Copy(dest, connection)
				return err
			}); err != nil {
				return newError("connection ends").Base(err)
			}
			return nil
		}
		inbound.Tag = i.tag
	}

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

func (i *RelayInbound) NewConnection(ctx context.Context, conn net.Conn, metadata M.Metadata) error {
	inbound := session.InboundFromContext(ctx)
	userInt, _ := A.UserFromContext[int](ctx)
	user := i.destinations[userInt]
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

func (i *RelayInbound) NewPacketConnection(ctx context.Context, conn N.PacketConn, metadata M.Metadata) error {
	inbound := session.InboundFromContext(ctx)
	userInt, _ := A.UserFromContext[int](ctx)
	user := i.destinations[userInt]
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

func (i *RelayInbound) NewError(ctx context.Context, err error) {
	if singbridge.ReturnError(err) != nil {
		newError(err).AtWarning().WriteToLog(session.ExportIDToError(ctx))
	}
}
