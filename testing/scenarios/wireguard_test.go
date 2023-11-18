package scenarios

import (
	"encoding/base64"
	"encoding/hex"
	"testing"

	"google.golang.org/protobuf/types/known/anypb"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/app/log"
	"github.com/v2fly/v2ray-core/v5/app/proxyman"
	"github.com/v2fly/v2ray-core/v5/common"
	clog "github.com/v2fly/v2ray-core/v5/common/log"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/serial"
	"github.com/v2fly/v2ray-core/v5/proxy/dokodemo"
	"github.com/v2fly/v2ray-core/v5/proxy/freedom"
	"github.com/v2fly/v2ray-core/v5/proxy/wireguard"
	"github.com/v2fly/v2ray-core/v5/testing/servers/tcp"
	"github.com/v2fly/v2ray-core/v5/testing/servers/udp"
)

func TestWireguard(t *testing.T) {
	tcpServer := tcp.Server{
		MsgProcessor: xor,
	}
	dest, err := tcpServer.Start()
	common.Must(err)
	defer tcpServer.Close()

	serverPrivate, _ := base64.StdEncoding.DecodeString("EGs4lTSJPmgELx6YiJAmPR2meWi6bY+e9rTdCipSj10=")
	serverPublic, _ := base64.StdEncoding.DecodeString("osAMIyil18HeZXGGBDC9KpZoM+L2iGyXWVSYivuM9B0=")
	clientPrivate, _ := base64.StdEncoding.DecodeString("CPQSpgxgdQRZa5SUbT3HLv+mmDVHLW5YR/rQlzum/2I=")
	clientPublic, _ := base64.StdEncoding.DecodeString("MmLJ5iHFVVBp7VsB0hxfpQ0wEzAbT2KQnpQpj0+RtBw=")

	serverPort := udp.PickPort()
	serverConfig := &core.Config{
		App: []*anypb.Any{
			serial.ToTypedMessage(&log.Config{
				Error: &log.LogSpecification{Level: clog.Severity_Debug, Type: log.LogType_Console},
			}),
		},
		Inbound: []*core.InboundHandlerConfig{
			{
				ReceiverSettings: serial.ToTypedMessage(&proxyman.ReceiverConfig{
					PortRange: net.SinglePortRange(serverPort),
					Listen:    net.NewIPOrDomain(net.LocalHostIP),
				}),
				ProxySettings: serial.ToTypedMessage(&wireguard.ServerConfig{
					Address:   []string{"10.0.0.1"},
					Mtu:       1420,
					SecretKey: hex.EncodeToString(serverPrivate),
					Peers: []*wireguard.PeerConfig{{
						PublicKey:  hex.EncodeToString(serverPublic),
						AllowedIps: []string{"0.0.0.0/0", "::0/0"},
					}},
				}),
			},
		},
		Outbound: []*core.OutboundHandlerConfig{
			{
				ProxySettings: serial.ToTypedMessage(&freedom.Config{}),
			},
		},
	}

	clientPort := tcp.PickPort()
	clientConfig := &core.Config{
		App: []*anypb.Any{
			serial.ToTypedMessage(&log.Config{
				Error: &log.LogSpecification{Level: clog.Severity_Debug, Type: log.LogType_Console},
			}),
		},
		Inbound: []*core.InboundHandlerConfig{
			{
				ReceiverSettings: serial.ToTypedMessage(&proxyman.ReceiverConfig{
					PortRange: net.SinglePortRange(clientPort),
					Listen:    net.NewIPOrDomain(net.LocalHostIP),
				}),
				ProxySettings: serial.ToTypedMessage(&dokodemo.Config{
					Address:  net.NewIPOrDomain(dest.Address),
					Port:     uint32(dest.Port),
					Networks: []net.Network{net.Network_TCP},
				}),
			},
		},
		Outbound: []*core.OutboundHandlerConfig{
			{
				ProxySettings: serial.ToTypedMessage(&wireguard.ClientConfig{
					Address:   []string{"10.0.0.2"},
					Mtu:       1420,
					SecretKey: hex.EncodeToString(clientPrivate),
					Peers: []*wireguard.PeerConfig{{
						Endpoint:   net.LocalHostIP.String() + ":" + serverPort.String(),
						PublicKey:  hex.EncodeToString(clientPublic),
						AllowedIps: []string{"0.0.0.0/0", "::0/0"},
					}},
				}),
			},
		},
	}

	servers, err := InitializeServerConfigs(serverConfig, clientConfig)
	common.Must(err)
	defer CloseAllServers(servers)
}
