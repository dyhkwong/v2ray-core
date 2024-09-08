package hysteria2

import (
	"context"
	"sync"

	hyClient "github.com/apernet/hysteria/core/v2/client"
	hyProtocol "github.com/apernet/hysteria/core/v2/international/protocol"
	"github.com/apernet/quic-go/quicvarint"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/uuid"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

type dialerConf struct {
	net.Destination
	*internet.MemoryStreamConfig
}

var RunningClient map[dialerConf](hyClient.Client)
var ClientMutex sync.Mutex
var MBps uint64 = 1000000 / 8 // MByte

func GetClientTLSConfig(streamSettings *internet.MemoryStreamConfig) (*hyClient.TLSConfig, error) {
	config := tls.ConfigFromStreamSettings(streamSettings)
	if config == nil {
		return nil, newError(Hy2MustNeedTLS)
	}
	tlsConfig := config.GetTLSConfig()

	return &hyClient.TLSConfig{
		RootCAs:               tlsConfig.RootCAs,
		ServerName:            tlsConfig.ServerName,
		InsecureSkipVerify:    tlsConfig.InsecureSkipVerify,
		VerifyPeerCertificate: tlsConfig.VerifyPeerCertificate,
	}, nil
}

func ResolveAddress(dest net.Destination) (net.Addr, error) {
	var destAddr *net.UDPAddr
	if dest.Address.Family().IsIP() {
		destAddr = &net.UDPAddr{
			IP:   dest.Address.IP(),
			Port: int(dest.Port),
		}
	} else {
		addr, err := net.ResolveUDPAddr("udp", dest.NetAddr())
		if err != nil {
			return nil, err
		}
		destAddr = addr
	}
	return destAddr, nil
}

type connFactory struct {
	hyClient.ConnFactory

	NewFunc    func(addr net.Addr) (net.PacketConn, error)
	Obfuscator Obfuscator
}

func (f *connFactory) New(addr net.Addr) (net.PacketConn, error) {
	if f.Obfuscator == nil {
		return f.NewFunc(addr)
	}
	conn, err := f.NewFunc(addr)
	if err != nil {
		return nil, err
	}
	return WrapPacketConn(conn, f.Obfuscator), nil
}

type connWrapper struct {
	net.Conn
	localAddr net.Addr
}

func (c *connWrapper) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	n, err = c.Read(p)
	return n, c.RemoteAddr(), err
}

func (c *connWrapper) WriteTo(p []byte, _ net.Addr) (n int, err error) {
	return c.Write(p)
}

func (c *connWrapper) LocalAddr() net.Addr {
	return c.localAddr
}

func NewConnWrapper(conn net.Conn) net.PacketConn {
	// https://github.com/quic-go/quic-go/commit/8189e75be6121fdc31dc1d6085f17015e9154667#diff-4c6aaadced390f3ce9bec0a9c9bb5203d5fa85df79023e3e0eec423dc9baa946R48-R62
	uuid := uuid.New()
	return &connWrapper{
		Conn:      conn,
		localAddr: &net.UnixAddr{Name: uuid.String()},
	}
}

func NewHyClient(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (hyClient.Client, error) {
	tlsConfig, err := GetClientTLSConfig(streamSettings)
	if err != nil {
		return nil, err
	}

	serverAddr, err := ResolveAddress(dest)
	if err != nil {
		return nil, err
	}

	config := streamSettings.ProtocolSettings.(*Config)
	hyConfig := &hyClient.Config{
		Auth:            config.GetPassword(),
		TLSConfig:       *tlsConfig,
		ServerAddr:      serverAddr,
		BandwidthConfig: hyClient.BandwidthConfig{MaxTx: config.Congestion.GetUpMbps() * MBps, MaxRx: config.GetCongestion().GetDownMbps() * MBps},
	}

	connFactory := &connFactory{
		NewFunc: func(addr net.Addr) (net.PacketConn, error) {
			rawConn, err := internet.DialSystem(ctx, net.DestinationFromAddr(addr), streamSettings.SocketSettings)
			if err != nil {
				return nil, newError("failed to dial to dest: ", err).AtWarning().Base(err)
			}
			switch conn := rawConn.(type) {
			case *internet.PacketConnWrapper:
				return conn.Conn, nil
			case net.PacketConn:
				return conn, nil
			default:
				return NewConnWrapper(conn), nil
			}
		},
	}
	if config.Obfs != nil && config.Obfs.Type == "salamander" {
		ob, err := NewSalamanderObfuscator([]byte(config.Obfs.Password))
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

func CloseHyClient(dest net.Destination, streamSettings *internet.MemoryStreamConfig) error {
	ClientMutex.Lock()
	defer ClientMutex.Unlock()

	client, found := RunningClient[dialerConf{dest, streamSettings}]
	if found {
		delete(RunningClient, dialerConf{dest, streamSettings})
		return client.Close()
	}
	return nil
}

func GetHyClient(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (hyClient.Client, error) {
	var err error
	var client hyClient.Client

	ClientMutex.Lock()
	client, found := RunningClient[dialerConf{dest, streamSettings}]
	ClientMutex.Unlock()
	if !found || !CheckHyClientHealthy(client) {
		if found {
			// retry
			CloseHyClient(dest, streamSettings)
		}
		client, err = NewHyClient(ctx, dest, streamSettings)
		if err != nil {
			return nil, err
		}
		ClientMutex.Lock()
		RunningClient[dialerConf{dest, streamSettings}] = client
		ClientMutex.Unlock()
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
	config := streamSettings.ProtocolSettings.(*Config)

	client, err := GetHyClient(ctx, dest, streamSettings)
	if err != nil {
		CloseHyClient(dest, streamSettings)
		return nil, err
	}

	quicConn := client.GetQuicConn()
	conn := &HyConn{
		local:  quicConn.LocalAddr(),
		remote: quicConn.RemoteAddr(),
	}

	outbound := session.OutboundFromContext(ctx)
	network := net.Network_TCP
	if outbound != nil {
		network = outbound.Target.Network
	}

	if network == net.Network_UDP && config.GetUseUdpExtension() { // only hysteria2 can use udpExtension
		conn.IsUDPExtension = true
		conn.IsServer = false
		conn.ClientUDPSession, err = client.UDP()
		if err != nil {
			CloseHyClient(dest, streamSettings)
			return nil, err
		}
		return conn, nil
	}

	conn.stream, err = client.OpenStream()
	if err != nil {
		CloseHyClient(dest, streamSettings)
		return nil, err
	}

	// write TCP frame type
	frameSize := int(quicvarint.Len(hyProtocol.FrameTypeTCPRequest))
	buf := make([]byte, frameSize)
	hyProtocol.VarintPut(buf, hyProtocol.FrameTypeTCPRequest)
	_, err = conn.stream.Write(buf)
	if err != nil {
		CloseHyClient(dest, streamSettings)
		return nil, err
	}
	return conn, nil
}

func init() {
	RunningClient = make(map[dialerConf]hyClient.Client)
	common.Must(internet.RegisterTransportDialer(protocolName, Dial))
}
