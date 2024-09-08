package hysteria2

import (
	"context"
	gotls "crypto/tls"
	"strings"

	"github.com/apernet/quic-go"
	"github.com/apernet/quic-go/http3"
	hyServer "github.com/dyhkwong/hysteria/core/v2/server"
	"github.com/dyhkwong/hysteria/extras/v2/obfs"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

// Listener is an internet.Listener that listens for TCP connections.
type Listener struct {
	hyServer hyServer.Server
	rawConn  net.PacketConn
	addConn  internet.ConnHandler
}

// Addr implements internet.Listener.Addr.
func (l *Listener) Addr() net.Addr {
	return l.rawConn.LocalAddr()
}

// Close implements internet.Listener.Close.
func (l *Listener) Close() error {
	return l.hyServer.Close()
}

func (l *Listener) StreamHijacker(ft http3.FrameType, conn quic.Connection, stream quic.Stream, err error) (bool, error) {
	// err always == nil

	tcpConn := &HyConn{
		stream: stream,
		local:  conn.LocalAddr(),
		remote: conn.RemoteAddr(),
	}
	l.addConn(tcpConn)
	return true, nil
}

func (l *Listener) UDPHijacker(entry *hyServer.UdpSessionEntry, originalAddr string) {
	addr, err := net.ResolveUDPAddr("udp", originalAddr)
	if err != nil {
		return
	}
	udpConn := &HyConn{
		IsUDPExtension:   true,
		IsServer:         true,
		ServerUDPSession: entry,
		remote:           addr,
		local:            l.rawConn.LocalAddr(),
	}
	l.addConn(udpConn)
}

// Listen creates a new Listener based on configurations.
func Listen(ctx context.Context, address net.Address, port net.Port, streamSettings *internet.MemoryStreamConfig, handler internet.ConnHandler) (internet.Listener, error) {
	tlsConfig, err := GetServerTLSConfig(streamSettings)
	if err != nil {
		return nil, err
	}

	if address.Family().IsDomain() {
		return nil, nil
	}

	config := streamSettings.ProtocolSettings.(*Config)
	rawConn, err := internet.ListenSystemPacket(context.Background(),
		&net.UDPAddr{
			IP:   address.IP(),
			Port: int(port),
		}, streamSettings.SocketSettings)
	if err != nil {
		return nil, err
	}

	listener := &Listener{
		rawConn: rawConn,
		addConn: handler,
	}

	hyConfig := &hyServer.Config{
		Conn:                  rawConn,
		TLSConfig:             tlsConfig,
		DisableUDP:            !config.GetUseUdpExtension(),
		StreamHijacker:        listener.StreamHijacker, // acceptStreams
		BandwidthConfig:       hyServer.BandwidthConfig{MaxTx: config.Congestion.GetUpMbps() * MBps, MaxRx: config.GetCongestion().GetDownMbps() * MBps},
		UdpSessionHijacker:    listener.UDPHijacker, // acceptUDPSession
		IgnoreClientBandwidth: config.GetIgnoreClientBandwidth(),
	}
	if len(config.GetPasswords()) > 0 {
		authenticator := &MultiUserAuthenticator{
			Passwords: make(map[string]any),
		}
		for _, password := range config.GetPasswords() {
			if index := strings.Index(password, ":"); index >= 0 {
				password = strings.ToLower(password[:index]) + ":" + password[index:]
			}
			authenticator.Passwords[password] = nil
		}
		hyConfig.Authenticator = authenticator
	} else {
		hyConfig.Authenticator = &Authenticator{Password: config.GetPassword()}
	}
	if config.Obfs != nil && config.Obfs.Type == "salamander" {
		ob, err := obfs.NewSalamanderObfuscator([]byte(config.Obfs.Password))
		if err != nil {
			return nil, err
		}
		hyConfig.Conn = obfs.WrapPacketConn(rawConn, ob)
	}
	hyServer, err := hyServer.NewServer(hyConfig)
	if err != nil {
		return nil, err
	}

	listener.hyServer = hyServer
	go hyServer.Serve()
	return listener, nil
}

func GetServerTLSConfig(streamSettings *internet.MemoryStreamConfig) (*gotls.Config, error) {
	config := tls.ConfigFromStreamSettings(streamSettings)
	if config == nil {
		return nil, newError(Hy2MustNeedTLS)
	}

	return config.GetTLSConfig(tls.WithNextProto("h3")), nil
}

type Authenticator struct {
	Password string
}

func (a *Authenticator) Authenticate(addr net.Addr, auth string, tx uint64) (ok bool, id string) {
	if auth == a.Password {
		return true, "user"
	}
	return false, ""
}

type MultiUserAuthenticator struct {
	Passwords map[string]any
}

func (a *MultiUserAuthenticator) Authenticate(addr net.Addr, auth string, tx uint64) (ok bool, id string) {
	username := "user"
	if index := strings.Index(auth, ":"); index >= 0 {
		username = strings.ToLower(auth[:index])
		auth = username + ":" + auth[index:]
	}
	if _, exist := a.Passwords[auth]; exist {
		return true, username
	}
	return false, ""
}

func init() {
	common.Must(internet.RegisterTransportListener(protocolName, Listen))
}
