package tls

import (
	"crypto/tls"

	utls "github.com/refraction-networking/utls"

	"github.com/v2fly/v2ray-core/v4/common/buf"
	"github.com/v2fly/v2ray-core/v4/common/net"
)

//go:generate go run github.com/v2fly/v2ray-core/v4/common/errors/errorgen

var _ buf.Writer = (*Conn)(nil)

var Fingerprints = map[string]*utls.ClientHelloID{
	"chrome":     &utls.HelloChrome_Auto,
	"firefox":    &utls.HelloFirefox_Auto,
	"safari":     &utls.HelloIOS_Auto,
	"randomized": &utls.HelloRandomized,
}

type Conn struct {
	*tls.Conn
}

type UConn struct {
	*utls.UConn
}

func (c *Conn) WriteMultiBuffer(mb buf.MultiBuffer) error {
	mb = buf.Compact(mb)
	mb, err := buf.WriteMultiBuffer(c, mb)
	buf.ReleaseMulti(mb)
	return err
}

func (c *Conn) HandshakeAddress() net.Address {
	if err := c.Handshake(); err != nil {
		return nil
	}
	state := c.ConnectionState()
	if state.ServerName == "" {
		return nil
	}
	return net.ParseAddress(state.ServerName)
}

func (c *UConn) HandshakeAddress() net.Address {
	if err := c.Handshake(); err != nil {
		return nil
	}
	state := c.ConnectionState()
	if state.ServerName == "" {
		return nil
	}
	return net.ParseAddress(state.ServerName)
}

// Client initiates a TLS client handshake on the given connection.
func Client(c net.Conn, config *tls.Config) net.Conn {
	tlsConn := tls.Client(c, config)
	return &Conn{Conn: tlsConn}
}

func copyConfig(c *tls.Config) *utls.Config {
	return &utls.Config{
		RootCAs:            c.RootCAs,
		NextProtos:         c.NextProtos,
		ServerName:         c.ServerName,
		InsecureSkipVerify: c.InsecureSkipVerify,
		MinVersion:         c.MinVersion,
		MaxVersion:         c.MaxVersion,
	}
}

func UClient(c net.Conn, config *tls.Config, fingerprint *utls.ClientHelloID) net.Conn {
	utlsConn := utls.UClient(c, copyConfig(config), *fingerprint)
	return &UConn{UConn: utlsConn}
}

// Server initiates a TLS server handshake on the given connection.
func Server(c net.Conn, config *tls.Config) net.Conn {
	tlsConn := tls.Server(c, config)
	return &Conn{Conn: tlsConn}
}
