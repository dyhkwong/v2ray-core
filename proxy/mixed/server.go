package mixed

import (
	"context"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/proxy/http"
	"github.com/v2fly/v2ray-core/v5/proxy/socks"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

const (
	socks5Version = 0x05
	socks4Version = 0x04
)

// Server is a Mixed proxy server
type Server struct {
	httpServer        http.Server
	socksServer       socks.Server
	socksOnlyNetworks []net.Network
	httpOnlyNetworks  []net.Network
	intersectNetworks []net.Network
}

// NewServer creates a new Server object.
func NewServer(ctx context.Context, config *ServerConfig) (*Server, error) {
	httpServer, err := http.NewServer(
		ctx,
		&http.ServerConfig{
			Timeout:          config.Timeout,
			Accounts:         config.Accounts,
			AllowTransparent: config.AllowTransparent,
			UserLevel:        config.UserLevel,
		},
	)
	if err != nil {
		return nil, newError("Errors in http config").Base(err).AtError()
	}
	socksAuthType := socks.AuthType_NO_AUTH
	if len(config.Accounts) > 0 {
		socksAuthType = socks.AuthType_PASSWORD
	}
	socksServer, err := socks.NewServer(
		ctx,
		&socks.ServerConfig{
			AuthType:       socksAuthType,
			Accounts:       config.Accounts,
			Address:        config.Address,
			UdpEnabled:     config.UdpEnabled,
			Timeout:        config.Timeout,
			UserLevel:      config.UserLevel,
			PacketEncoding: config.PacketEncoding,
			DeferLastReply: config.DeferLastReply,
		})
	if err != nil {
		return nil, newError("Errors in socks config").Base(err).AtError()
	}

	socksNetworks := socksServer.Network()
	httpNetworks := httpServer.Network()

	socksOnlyNetworks := make([]net.Network, 0)
	httpOnlyNetworks := make([]net.Network, 0)
	intersectNetworks := make([]net.Network, 0)

	for _, n := range socksNetworks {
		if !isInNetworkSlice(n, &httpNetworks) {
			socksOnlyNetworks = append(socksOnlyNetworks, n)
		} else {
			intersectNetworks = append(intersectNetworks, n)
		}
	}

	for _, n := range httpNetworks {
		if !isInNetworkSlice(n, &socksNetworks) {
			httpOnlyNetworks = append(httpOnlyNetworks, n)
		}
	}

	s := &Server{
		httpServer:        *httpServer,
		socksServer:       *socksServer,
		socksOnlyNetworks: socksOnlyNetworks,
		httpOnlyNetworks:  httpOnlyNetworks,
		intersectNetworks: intersectNetworks,
	}
	return s, nil
}

// Network implements proxy.Inbound.
func (s *Server) Network() []net.Network {
	returnNetwork := s.intersectNetworks
	returnNetwork = append(returnNetwork, s.socksOnlyNetworks...)
	returnNetwork = append(returnNetwork, s.httpOnlyNetworks...)
	return returnNetwork
}

// Process implements proxy.Inbound.
func (s *Server) Process(ctx context.Context, network net.Network, conn internet.Connection, dispatcher routing.Dispatcher) error {
	// Socks only
	if isInNetworkSlice(network, &s.socksOnlyNetworks) {
		newError("Connection is identified as Socks").AtDebug().WriteToLog(session.ExportIDToError(ctx))
		return s.socksServer.Process(ctx, network, conn, dispatcher)
	}

	// HTTP only
	if isInNetworkSlice(network, &s.httpOnlyNetworks) {
		newError("Connection is identified as HTTP").AtDebug().WriteToLog(session.ExportIDToError(ctx))
		return s.httpServer.Process(ctx, network, conn, dispatcher)
	}

	// no UDP for BufferedConnection
	if network == net.Network_UDP {
		return newError("UDP is only available for Socks").AtError()
	}

	// read first byte to distinguish HTTP and Socks
	bufferedConnection := NewBufferedConnection(conn)
	firstByte, err := bufferedConnection.Peek(1)
	if err != nil {
		return newError("Read first byte failed").Base(err).AtError()
	}
	newError("First byte", firstByte).AtDebug().WriteToLog(session.ExportIDToError(ctx))

	if firstByte[0] == socks4Version || firstByte[0] == socks5Version {
		newError("Connection is identified as Socks").AtDebug().WriteToLog(session.ExportIDToError(ctx))
		return s.socksServer.Process(ctx, network, bufferedConnection, dispatcher)
	}
	newError("Connection is identified as HTTP").AtDebug().WriteToLog(session.ExportIDToError(ctx))
	return s.httpServer.Process(ctx, network, bufferedConnection, dispatcher)
}

func init() {
	common.Must(common.RegisterConfig((*ServerConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewServer(ctx, config.(*ServerConfig))
	}))
}

// isInNetworkSlice gets whether the network is in slice
func isInNetworkSlice(network net.Network, networks *[]net.Network) bool {
	found := false
	for _, n := range *networks {
		if network == n {
			found = true
			break
		}
	}
	return found
}
