package taggedimpl

import (
	"context"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/net/cnc"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tagged"
)

func DialTaggedOutbound(ctx context.Context, dest net.Destination, tag string) (net.Conn, error) {
	instance := core.FromContext(ctx)
	if instance == nil {
		return nil, newError("Instance context variable is not in context, dial denied. ")
	}
	dispatcher, ok := instance.GetFeature(routing.DispatcherType()).(routing.Dispatcher)
	if !ok || dispatcher == nil {
		return nil, newError("Required Feature dispatcher not resolved")
	}

	content := new(session.Content)

	ctx = session.ContextWithContent(ctx, content)
	ctx = session.SetForcedOutboundTagToContext(ctx, tag)

	r, err := dispatcher.Dispatch(ctx, dest)
	if err != nil {
		return nil, err
	}
	var readerOpt cnc.ConnectionOption
	if dest.Network == net.Network_TCP {
		readerOpt = cnc.ConnectionOutputMulti(r.Reader)
	} else {
		readerOpt = cnc.ConnectionOutputMultiUDP(r.Reader)
	}
	return cnc.NewConnection(cnc.ConnectionInputMulti(r.Writer), readerOpt), nil
}

func init() {
	tagged.Dialer = DialTaggedOutbound
}
