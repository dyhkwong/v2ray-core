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
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

const (
	FrameTypeTCPRequest = 0x401
)

type dialerConf struct {
	net.Destination
	*internet.MemoryStreamConfig
}

var (
	RunningClient map[dialerConf](hyClient.Client)
	ClientMutex   sync.Mutex
)

type connFactory struct {
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

func NewHyClient(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (hyClient.Client, error) {
	tlsSettings := tls.ConfigFromStreamSettings(streamSettings)
	if tlsSettings == nil {
		tlsSettings = &tls.Config{
			ServerName:    internalDomain,
			AllowInsecure: true,
		}
	}
	tlsConfig := tlsSettings.GetTLSConfig(tls.WithDestination(dest))
	hyTLSConfig := &hyClient.TLSConfig{
		ServerName:            tlsConfig.ServerName,
		InsecureSkipVerify:    tlsConfig.InsecureSkipVerify,
		VerifyPeerCertificate: tlsConfig.VerifyPeerCertificate,
		RootCAs:               tlsConfig.RootCAs,
	}

	var serverAddr *net.UDPAddr
	if dest.Address.Family().IsIP() {
		serverAddr = &net.UDPAddr{
			IP:   dest.Address.IP(),
			Port: int(dest.Port),
		}
	} else {
		addr, err := net.ResolveUDPAddr("udp", dest.NetAddr())
		if err != nil {
			return nil, err
		}
		serverAddr = addr
	}

	config := streamSettings.ProtocolSettings.(*Config)
	hyConfig := &hyClient.Config{
		TLSConfig:  *hyTLSConfig,
		Auth:       config.GetPassword(),
		ServerAddr: serverAddr,
	}

	connFactory := &connFactory{
		NewFunc: func(addr net.Addr) (net.PacketConn, error) {
			rawConn, err := internet.DialSystem(ctx, net.DestinationFromAddr(addr), streamSettings.SocketSettings)
			if err != nil {
				return nil, newError("failed to dial to dest: ", err).AtWarning().Base(err)
			}
			var udpConn *net.UDPConn
			switch conn := rawConn.(type) {
			case *net.UDPConn:
				udpConn = conn
			case *internet.PacketConnWrapper:
				udpConn = conn.Conn.(*net.UDPConn)
			default:
				rawConn.Close()
				return nil, newError("QUIC with sockopt is unsupported").AtWarning()
			}
			return udpConn, nil
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
	ClientMutex.Lock()
	client, found := RunningClient[dialerConf{dest, streamSettings}]
	ClientMutex.Unlock()
	var err error
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
	// TODO: Clean idle connections
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
		conn.Target = outbound.Target
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
	frameSize := quicvarint.Len(FrameTypeTCPRequest)
	buf := make([]byte, frameSize)
	hyProtocol.VarintPut(buf, FrameTypeTCPRequest)
	conn.stream.Write(buf)
	return conn, nil
}

func init() {
	RunningClient = make(map[dialerConf]hyClient.Client)
	common.Must(internet.RegisterTransportDialer(protocolName, Dial))
}
