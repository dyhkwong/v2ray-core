package transportcommon

import (
	"context"

	"github.com/v2fly/v2ray-core/v5/common/environment"
	"github.com/v2fly/v2ray-core/v5/common/environment/envctx"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/security"
)

func DialWithSecuritySettings(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig, options ...security.Option) (internet.Connection, error) {
	transportEnvironment := envctx.EnvironmentFromContext(ctx).(environment.TransportEnvironment)
	dialer := transportEnvironment.Dialer()
	conn, err := dialer.Dial(ctx, nil, dest, streamSettings.SocketSettings)
	if err != nil {
		return nil, newError("failed to dial to ", dest).Base(err)
	}
	securityEngine, err := security.CreateSecurityEngineFromSettings(ctx, streamSettings)
	if err != nil {
		return nil, newError("unable to create security engine").Base(err)
	}

	if securityEngine != nil {
		if len(options) == 0 {
			options = []security.Option{security.OptionWithDestination{Dest: dest}}
		}
		conn, err = securityEngine.Client(conn, options...)
		if err != nil {
			return nil, newError("unable to create security protocol client from security engine").Base(err)
		}
	}
	return internet.Connection(conn), nil
}
