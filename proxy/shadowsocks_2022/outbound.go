package shadowsocks_2022 // nolint:stylecheck

import (
	"context"
	"io"
	"time"

	shadowsocks "github.com/sagernet/sing-shadowsocks2"
	B "github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/bufio"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

func init() {
	common.Must(common.RegisterConfig((*ClientConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewClient(ctx, config.(*ClientConfig))
	}))
}

type Outbound struct {
	ctx    context.Context
	server net.Destination
	method shadowsocks.Method
}

func NewClient(ctx context.Context, config *ClientConfig) (*Outbound, error) {
	o := &Outbound{
		ctx: ctx,
		server: net.Destination{
			Address: config.Address.AsAddress(),
			Port:    net.Port(config.Port),
			Network: net.Network_TCP,
		},
	}
	method, err := shadowsocks.CreateMethod(ctx, config.Method, shadowsocks.MethodOptions{Password: config.Key})
	if err != nil {
		return nil, newError("create method").Base(err)
	}
	o.method = method
	return o, nil
}

func (o *Outbound) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified")
	}
	destination := outbound.Target
	network := destination.Network

	newError("tunneling request to ", destination, " via ", o.server.NetAddr()).WriteToLog(session.ExportIDToError(ctx))

	serverDestination := o.server
	serverDestination.Network = network

	connection, err := dialer.Dial(ctx, serverDestination)
	if err != nil {
		return newError("failed to connect to server").Base(err)
	}

	if network == net.Network_TCP {
		serverConn := o.method.DialEarlyConn(connection, toSocksaddr(destination))
		var handshake bool
		if timeoutReader, isTimeoutReader := link.Reader.(buf.TimeoutReader); isTimeoutReader {
			mb, err := timeoutReader.ReadMultiBufferTimeout(time.Millisecond * 100)
			if err != nil && err != buf.ErrNotTimeoutReader && err != buf.ErrReadTimeout {
				return newError("read payload").Base(err)
			}
			payload := B.New()
			for {
				payload.Reset()
				nb, n := buf.SplitBytes(mb, payload.FreeBytes())
				if n > 0 {
					payload.Truncate(n)
					_, err = serverConn.Write(payload.Bytes())
					if err != nil {
						payload.Release()
						return newError("write payload").Base(err)
					}
					handshake = true
				}
				if nb.IsEmpty() {
					break
				}
				mb = nb
			}
			payload.Release()
		}
		if !handshake {
			_, err = serverConn.Write(nil)
			if err != nil {
				return newError("client handshake").Base(err)
			}
		}
		conn := &pipeConnWrapper{
			W: link.Writer,
		}
		if ir, ok := link.Reader.(io.Reader); ok {
			conn.R = ir
		} else {
			conn.R = &buf.BufferedReader{Reader: link.Reader}
		}

		return returnError(bufio.CopyConn(ctx, conn, serverConn))
	} else {
		packetConn := &packetConnWrapper{
			Reader: link.Reader,
			Writer: link.Writer,
			Dest:   destination,
		}

		serverConn := o.method.DialPacketConn(connection)
		return returnError(bufio.CopyPacketConn(ctx, packetConn, serverConn))
	}
}
