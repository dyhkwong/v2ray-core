package singbridge

import (
	"context"

	"github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/common/network"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/net/cnc"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/proxy"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/pipe"
)

var _ network.Dialer = (*dialerWrapper)(nil)

type dialerWrapper struct {
	dialer internet.Dialer
}

func NewDialerWrapper(dialer internet.Dialer) *dialerWrapper {
	return &dialerWrapper{
		dialer: dialer,
	}
}

func (d *dialerWrapper) DialContext(ctx context.Context, network string, destination metadata.Socksaddr) (net.Conn, error) {
	return d.dialer.Dial(ctx, ToDestination(destination, ToNetwork(network)))
}

func (d *dialerWrapper) ListenPacket(ctx context.Context, destination metadata.Socksaddr) (net.PacketConn, error) {
	panic("invalid")
}

func NewOutboundDialerWrapper(outbound proxy.Outbound, dialer internet.Dialer) *outboundDialerWrapper {
	return &outboundDialerWrapper{outbound, dialer}
}

type outboundDialerWrapper struct {
	outbound proxy.Outbound
	dialer   internet.Dialer
}

func (d *outboundDialerWrapper) DialContext(ctx context.Context, network string, destination metadata.Socksaddr) (net.Conn, error) {
	ctx = session.ContextWithOutbound(ctx, &session.Outbound{
		Target: ToDestination(destination, ToNetwork(network)),
	})
	opts := []pipe.Option{pipe.WithSizeLimit(64 * 1024)}
	uplinkReader, uplinkWriter := pipe.New(opts...)
	downlinkReader, downlinkWriter := pipe.New(opts...)
	conn := cnc.NewConnection(cnc.ConnectionInputMulti(downlinkWriter), cnc.ConnectionOutputMulti(uplinkReader))
	go d.outbound.Process(core.ToBackgroundDetachedContext(ctx), &transport.Link{Reader: downlinkReader, Writer: uplinkWriter}, d.dialer)
	return conn, nil
}

func (d *outboundDialerWrapper) ListenPacket(ctx context.Context, destination metadata.Socksaddr) (net.PacketConn, error) {
	panic("invalid")
}
