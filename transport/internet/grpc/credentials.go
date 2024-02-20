package grpc

import (
	"context"
	_ "unsafe"

	"google.golang.org/grpc/credentials"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/security"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls/utls"
)

type securityEngineCreds struct {
	securityEngine security.Engine
	serverName     string
	ctx            context.Context
	dest           net.Destination
	streamSettings *internet.MemoryStreamConfig
}

func newSecurityEngineCreds(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (credentials.TransportCredentials, error) {
	securityEngine, err := security.CreateSecurityEngineFromSettings(ctx, streamSettings)
	if err != nil {
		return nil, newError("unable to create security engine").Base(err)
	}
	var serverName string
	switch dest.Address.Family() {
	case net.AddressFamilyDomain:
		serverName = dest.Address.Domain()
	case net.AddressFamilyIPv4, net.AddressFamilyIPv6:
		serverName = dest.Address.IP().String()
	}
	if engine, ok := securityEngine.(*utls.Engine); ok && len(engine.GetServerName()) > 0 {
		serverName = engine.GetServerName()
	}
	return &securityEngineCreds{
		securityEngine: securityEngine,
		serverName:     serverName,
		ctx:            ctx,
		dest:           dest,
		streamSettings: streamSettings,
	}, nil
}

// Info implements credentials.TransportCredentials.
func (c securityEngineCreds) Info() credentials.ProtocolInfo {
	return credentials.ProtocolInfo{
		SecurityProtocol: "tls",
		SecurityVersion:  "1.2",
		ServerName:       c.serverName,
	}
}

// ClientHandshake implements credentials.TransportCredentials.
func (c *securityEngineCreds) ClientHandshake(ctx context.Context, authority string, rawConn net.Conn) (_ net.Conn, _ credentials.AuthInfo, err error) {
	var conn security.Conn
	errChannel := make(chan error, 1)
	go func() {
		var e error
		conn, e = c.securityEngine.Client(rawConn, security.OptionWithDestination{Dest: c.dest})
		errChannel <- e
		close(errChannel)
	}()
	select {
	case err := <-errChannel:
		if err != nil {
			if conn != nil {
				conn.Close()
			}
			return nil, nil, err
		}
	case <-ctx.Done():
		if conn != nil {
			conn.Close()
		}
		return nil, nil, ctx.Err()
	}
	authInfo := securityEngineAuthInfo{
		CommonAuthInfo: credentials.CommonAuthInfo{
			SecurityLevel: credentials.PrivacyAndIntegrity,
		},
		conn: conn,
	}
	return conn, authInfo, nil
}

// ServerHandshake implements credentials.TransportCredentials.
func (c *securityEngineCreds) ServerHandshake(rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	panic("unimplemented")
}

// Clone implements credentials.TransportCredentials.
func (c *securityEngineCreds) Clone() credentials.TransportCredentials {
	creds, err := newSecurityEngineCreds(c.ctx, c.dest, c.streamSettings)
	if err != nil {
		panic(err)
	}
	return creds
}

// OverrideServerName implements credentials.TransportCredentials.
func (c *securityEngineCreds) OverrideServerName(serverNameOverride string) error {
	return nil
}

// TLSInfo contains the auth information for a TLS authenticated connection.
// It implements the AuthInfo interface.
type securityEngineAuthInfo struct {
	credentials.CommonAuthInfo
	conn security.Conn
}

// AuthType returns the type of TLSInfo as a string.
func (t securityEngineAuthInfo) AuthType() string {
	return "tls"
}

// GetSecurityValue returns security info requested by channelz.
func (t securityEngineAuthInfo) GetSecurityValue() credentials.ChannelzSecurityValue {
	switch conn := t.conn.(type) {
	case utls.UTLSClientConnection:
		state := conn.ConnectionState()
		v := &credentials.TLSChannelzSecurityValue{
			StandardName: cipherSuiteLookup(state.CipherSuite),
		}
		// Currently there's no way to get LocalCertificate info from tls package.
		if len(state.PeerCertificates) > 0 {
			v.RemoteCertificate = state.PeerCertificates[0].Raw
		}
		return v
	default:
		return nil
	}
}

//go:linkname cipherSuiteLookup google.golang.org/grpc/credentials.cipherSuiteLookup
func cipherSuiteLookup(cipherSuiteID uint16) string
