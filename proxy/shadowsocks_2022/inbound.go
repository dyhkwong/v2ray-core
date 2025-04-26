package shadowsocks_2022 //nolint:stylecheck

import (
	"context"
	"io"
	"strconv"

	shadowsocks "github.com/sagernet/sing-shadowsocks"
	"github.com/sagernet/sing-shadowsocks/shadowaead_2022"
	B "github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/bufio"
	E "github.com/sagernet/sing/common/exceptions"
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
	"github.com/v2fly/v2ray-core/v5/common/task"
	"github.com/v2fly/v2ray-core/v5/common/uuid"
	features_inbound "github.com/v2fly/v2ray-core/v5/features/inbound"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/proxy/sip003"
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

	tag            string
	pluginTag      string
	plugin         sip003.Plugin
	pluginOverride net.Destination
	receiverPort   int
}

func (i *Inbound) Initialize(self features_inbound.Handler) {
	i.tag = self.Tag()
}

func (i *Inbound) Close() error {
	if i.plugin != nil {
		return i.plugin.Close()
	}
	return nil
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
		}, inbound, true)
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

func (i *Inbound) Network() []net.Network {
	return i.networks
}

func (i *Inbound) Process(ctx context.Context, network net.Network, connection internet.Connection, dispatcher routing.Dispatcher) error {
	inbound := session.InboundFromContext(ctx)

	if i.plugin != nil {
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
		return returnError(i.service.NewConnection(ctx, connection, metadata))
	} else {
		reader := buf.NewReader(connection)
		pc := &natPacketConn{connection}
		for {
			mb, err := reader.ReadMultiBuffer()
			if err != nil {
				buf.ReleaseMulti(mb)
				return returnError(err)
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
	link, err := dispatcher.Dispatch(ctx, toDestination(metadata.Destination, net.Network_TCP))
	if err != nil {
		return err
	}
	outConn := &pipeConnWrapper{
		&buf.BufferedReader{Reader: link.Reader},
		link.Writer,
		conn,
	}
	return bufio.CopyConn(ctx, conn, outConn)
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
	destination := toDestination(metadata.Destination, net.Network_UDP)
	link, err := dispatcher.Dispatch(ctx, destination)
	if err != nil {
		return err
	}
	outConn := &packetConnWrapper{
		Reader: link.Reader,
		Writer: link.Writer,
		Dest:   destination,
	}
	return bufio.CopyPacketConn(ctx, conn, outConn)
}

func (i *Inbound) NewError(ctx context.Context, err error) {
	if E.IsClosed(err) {
		return
	}
	newError(err).AtWarning().WriteToLog()
}

type natPacketConn struct {
	net.Conn
}

func (c *natPacketConn) ReadPacket(buffer *B.Buffer) (addr M.Socksaddr, err error) {
	_, err = buffer.ReadFrom(c)
	return
}

func (c *natPacketConn) WritePacket(buffer *B.Buffer, addr M.Socksaddr) error {
	_, err := buffer.WriteTo(c)
	return err
}
