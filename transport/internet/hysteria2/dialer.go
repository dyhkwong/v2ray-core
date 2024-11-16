package hysteria2

import (
	"context"
	gotls "crypto/tls"
	"sync"
	"time"

	"github.com/apernet/quic-go/quicvarint"
	hyClient "github.com/dyhkwong/hysteria/core/v2/client"
	hyProtocol "github.com/dyhkwong/hysteria/core/v2/international/protocol"
	"github.com/dyhkwong/hysteria/extras/v2/obfs"
	"github.com/dyhkwong/hysteria/extras/v2/transport/udphop"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/environment"
	"github.com/v2fly/v2ray-core/v5/common/environment/envctx"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

type dialerConf struct {
	net.Destination
	*internet.MemoryStreamConfig
}

type transportConnectionState struct {
	scopedDialerMap    map[dialerConf]hyClient.Client
	scopedDialerAccess sync.Mutex
}

func (t *transportConnectionState) IsTransientStorageLifecycleReceiver() {
}

func (t *transportConnectionState) Close() error {
	t.scopedDialerAccess.Lock()
	for _, client := range t.scopedDialerMap {
		_ = client.Close()
	}
	clear(t.scopedDialerMap)
	t.scopedDialerAccess.Unlock()
	return nil
}

var MBps uint64 = 1000000 / 8 // MByte

func GetClientTLSConfig(dest net.Destination, streamSettings *internet.MemoryStreamConfig) (*gotls.Config, error) {
	config := tls.ConfigFromStreamSettings(streamSettings)
	if config == nil {
		return nil, newError(Hy2MustNeedTLS)
	}

	return config.GetTLSConfig(tls.WithDestination(dest), tls.WithNextProto("h3")), nil
}

func ResolveAddress(ctx context.Context, dest net.Destination, resolver func(ctx context.Context, domain string) net.Address) (net.Addr, error) {
	if dest.Address.Family().IsIP() {
		return &net.UDPAddr{
			IP:   dest.Address.IP(),
			Port: int(dest.Port),
		}, nil
	}
	if resolver != nil {
		if addr := resolver(ctx, dest.Address.Domain()); addr != nil {
			return &net.UDPAddr{
				IP:   addr.IP(),
				Port: int(dest.Port),
			}, nil
		}
		return nil, newError("failed to resolve domain ", dest.Address.Domain())
	}
	return net.ResolveUDPAddr("udp", dest.NetAddr())
}

type connFactory struct {
	hyClient.ConnFactory

	NewFunc    func(addr net.Addr) (net.PacketConn, error)
	Obfuscator obfs.Obfuscator
}

func (f *connFactory) New(addr net.Addr) (net.PacketConn, error) {
	conn, err := f.NewFunc(addr)
	if err != nil {
		return nil, err
	}
	if f.Obfuscator == nil {
		return conn, nil
	}
	return obfs.WrapPacketConn(conn, f.Obfuscator), nil
}

func NewHyClient(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig, resolver func(ctx context.Context, domain string) net.Address) (hyClient.Client, error) {
	tlsConfig, err := GetClientTLSConfig(dest, streamSettings)
	if err != nil {
		return nil, err
	}

	serverAddr, err := ResolveAddress(ctx, dest, resolver)
	if err != nil {
		return nil, err
	}

	config := streamSettings.ProtocolSettings.(*Config)
	hyConfig := &hyClient.Config{
		Auth:            config.GetPassword(),
		TLSConfig:       tlsConfig,
		ServerAddr:      serverAddr,
		BandwidthConfig: hyClient.BandwidthConfig{MaxTx: config.Congestion.GetUpMbps() * MBps, MaxRx: config.GetCongestion().GetDownMbps() * MBps},
		FastOpen:        true,
	}

	if len(config.HopPorts) > 0 {
		if config.HopPorts == "all" || config.HopPorts == "*" {
			return nil, newError("invalid hopPorts")
		}
		host, _, err := net.SplitHostPort(serverAddr.String())
		if err != nil {
			return nil, err
		}
		udpHopAddr, err := udphop.ResolveUDPHopAddr(net.JoinHostPort(host, config.HopPorts))
		if err != nil {
			return nil, err
		}
		hyConfig.ServerAddr = udpHopAddr
	}

	connFactory := &connFactory{
		NewFunc: func(addr net.Addr) (net.PacketConn, error) {
			if len(config.HopPorts) > 0 {
				return udphop.NewUDPHopPacketConn(addr.(*udphop.UDPHopAddr), time.Duration(config.HopInterval)*time.Second,
					func() (net.PacketConn, error) {
						rawConn, err := internet.DialSystem(ctx, net.DestinationFromAddr(serverAddr), streamSettings.SocketSettings)
						if err != nil {
							return nil, newError("failed to dial to dest: ", err).AtWarning().Base(err)
						}
						var pc net.PacketConn
						switch rc := rawConn.(type) {
						case *internet.PacketConnWrapper:
							pc = rc.Conn
						case net.PacketConn:
							pc = rc
						default:
							pc = internet.NewConnWrapper(rc)
						}
						return pc, nil
					},
				)
			}
			rawConn, err := internet.DialSystem(ctx, net.DestinationFromAddr(addr), streamSettings.SocketSettings)
			if err != nil {
				return nil, newError("failed to dial to dest: ", err).AtWarning().Base(err)
			}
			var pc net.PacketConn
			switch rc := rawConn.(type) {
			case *internet.PacketConnWrapper:
				pc = rc.Conn
			case net.PacketConn:
				pc = rc
			default:
				pc = internet.NewConnWrapper(rc)
			}
			return pc, nil
		},
	}
	if config.Obfs != nil && config.Obfs.Type == "salamander" {
		ob, err := obfs.NewSalamanderObfuscator([]byte(config.Obfs.Password))
		if err != nil {
			return nil, err
		}
		connFactory.Obfuscator = ob
	}
	hyConfig.ConnFactory = connFactory

	client, _, err := hyClient.NewClient(hyConfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func CloseHyClient(state *transportConnectionState, dest net.Destination, streamSettings *internet.MemoryStreamConfig) error {
	state.scopedDialerAccess.Lock()
	defer state.scopedDialerAccess.Unlock()

	client, found := state.scopedDialerMap[dialerConf{dest, streamSettings}]
	if found {
		delete(state.scopedDialerMap, dialerConf{dest, streamSettings})
		return client.Close()
	}
	return nil
}

func GetHyClient(ctx context.Context, state *transportConnectionState, dest net.Destination, streamSettings *internet.MemoryStreamConfig, resolver func(ctx context.Context, domain string) net.Address) (hyClient.Client, error) {
	var err error
	var client hyClient.Client

	state.scopedDialerAccess.Lock()
	client, found := state.scopedDialerMap[dialerConf{dest, streamSettings}]
	state.scopedDialerAccess.Unlock()
	if !found || !CheckHyClientHealthy(client) {
		if found {
			// retry
			CloseHyClient(state, dest, streamSettings)
		}
		client, err = NewHyClient(ctx, dest, streamSettings, resolver)
		if err != nil {
			return nil, err
		}
		state.scopedDialerAccess.Lock()
		state.scopedDialerMap[dialerConf{dest, streamSettings}] = client
		state.scopedDialerAccess.Unlock()
	}
	return client, nil
}

func CheckHyClientHealthy(client hyClient.Client) bool {
	quicConn := client.GetQuicConn()
	if quicConn == nil {
		return false
	}
	select {
	case <-quicConn.Context().Done():
		return false
	default:
	}
	return true
}

func Dial(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (internet.Connection, error) {
	dest.Network = net.Network_UDP
	transportEnvironment := envctx.EnvironmentFromContext(ctx).(environment.TransportEnvironment)
	state, err := transportEnvironment.TransientStorage().Get(ctx, "hysteria2-transport-connection-state")
	if err != nil {
		state = &transportConnectionState{}
		transportEnvironment.TransientStorage().Put(ctx, "hysteria2-transport-connection-state", state)
		state, err = transportEnvironment.TransientStorage().Get(ctx, "hysteria2-transport-connection-state")
		if err != nil {
			return nil, newError("failed to get hysteria2 transport connection state").Base(err)
		}
	}
	stateTyped := state.(*transportConnectionState)
	stateTyped.scopedDialerAccess.Lock()
	if stateTyped.scopedDialerMap == nil {
		stateTyped.scopedDialerMap = make(map[dialerConf]hyClient.Client)
	}
	stateTyped.scopedDialerAccess.Unlock()

	config := streamSettings.ProtocolSettings.(*Config)

	var resolver func(ctx context.Context, domain string) net.Address
	outbound := session.OutboundFromContext(ctx)
	if outbound != nil {
		resolver = outbound.Resolver
	}
	client, err := GetHyClient(ctx, stateTyped, dest, streamSettings, resolver)
	if err != nil {
		CloseHyClient(stateTyped, dest, streamSettings)
		return nil, err
	}

	quicConn := client.GetQuicConn()
	conn := &HyConn{
		local:  quicConn.LocalAddr(),
		remote: quicConn.RemoteAddr(),
	}

	network := net.Network_TCP
	if outbound != nil {
		network = outbound.Target.Network
	}

	if network == net.Network_UDP && config.GetUseUdpExtension() { // only hysteria2 can use udpExtension
		conn.IsUDPExtension = true
		conn.IsServer = false
		conn.ClientUDPSession, err = client.UDP()
		if err != nil {
			CloseHyClient(stateTyped, dest, streamSettings)
			return nil, err
		}
		return conn, nil
	}

	conn.stream, err = client.OpenStream()
	if err != nil {
		CloseHyClient(stateTyped, dest, streamSettings)
		return nil, err
	}

	// write TCP frame type
	frameSize := quicvarint.Len(hyProtocol.FrameTypeTCPRequest)
	buf := make([]byte, frameSize)
	hyProtocol.VarintPut(buf, hyProtocol.FrameTypeTCPRequest)
	_, err = conn.stream.Write(buf)
	if err != nil {
		CloseHyClient(stateTyped, dest, streamSettings)
		return nil, err
	}
	return conn, nil
}

func init() {
	common.Must(internet.RegisterTransportDialer(protocolName, Dial))
}
