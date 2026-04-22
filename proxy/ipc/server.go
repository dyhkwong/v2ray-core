package ipc

import (
	"context"
	"io"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/errors"
	"github.com/v2fly/v2ray-core/v5/common/log"
	"github.com/v2fly/v2ray-core/v5/common/net"
	udp_proto "github.com/v2fly/v2ray-core/v5/common/protocol/udp"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal"
	"github.com/v2fly/v2ray-core/v5/common/task"
	"github.com/v2fly/v2ray-core/v5/features/policy"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/udp"
)

func init() {
	common.Must(common.RegisterConfig((*ServerConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewServer(ctx, config.(*ServerConfig))
	}))
}

// Server is an inbound connection handler that handles messages in trojan protocol.
type Server struct {
	config        *ServerConfig
	policyManager policy.Manager
}

// NewServer creates a new trojan inbound handler.
func NewServer(ctx context.Context, config *ServerConfig) (*Server, error) {
	v := core.MustFromContext(ctx)
	server := &Server{
		policyManager: v.GetFeature(policy.ManagerType()).(policy.Manager),
	}
	return server, nil
}

// Network implements proxy.Inbound.Network().
func (s *Server) Network() []net.Network {
	return []net.Network{net.Network_TCP, net.Network_UNIX}
}

// Process implements proxy.Inbound.Process().
func (s *Server) Process(ctx context.Context, network net.Network, conn internet.Connection, dispatcher routing.Dispatcher) error {
	sid := session.ExportIDToError(ctx)

	clientReader := &connReader{Reader: conn}
	if err := clientReader.parseHeader(); err != nil {
		log.Record(&log.AccessMessage{
			From:   conn.RemoteAddr(),
			To:     "",
			Status: log.AccessRejected,
			Reason: err,
		})
		return newError("failed to create request from: ", conn.RemoteAddr()).Base(err)
	}

	destination := clientReader.target

	inbound := session.InboundFromContext(ctx)
	if inbound == nil {
		panic("no inbound metadata")
	}

	level := uint32(0)
	if s.config != nil {
		level = uint32(s.config.Level)
	}
	sessionPolicy := s.policyManager.ForLevel(level)

	if destination.Network == net.Network_UDP {
		return s.handleUDPPayload(ctx, &packetReader{Reader: clientReader}, &packetWriter{Writer: conn}, dispatcher)
	}

	ctx = log.ContextWithAccessMessage(ctx, &log.AccessMessage{
		From:   conn.RemoteAddr(),
		To:     destination,
		Status: log.AccessAccepted,
		Reason: "",
	})

	newError("received request for ", destination).WriteToLog(sid)
	return s.handleConnection(ctx, sessionPolicy, destination, clientReader, buf.NewWriter(conn), dispatcher)
}

func (s *Server) handleUDPPayload(ctx context.Context, clientReader *packetReader, clientWriter *packetWriter, dispatcher routing.Dispatcher) error {
	udpServer := udp.NewSplitDispatcher(dispatcher, func(ctx context.Context, packet *udp_proto.Packet) {
		if err := clientWriter.writeMultiBufferWithMetadata(buf.MultiBuffer{packet.Payload}, packet.Source); err != nil {
			newError("failed to write response").Base(err).AtWarning().WriteToLog(session.ExportIDToError(ctx))
		}
	})

	inbound := session.InboundFromContext(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			p, err := clientReader.readMultiBufferWithMetadata()
			if err != nil {
				if errors.Cause(err) != io.EOF {
					return newError("unexpected EOF").Base(err)
				}
				return nil
			}
			currentPacketCtx := ctx
			currentPacketCtx = log.ContextWithAccessMessage(currentPacketCtx, &log.AccessMessage{
				From:   inbound.Source,
				To:     p.target,
				Status: log.AccessAccepted,
				Reason: "",
			})
			newError("tunnelling request to ", p.target).WriteToLog(session.ExportIDToError(ctx))

			for _, b := range p.buffer {
				udpServer.Dispatch(currentPacketCtx, p.target, b)
			}
		}
	}
}

func (s *Server) handleConnection(ctx context.Context, sessionPolicy policy.Session,
	destination net.Destination,
	clientReader buf.Reader,
	clientWriter buf.Writer, dispatcher routing.Dispatcher,
) error {
	ctx, cancel := context.WithCancel(ctx)
	timer := signal.CancelAfterInactivity(ctx, cancel, sessionPolicy.Timeouts.ConnectionIdle)
	ctx = policy.ContextWithBufferPolicy(ctx, sessionPolicy.Buffer)

	link, err := dispatcher.Dispatch(ctx, destination)
	if err != nil {
		return newError("failed to dispatch request to ", destination).Base(err)
	}

	requestDone := func() error {
		defer timer.SetTimeout(sessionPolicy.Timeouts.DownlinkOnly)
		if err := buf.Copy(clientReader, link.Writer, buf.UpdateActivity(timer)); err != nil {
			return newError("failed to transfer request").Base(err)
		}
		return nil
	}

	responseDone := func() error {
		defer timer.SetTimeout(sessionPolicy.Timeouts.UplinkOnly)
		if err := buf.Copy(link.Reader, clientWriter, buf.UpdateActivity(timer)); err != nil {
			return newError("failed to write response").Base(err)
		}
		return nil
	}

	requestDonePost := task.OnSuccess(requestDone, task.Close(link.Writer))
	if err := task.Run(ctx, requestDonePost, responseDone); err != nil {
		common.Must(common.Interrupt(link.Reader))
		common.Must(common.Interrupt(link.Writer))
		return newError("connection ends").Base(err)
	}

	return nil
}
