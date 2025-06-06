package grpc

import (
	"context"
	gotls "crypto/tls"
	gonet "net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/environment"
	"github.com/v2fly/v2ray-core/v5/common/environment/envctx"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/grpc/encoding"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls/utls"
)

func Dial(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (internet.Connection, error) {
	newError("creating connection to ", dest).WriteToLog(session.ExportIDToError(ctx))

	conn, err := dialgRPC(ctx, dest, streamSettings)
	if err != nil {
		return nil, newError("failed to dial Grpc").Base(err)
	}
	return internet.Connection(conn), nil
}

func init() {
	common.Must(internet.RegisterTransportDialer(protocolName, Dial))
}

type dialerConf struct {
	net.Destination
	*internet.MemoryStreamConfig
}

type transportConnectionState struct {
	scopedDialerMap    map[dialerConf]*grpc.ClientConn
	scopedDialerAccess sync.Mutex
}

func (t *transportConnectionState) IsTransientStorageLifecycleReceiver() {
}

func (t *transportConnectionState) Close() error {
	t.scopedDialerAccess.Lock()
	defer t.scopedDialerAccess.Unlock()
	for _, conn := range t.scopedDialerMap {
		_ = conn.Close()
	}
	t.scopedDialerMap = nil
	return nil
}

type dialerCanceller func()

func dialgRPC(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (net.Conn, error) {
	grpcSettings := streamSettings.ProtocolSettings.(*Config)

	transportCredentials := insecure.NewCredentials()
	switch streamSettings.SecuritySettings.(type) {
	case *tls.Config:
		if config := tls.ConfigFromStreamSettings(streamSettings); config != nil {
			tlsConfig := config.GetTLSConfig(tls.WithDestination(dest))
			// https://github.com/grpc/grpc-go/blob/98959d9a4904e98bbf8b423ce6a3cb5d36f90ee1/credentials/tls.go#L205-L210
			if tlsConfig.EncryptedClientHelloConfigList != nil && tlsConfig.MinVersion == 0 && (tlsConfig.MaxVersion == 0 || tlsConfig.MaxVersion >= gotls.VersionTLS13) {
				config.MinVersion = gotls.VersionTLS13
			}
			transportCredentials = credentials.NewTLS(tlsConfig)
		}
	case *utls.Config:
		if creds, err := newSecurityEngineCreds(ctx, dest, streamSettings); err == nil {
			transportCredentials = creds
		} else {
			newError("failed to create utls grpc credentials").Base(err).WriteToLog(session.ExportIDToError(ctx))
		}
	}
	dialOption := grpc.WithTransportCredentials(transportCredentials)

	conn, canceller, err := getGrpcClient(ctx, dest, dialOption, streamSettings)
	if err != nil {
		return nil, newError("Cannot dial grpc").Base(err)
	}
	client := encoding.NewGunServiceClient(conn)
	gunService, err := client.(encoding.GunServiceClientX).TunCustomName(ctx, grpcSettings.ServiceName)
	if err != nil {
		canceller()
		return nil, newError("Cannot dial grpc").Base(err)
	}
	return encoding.NewGunConn(gunService, nil), nil
}

func getGrpcClient(ctx context.Context, dest net.Destination, dialOption grpc.DialOption, streamSettings *internet.MemoryStreamConfig) (*grpc.ClientConn, dialerCanceller, error) {
	transportEnvironment := envctx.EnvironmentFromContext(ctx).(environment.TransportEnvironment)
	state, err := transportEnvironment.TransientStorage().Get(ctx, "grpc-transport-connection-state")
	if err != nil {
		state = &transportConnectionState{}
		transportEnvironment.TransientStorage().Put(ctx, "grpc-transport-connection-state", state)
		state, err = transportEnvironment.TransientStorage().Get(ctx, "grpc-transport-connection-state")
		if err != nil {
			return nil, nil, newError("failed to get grpc transport connection state").Base(err)
		}
	}
	stateTyped := state.(*transportConnectionState)

	stateTyped.scopedDialerAccess.Lock()
	defer stateTyped.scopedDialerAccess.Unlock()

	if stateTyped.scopedDialerMap == nil {
		stateTyped.scopedDialerMap = make(map[dialerConf]*grpc.ClientConn)
	}

	canceller := func() {
		stateTyped.scopedDialerAccess.Lock()
		defer stateTyped.scopedDialerAccess.Unlock()
		delete(stateTyped.scopedDialerMap, dialerConf{dest, streamSettings})
	}

	if client, found := stateTyped.scopedDialerMap[dialerConf{dest, streamSettings}]; found && client.GetState() != connectivity.Shutdown {
		return client, canceller, nil
	}

	conn, err := grpc.NewClient(
		dest.Address.String()+":"+dest.Port.String(),
		dialOption,
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  500 * time.Millisecond,
				Multiplier: 1.5,
				Jitter:     0.2,
				MaxDelay:   19 * time.Second,
			},
			MinConnectTimeout: 5 * time.Second,
		}),
		grpc.WithContextDialer(func(ctxGrpc context.Context, s string) (gonet.Conn, error) {
			rawHost, rawPort, err := net.SplitHostPort(s)
			if err != nil {
				return nil, err
			}
			if len(rawPort) == 0 {
				rawPort = "443"
			}
			port, err := net.PortFromString(rawPort)
			if err != nil {
				return nil, err
			}
			address := net.ParseAddress(rawHost)
			detachedContext := core.ToBackgroundDetachedContext(ctx)
			return internet.DialSystem(detachedContext, net.TCPDestination(address, port), streamSettings.SocketSettings)
		}),
	)
	canceller = func() {
		stateTyped.scopedDialerAccess.Lock()
		defer stateTyped.scopedDialerAccess.Unlock()
		delete(stateTyped.scopedDialerMap, dialerConf{dest, streamSettings})
		if err != nil {
			conn.Close()
		}
	}
	stateTyped.scopedDialerMap[dialerConf{dest, streamSettings}] = conn
	return conn, canceller, err
}
