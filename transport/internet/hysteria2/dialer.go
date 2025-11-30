//go:build !confonly

package hysteria2

import (
	"context"
	gotls "crypto/tls"
	"sync"

	"github.com/apernet/quic-go"
	"github.com/apernet/quic-go/quicvarint"
	hyClient "github.com/dyhkwong/hysteria/core/v2/client"
	hyProtocol "github.com/dyhkwong/hysteria/core/v2/international/protocol"
	"github.com/dyhkwong/hysteria/core/v2/international/utils"
	"github.com/dyhkwong/hysteria/extras/v2/obfs"

	"github.com/v2fly/v2ray-core/v4/common"
	"github.com/v2fly/v2ray-core/v4/common/environment"
	"github.com/v2fly/v2ray-core/v4/common/environment/envctx"
	"github.com/v2fly/v2ray-core/v4/common/net"
	"github.com/v2fly/v2ray-core/v4/common/session"
	"github.com/v2fly/v2ray-core/v4/features/dns/localdns"
	"github.com/v2fly/v2ray-core/v4/transport/internet"
	"github.com/v2fly/v2ray-core/v4/transport/internet/tls"
)

var _ hyClient.Client = (*lateInitHysteriaClient)(nil)

type lateInitHysteriaClient struct {
	initMutex      sync.Mutex
	clientMutex    sync.Mutex
	client         hyClient.Client
	closed         bool
	ctx            context.Context
	dest           net.Destination
	streamSettings *internet.MemoryStreamConfig
	resolver       func(ctx context.Context, domain string) net.Address
}

func (c *lateInitHysteriaClient) init() error {
	c.initMutex.Lock()
	defer c.initMutex.Unlock()
	c.clientMutex.Lock()
	if c.closed {
		c.clientMutex.Unlock()
		return newError("client closed")
	}
	if c.client != nil {
		c.clientMutex.Unlock()
		return nil
	}
	c.clientMutex.Unlock()
	client, err := NewHyClient(c.ctx, c.dest, c.streamSettings, c.resolver)
	if err != nil {
		return err
	}
	c.clientMutex.Lock()
	defer c.clientMutex.Unlock()
	if c.closed {
		client.Close()
		return newError("client closed")
	}
	c.client = client
	return nil
}

func (c *lateInitHysteriaClient) TCP(addr string) (net.Conn, error) {
	if err := c.init(); err != nil {
		return nil, err
	}
	return c.client.TCP(addr)
}

func (c *lateInitHysteriaClient) UDP() (hyClient.HyUDPConn, error) {
	if err := c.init(); err != nil {
		return nil, err
	}
	return c.client.UDP()
}

func (c *lateInitHysteriaClient) Close() error {
	c.clientMutex.Lock()
	defer c.clientMutex.Unlock()
	c.closed = true
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func (c *lateInitHysteriaClient) OpenStream() (*utils.QStream, error) {
	if err := c.init(); err != nil {
		return nil, err
	}
	return c.client.OpenStream()
}

func (c *lateInitHysteriaClient) GetQuicConn() *quic.Conn {
	if err := c.init(); err != nil {
		return nil
	}
	return c.client.GetQuicConn()
}

func (c *lateInitHysteriaClient) getQuicConn() (*quic.Conn, error) {
	if err := c.init(); err != nil {
		return nil, err
	}
	quicConn := c.client.GetQuicConn()
	if quicConn == nil {
		return nil, newError("get quic conn failed")
	}
	return quicConn, nil
}

type dialerConf struct {
	net.Destination
	*internet.MemoryStreamConfig
}

type transportConnectionState struct {
	scopedDialerMap    map[dialerConf]*lateInitHysteriaClient
	scopedDialerAccess sync.Mutex
}

type dialerCanceller func()

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
	switch {
	case dest.Address.Family().IsIP():
		return &net.UDPAddr{
			IP:   dest.Address.IP(),
			Port: int(dest.Port),
		}, nil
	case resolver != nil:
		if addr := resolver(ctx, dest.Address.Domain()); addr != nil {
			return &net.UDPAddr{
				IP:   addr.IP(),
				Port: int(dest.Port),
			}, nil
		}
		return nil, newError("failed to resolve domain ", dest.Address.Domain())
	default:
		addr, err := localdns.New().LookupIP(dest.Address.Domain())
		if err != nil {
			return nil, err
		}
		return &net.UDPAddr{
			IP:   addr[0],
			Port: int(dest.Port),
		}, nil
	}
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

	connFactory := &connFactory{
		NewFunc: func(addr net.Addr) (net.PacketConn, error) {
			rawConn, err := internet.DialSystem(ctx, net.DestinationFromAddr(addr), streamSettings.SocketSettings)
			if err != nil {
				return nil, newError("failed to dial to dest: ", err).AtWarning().Base(err)
			}
			var packetConn net.PacketConn
			switch rawConn := rawConn.(type) {
			case *internet.PacketConnWrapper:
				packetConn = rawConn.Conn
			case net.PacketConn:
				packetConn = rawConn
			default:
				packetConn = internet.NewConnWrapper(rawConn)
			}
			return packetConn, nil
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

func GetHyClient(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig, resolver func(ctx context.Context, domain string) net.Address) (*lateInitHysteriaClient, dialerCanceller, error) {
	dest.Network = net.Network_UDP
	transportEnvironment := envctx.EnvironmentFromContext(ctx).(environment.TransportEnvironment)
	state, err := transportEnvironment.TransientStorage().Get(ctx, "hysteria2-transport-connection-state")
	if err != nil {
		state = &transportConnectionState{}
		transportEnvironment.TransientStorage().Put(ctx, "hysteria2-transport-connection-state", state)
		state, err = transportEnvironment.TransientStorage().Get(ctx, "hysteria2-transport-connection-state")
		if err != nil {
			return nil, nil, newError("failed to get hysteria2 transport connection state").Base(err)
		}
	}
	stateTyped := state.(*transportConnectionState)
	stateTyped.scopedDialerAccess.Lock()
	defer stateTyped.scopedDialerAccess.Unlock()
	if stateTyped.scopedDialerMap == nil {
		stateTyped.scopedDialerMap = make(map[dialerConf]*lateInitHysteriaClient)
	}
	canceller := func() {
		CloseHyClient(stateTyped, dest, streamSettings)
	}
	client, found := stateTyped.scopedDialerMap[dialerConf{dest, streamSettings}]
	if found {
		return client, canceller, nil
	}
	client = &lateInitHysteriaClient{
		ctx:            ctx,
		dest:           dest,
		streamSettings: streamSettings,
		resolver:       resolver,
	}
	stateTyped.scopedDialerMap[dialerConf{dest, streamSettings}] = client
	return client, canceller, nil
}

func CheckHyClientHealthy(quicConn *quic.Conn) bool {
	select {
	case <-quicConn.Context().Done():
		return false
	default:
		return true
	}
}

func Dial(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (internet.Connection, error) {
	config := streamSettings.ProtocolSettings.(*Config)
	var resolver func(ctx context.Context, domain string) net.Address
	outbound := session.OutboundFromContext(ctx)
	if outbound != nil {
		resolver = outbound.Resolver
	}
	client, canceller, err := GetHyClient(ctx, dest, streamSettings, resolver)
	if err != nil {
		return nil, err
	}

	quicConn, err := client.getQuicConn()
	if err != nil {
		canceller()
		return nil, err
	}

	if !CheckHyClientHealthy(quicConn) {
		// retry
		canceller()
		client, canceller, err = GetHyClient(ctx, dest, streamSettings, resolver)
		if err != nil {
			return nil, err
		}
		quicConn, err = client.getQuicConn()
		if err != nil {
			canceller()
			return nil, err
		}
	}

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
			canceller()
			return nil, err
		}
		return conn, nil
	}

	conn.stream, err = client.OpenStream()
	if err != nil {
		return nil, err
	}

	// write TCP frame type
	frameSize := quicvarint.Len(hyProtocol.FrameTypeTCPRequest)
	buf := make([]byte, frameSize)
	hyProtocol.VarintPut(buf, hyProtocol.FrameTypeTCPRequest)
	_, err = conn.stream.Write(buf)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func init() {
	common.Must(internet.RegisterTransportDialer(protocolName, Dial))
}
