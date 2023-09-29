package reality

import (
	"context"

	utls "github.com/metacubex/utls"

	"github.com/v2fly/v2ray-core/v5/common/net"
)

type Conn struct {
	*utls.Conn
}

func (c *Conn) HandshakeAddress() net.Address {
	if err := c.Handshake(); err != nil {
		return nil
	}
	state := c.ConnectionState()
	if state.ServerName == "" {
		return nil
	}
	return net.ParseAddress(state.ServerName)
}

func Server(ctx context.Context, conn net.Conn, config *utls.RealityConfig) (net.Conn, error) {
	realityConn, err := utls.RealityServer(ctx, conn, config)
	if err != nil {
		return nil, err
	}
	return &Conn{Conn: realityConn}, nil
}
