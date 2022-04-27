package singbridge

import (
	"context"

	"github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/common/network"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
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
