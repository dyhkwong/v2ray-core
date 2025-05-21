package quic

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/quic-go/quic-go"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/task"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

type connectionContext struct {
	rawConn net.PacketConn
	conn    quic.EarlyConnection
}

var errConnectionClosed = newError("connection closed")

func (cc *connectionContext) openStream(destAddr net.Addr) (*interConn, error) {
	if !isActive(cc.conn) {
		return nil, errConnectionClosed
	}

	stream, err := cc.conn.OpenStream()
	if err != nil {
		return nil, err
	}

	conn := &interConn{
		stream: stream,
		local:  cc.conn.LocalAddr(),
		remote: destAddr,
	}

	return conn, nil
}

type dialerConf struct {
	net.Destination
	*internet.MemoryStreamConfig
}

type clientConnections struct {
	access  sync.Mutex
	conns   []*connectionContext
	cleanup *task.Periodic
}

func isActive(c quic.EarlyConnection) bool {
	select {
	case <-c.Context().Done():
		return false
	default:
		return true
	}
}

func (c *clientConnections) removeInactiveConnections() {
	c.conns = slices.DeleteFunc(c.conns, func(cc *connectionContext) bool {
		if isActive(cc.conn) {
			return false
		}
		if err := cc.conn.CloseWithError(0, ""); err != nil {
			newError("failed to close connection").Base(err).WriteToLog()
		}
		if err := cc.rawConn.Close(); err != nil {
			newError("failed to close raw connection").Base(err).WriteToLog()
		}
		return true
	})
}

func (c *clientConnections) openStream(destAddr net.Addr) *interConn {
	for _, c := range c.conns {
		if !isActive(c.conn) {
			continue
		}

		conn, err := c.openStream(destAddr)
		if err != nil {
			continue
		}

		return conn
	}

	return nil
}

func (c *clientConnections) cleanConnections() error {
	c.access.Lock()
	c.removeInactiveConnections()
	c.access.Unlock()
	return nil
}

func (c *clientConnections) openConnection(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (internet.Connection, error) {
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

	if conn := c.openStream(destAddr); conn != nil {
		return conn, nil
	}

	c.access.Lock()
	c.removeInactiveConnections()
	c.access.Unlock()

	newError("dialing QUIC to ", dest).WriteToLog()

	detachedContext := core.ToBackgroundDetachedContext(ctx)
	rawConn, err := internet.DialSystem(detachedContext, dest, streamSettings.SocketSettings)
	if err != nil {
		return nil, newError("failed to dial to dest: ", err).AtWarning().Base(err)
	}

	quicConfig := &quic.Config{
		HandshakeIdleTimeout: time.Second * 8,
		MaxIdleTimeout:       time.Second * 30,
		KeepAlivePeriod:      time.Second * 15,
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

	sysConn, err := wrapSysConn(pc, streamSettings.ProtocolSettings.(*Config))
	if err != nil {
		rawConn.Close()
		return nil, err
	}

	tr := quic.Transport{
		Conn:               sysConn,
		ConnectionIDLength: 12,
	}

	tlsConfig := tls.ConfigFromStreamSettings(streamSettings)
	if tlsConfig == nil {
		tlsConfig = &tls.Config{
			ServerName:    internalDomain,
			AllowInsecure: true,
		}
	}

	conn, err := tr.DialEarly(detachedContext, destAddr, tlsConfig.GetTLSConfig(tls.WithDestination(dest)), quicConfig)
	if err != nil {
		sysConn.Close()
		return nil, err
	}

	cc := &connectionContext{
		conn:    conn,
		rawConn: sysConn,
	}

	c.access.Lock()
	c.conns = append(c.conns, cc)
	c.access.Unlock()
	return cc.openStream(destAddr)
}

var (
	globalDialerMap    map[dialerConf]*clientConnections
	globalDialerAccess sync.Mutex
)

func Dial(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (internet.Connection, error) {
	dest.Network = net.Network_UDP
	globalDialerAccess.Lock()
	if globalDialerMap == nil {
		globalDialerMap = make(map[dialerConf]*clientConnections)
	}
	client, found := globalDialerMap[dialerConf{dest, streamSettings}]
	if !found {
		client = new(clientConnections)
		client.cleanup = &task.Periodic{
			Interval: time.Minute,
			Execute:  client.cleanConnections,
		}
		common.Must(client.cleanup.Start())
		globalDialerMap[dialerConf{dest, streamSettings}] = client
	}
	globalDialerAccess.Unlock()
	return client.openConnection(ctx, dest, streamSettings)
}

func init() {
	common.Must(internet.RegisterTransportDialer(protocolName, Dial))
}
