package shadowsocks

import (
	"context"
	"io"
	"strconv"
	"time"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/app/proxyman"
	app_inbound "github.com/v2fly/v2ray-core/v5/app/proxyman/inbound"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/log"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/net/packetaddr"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
	udp_proto "github.com/v2fly/v2ray-core/v5/common/protocol/udp"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal"
	"github.com/v2fly/v2ray-core/v5/common/task"
	"github.com/v2fly/v2ray-core/v5/common/uuid"
	"github.com/v2fly/v2ray-core/v5/features/inbound"
	"github.com/v2fly/v2ray-core/v5/features/policy"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/proxy"
	"github.com/v2fly/v2ray-core/v5/proxy/sip003"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/udp"
)

type MultiUserServer struct {
	config        *ServerConfig
	validator     *Validator
	policyManager policy.Manager

	tag            string
	pluginTag      string
	plugin         sip003.Plugin
	pluginOverride net.Destination
	receiverPort   int

	streamPlugin sip003.StreamPlugin
}

func (s *MultiUserServer) Initialize(self inbound.Handler) {
	s.tag = self.Tag()
}

func (s *MultiUserServer) Close() error {
	if s.plugin != nil {
		return s.plugin.Close()
	}
	return nil
}

// NewMultiUserServer create a new Shadowsocks multi-user server.
func NewMultiUserServer(ctx context.Context, config *ServerConfig) (proxy.Inbound, error) {
	if len(config.GetUsers()) == 0 {
		return nil, newError("users are not specified")
	}
	validator := new(Validator)
	for _, user := range config.Users {
		mUser, err := user.ToMemoryUser()
		if err != nil {
			return nil, newError("failed to parse user account").Base(err)
		}
		if err := validator.Add(mUser); err != nil {
			return nil, newError("failed to add user").Base(err)
		}
	}
	v := core.MustFromContext(ctx)
	s := &MultiUserServer{
		config:        config,
		validator:     validator,
		policyManager: v.GetFeature(policy.ManagerType()).(policy.Manager),
	}

	if config.Plugin != "" {
		var plugin sip003.Plugin
		if pc := sip003.Plugins[config.Plugin]; pc != nil {
			plugin = pc()
		} else if sip003.PluginLoader == nil {
			return nil, newError("plugin loader not registered")
		} else {
			plugin = sip003.PluginLoader(config.Plugin)
		}

		if streamPlugin, ok := plugin.(sip003.StreamPlugin); ok {
			s.streamPlugin = streamPlugin
			if err := streamPlugin.InitStreamPlugin("", config.PluginOpts); err != nil {
				return nil, newError("failed to start plugin").Base(err)
			}
			return s, nil
		}

		port, err := net.GetFreePort()
		if err != nil {
			return nil, newError("failed to get free port for sip003 plugin").Base(err)
		}
		s.receiverPort, err = net.GetFreePort()
		if err != nil {
			return nil, newError("failed to get free port for sip003 plugin receiver").Base(err)
		}
		u := uuid.New()
		tag := "v2ray.system.shadowsocks-inbound-plugin-receiver." + u.String()
		s.pluginTag = tag
		handler, err := app_inbound.NewAlwaysOnInboundHandlerWithProxy(ctx, tag, &proxyman.ReceiverConfig{
			Listen:    net.NewIPOrDomain(net.LocalHostIP),
			PortRange: net.SinglePortRange(net.Port(s.receiverPort)),
		}, s, true)
		if err != nil {
			return nil, newError("failed to create sip003 plugin inbound").Base(err)
		}
		inboundManager := v.GetFeature(inbound.ManagerType()).(inbound.Manager)
		if err := inboundManager.AddHandler(ctx, handler); err != nil {
			return nil, newError("failed to add sip003 plugin inbound").Base(err)
		}
		s.pluginOverride = net.Destination{
			Network: net.Network_TCP,
			Address: net.LocalHostIP,
			Port:    net.Port(port),
		}
		if err := plugin.Init(net.LocalHostIP.String(), strconv.Itoa(s.receiverPort), net.LocalHostIP.String(), strconv.Itoa(port), config.PluginOpts, config.PluginArgs); err != nil {
			return nil, newError("failed to start plugin").Base(err)
		}
		s.plugin = plugin
	}

	return s, nil
}

// AddUser implements proxy.UserManager.AddUser().
func (s *MultiUserServer) AddUser(ctx context.Context, u *protocol.MemoryUser) error {
	return s.validator.Add(u)
}

// RemoveUser implements proxy.UserManager.RemoveUser().
func (s *MultiUserServer) RemoveUser(ctx context.Context, e string) error {
	return s.validator.Del(e)
}

func (s *MultiUserServer) Network() []net.Network {
	list := s.config.Network
	if len(list) == 0 {
		list = append(list, net.Network_TCP)
	}
	if s.config.UdpEnabled {
		list = append(list, net.Network_UDP)
	}
	return list
}

func (s *MultiUserServer) Process(ctx context.Context, network net.Network, conn internet.Connection, dispatcher routing.Dispatcher) error {
	switch network {
	case net.Network_TCP:
		return s.handleConnection(ctx, conn, dispatcher)
	case net.Network_UDP:
		return s.handlerUDPPayload(ctx, conn, dispatcher)
	default:
		return newError("unknown network: ", network)
	}
}

func (s *MultiUserServer) handlerUDPPayload(ctx context.Context, conn internet.Connection, dispatcher routing.Dispatcher) error {
	udpDispatcherConstructor := udp.NewSplitDispatcher
	switch s.config.PacketEncoding {
	case packetaddr.PacketAddrType_None:
		break
	case packetaddr.PacketAddrType_Packet:
		packetAddrDispatcherFactory := udp.NewPacketAddrDispatcherCreator(ctx)
		udpDispatcherConstructor = packetAddrDispatcherFactory.NewPacketAddrDispatcher
	}

	udpServer := udpDispatcherConstructor(dispatcher, func(ctx context.Context, packet *udp_proto.Packet) {
		var request *protocol.RequestHeader
		request = protocol.RequestHeaderFromContext(ctx)
		if request == nil {
			request = &protocol.RequestHeader{}
		}
		if packet.Source.IsValid() {
			request.Port = packet.Source.Port
			request.Address = packet.Source.Address
		}

		payload := packet.Payload
		data, err := EncodeUDPPacket(request, payload.Bytes(), nil)
		payload.Release()

		if err != nil {
			newError("failed to encode UDP packet").Base(err).AtWarning().WriteToLog(session.ExportIDToError(ctx))
			return
		}
		defer data.Release()

		conn.Write(data.Bytes())
	})

	inbound := session.InboundFromContext(ctx)
	if inbound == nil {
		panic("no inbound metadata")
	}

	reader := buf.NewPacketReader(conn)
	for {
		mpayload, err := reader.ReadMultiBuffer()
		if err != nil {
			break
		}

		for _, payload := range mpayload {
			var request *protocol.RequestHeader
			var data *buf.Buffer
			var err error
			if inbound.User != nil {
				validator := new(Validator)
				validator.Add(inbound.User)
				request, data, err = DecodeUDPPacketMultiUser(validator, payload)
			} else {
				request, data, err = DecodeUDPPacketMultiUser(s.validator, payload)
				if err == nil {
					inbound.User = request.User
				}
			}
			if err != nil {
				if inbound := session.InboundFromContext(ctx); inbound != nil && inbound.Source.IsValid() {
					newError("dropping invalid UDP packet from: ", inbound.Source).Base(err).WriteToLog(session.ExportIDToError(ctx))
					log.Record(&log.AccessMessage{
						From:   inbound.Source,
						To:     "",
						Status: log.AccessRejected,
						Reason: err,
					})
				}
				payload.Release()
				continue
			}

			currentPacketCtx := ctx
			dest := request.Destination()
			if inbound.Source.IsValid() {
				currentPacketCtx = log.ContextWithAccessMessage(ctx, &log.AccessMessage{
					From:   inbound.Source,
					To:     dest,
					Status: log.AccessAccepted,
					Reason: "",
					Email:  request.User.Email,
				})
			}
			newError("tunnelling request to ", dest).WriteToLog(session.ExportIDToError(currentPacketCtx))

			currentPacketCtx = protocol.ContextWithRequestHeader(currentPacketCtx, request)
			udpServer.Dispatch(currentPacketCtx, dest, data)
		}
	}

	return nil
}

func (s *MultiUserServer) handleConnection(ctx context.Context, conn internet.Connection, dispatcher routing.Dispatcher) error {
	inbound := session.InboundFromContext(ctx)
	if inbound == nil {
		panic("no inbound metadata")
	}

	if s.plugin != nil {
		if inbound.Tag != s.pluginTag {
			dest, err := internet.Dial(ctx, s.pluginOverride, nil)
			if err != nil {
				return newError("failed to handle request to shadowsocks SIP003 plugin").Base(err)
			}
			if err := task.Run(ctx, func() error {
				_, err := io.Copy(conn, dest)
				return err
			}, func() error {
				_, err := io.Copy(dest, conn)
				return err
			}); err != nil {
				return newError("connection ends").Base(err)
			}
			return nil
		}
		inbound.Tag = s.tag
	} else if s.streamPlugin != nil {
		conn = s.streamPlugin.StreamConn(conn)
	}

	sessionPolicy := s.policyManager.ForLevel(0)

	conn.SetReadDeadline(time.Now().Add(sessionPolicy.Timeouts.Handshake))

	bufferedReader := buf.BufferedReader{Reader: buf.NewReader(conn)}

	request, bodyReader, err := ReadTCPSessionMultiUser(s.validator, &bufferedReader)
	sessionPolicy = s.policyManager.ForLevel(request.User.Level)
	inbound.User = request.User

	if err != nil {
		log.Record(&log.AccessMessage{
			From:   conn.RemoteAddr(),
			To:     "",
			Status: log.AccessRejected,
			Reason: err,
		})
		return newError("failed to create request from: ", conn.RemoteAddr()).Base(err)
	}
	conn.SetReadDeadline(time.Time{})

	dest := request.Destination()
	ctx = log.ContextWithAccessMessage(ctx, &log.AccessMessage{
		From:   conn.RemoteAddr(),
		To:     dest,
		Status: log.AccessAccepted,
		Reason: "",
		Email:  request.User.Email,
	})
	newError("tunnelling request to ", dest).WriteToLog(session.ExportIDToError(ctx))

	ctx, cancel := context.WithCancel(ctx)
	timer := signal.CancelAfterInactivity(ctx, cancel, sessionPolicy.Timeouts.ConnectionIdle)

	ctx = policy.ContextWithBufferPolicy(ctx, sessionPolicy.Buffer)
	link, err := dispatcher.Dispatch(ctx, dest)
	if err != nil {
		return err
	}

	responseDone := func() error {
		defer timer.SetTimeout(sessionPolicy.Timeouts.UplinkOnly)

		bufferedWriter := buf.NewBufferedWriter(buf.NewWriter(conn))
		responseWriter, err := WriteTCPResponseMultiUser(request, bufferedWriter)
		if err != nil {
			return newError("failed to write response").Base(err)
		}

		{
			payload, err := link.Reader.ReadMultiBuffer()
			if err != nil {
				return err
			}
			if err := responseWriter.WriteMultiBuffer(payload); err != nil {
				return err
			}
		}

		if err := bufferedWriter.SetBuffered(false); err != nil {
			return err
		}

		if err := buf.Copy(link.Reader, responseWriter, buf.UpdateActivity(timer)); err != nil {
			return newError("failed to transport all TCP response").Base(err)
		}

		return nil
	}

	requestDone := func() error {
		defer timer.SetTimeout(sessionPolicy.Timeouts.DownlinkOnly)

		if err := buf.Copy(bodyReader, link.Writer, buf.UpdateActivity(timer)); err != nil {
			return newError("failed to transport all TCP request").Base(err)
		}

		return nil
	}

	requestDoneAndCloseWriter := task.OnSuccess(requestDone, task.Close(link.Writer))
	if err := task.Run(ctx, requestDoneAndCloseWriter, responseDone); err != nil {
		common.Interrupt(link.Reader)
		common.Interrupt(link.Writer)
		return newError("connection ends").Base(err)
	}

	return nil
}
