package wireguard

import (
	"context"
	"errors"
	"io"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/log"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal"
	"github.com/v2fly/v2ray-core/v5/common/task"
	"github.com/v2fly/v2ray-core/v5/features/policy"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

type Server struct {
	ctx           context.Context
	inboundTag    *session.Inbound
	contentTag    *session.Content
	bind          *netBindServer
	dispatcher    routing.Dispatcher
	policyManager policy.Manager
}

func NewServer(ctx context.Context, conf *ServerConfig) (*Server, error) {
	v := core.MustFromContext(ctx)

	addresses, _, _, err := parseEndpoints(conf.Address)
	if err != nil {
		return nil, err
	}

	server := &Server{
		bind: &netBindServer{
			netBind: netBind{
				workers: int(conf.NumWorkers),
			},
		},
		policyManager: v.GetFeature(policy.ManagerType()).(policy.Manager),
	}

	tun, err := createTun(addresses, int(conf.Mtu), server.forwardConnection)
	if err != nil {
		return nil, err
	}

	if err = tun.BuildDevice(createIPCRequest(conf.SecretKey, conf.Peers, true), server.bind); err != nil {
		_ = tun.Close()
		return nil, err
	}

	return server, nil
}

// Network implements proxy.Inbound.
func (*Server) Network() []net.Network {
	return []net.Network{net.Network_UDP}
}

// Process implements proxy.Inbound.
func (s *Server) Process(ctx context.Context, network net.Network, conn internet.Connection, dispatcher routing.Dispatcher) error {
	s.ctx = ctx
	s.dispatcher = dispatcher
	s.inboundTag = session.InboundFromContext(ctx)
	s.contentTag = session.ContentFromContext(ctx)

	ep, err := s.bind.ParseEndpoint(conn.RemoteAddr().String())
	if err != nil {
		return err
	}

	nep := ep.(*netEndpoint)
	nep.conn = conn

	reader := buf.NewPacketReader(conn)
	for {
		mpayload, err := reader.ReadMultiBuffer()
		if err != nil {
			return err
		}

		for _, payload := range mpayload {
			v, ok := <-s.bind.readQueue
			if !ok {
				return nil
			}
			i, err := payload.Read(v.buff)

			v.bytes = i
			v.endpoint = nep
			v.err = err
			v.waiter.Done()
			if err != nil && errors.Is(err, io.EOF) {
				nep.conn = nil
				return nil
			}
		}
	}
}

func subContextFromMuxInbound(ctx context.Context) context.Context {
	newOutbound := session.Outbound{}
	content := session.ContentFromContext(ctx)
	newContent := session.Content{}
	if content != nil {
		newContent = *content
		if content.Attributes != nil {
			panic("content.Attributes != nil")
		}
	}
	return session.ContextWithContent(session.ContextWithOutbound(ctx, &newOutbound), &newContent)

}

func (s *Server) forwardConnection(dest net.Destination, conn net.Conn) {
	defer conn.Close()

	ctx, cancel := context.WithCancel(core.ToBackgroundDetachedContext(s.ctx))
	sid := session.NewID()
	ctx = session.ContextWithID(ctx, sid)
	inbound := session.Inbound{} // since promiscuousModeHandler mixed-up context, we shallow copy inbound (tag) and content (configs)
	if s.inboundTag != nil {
		inbound = *s.inboundTag
	}

	// overwrite the source to use the tun address for each sub context.
	// Since gvisor.ForwarderRequest doesn't provide any info to associate the sub-context with the Parent context
	// Currently we have no way to link to the original source address
	ctx = session.ContextWithInbound(ctx, &inbound)
	if s.contentTag != nil {
		ctx = session.ContextWithContent(ctx, s.contentTag)
	}
	ctx = subContextFromMuxInbound(ctx)

	inbound.Source = net.DestinationFromAddr(conn.RemoteAddr())
	plcy := s.policyManager.ForLevel(0)
	timer := signal.CancelAfterInactivity(ctx, cancel, plcy.Timeouts.ConnectionIdle)

	ctx = log.ContextWithAccessMessage(ctx, &log.AccessMessage{
		From:   net.TCPDestination(net.AnyIP, 0),
		To:     dest,
		Status: log.AccessAccepted,
		Reason: "",
	})

	link, err := s.dispatcher.Dispatch(ctx, dest)
	if err != nil {
		newError("dispatch connection").Base(err).AtError().WriteToLog()
	}
	defer cancel()

	requestDone := func() error {
		defer timer.SetTimeout(plcy.Timeouts.DownlinkOnly)
		if err := buf.Copy(buf.NewReader(conn), link.Writer, buf.UpdateActivity(timer)); err != nil {
			return newError("failed to transport request").Base(err)
		}

		return nil
	}

	responseDone := func() error {
		defer timer.SetTimeout(plcy.Timeouts.UplinkOnly)
		if err := buf.Copy(link.Reader, buf.NewWriter(conn), buf.UpdateActivity(timer)); err != nil {
			return newError("failed to transport response").Base(err)
		}

		return nil
	}

	requestDonePost := task.OnSuccess(requestDone, task.Close(link.Writer))
	if err := task.Run(ctx, requestDonePost, responseDone); err != nil {
		common.Interrupt(link.Reader)
		common.Interrupt(link.Writer)
		newError("connection ends").Base(err).AtDebug().WriteToLog()
		return
	}
}

func init() {
	common.Must(common.RegisterConfig((*ServerConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewServer(ctx, config.(*ServerConfig))
	}))
}
