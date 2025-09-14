package singbridge

import (
	"crypto/tls"

	singtls "github.com/sagernet/sing/common/tls"

	"github.com/v2fly/v2ray-core/v5/common/net"
)

var _ singtls.Config = (*tlsConfigWrapper)(nil)

func NewTLSConfigWrapper(config *tls.Config) *tlsConfigWrapper {
	return &tlsConfigWrapper{
		config: config,
	}
}

type tlsConfigWrapper struct {
	config *tls.Config
}

func (c *tlsConfigWrapper) ServerName() string {
	return c.config.ServerName
}

func (c *tlsConfigWrapper) SetServerName(serverName string) {
	c.config.ServerName = serverName
}

func (c *tlsConfigWrapper) NextProtos() []string {
	return c.config.NextProtos
}

func (c *tlsConfigWrapper) SetNextProtos(nextProto []string) {
	c.config.NextProtos = nextProto
}

func (c *tlsConfigWrapper) Config() (*tls.Config, error) {
	return c.config, nil
}

func (c *tlsConfigWrapper) Client(conn net.Conn) (singtls.Conn, error) {
	return tls.Client(conn, c.config), nil
}

func (c *tlsConfigWrapper) Clone() singtls.Config {
	return &tlsConfigWrapper{c.config.Clone()}
}
