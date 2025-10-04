package quic

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/quic-go/quic-go"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/environment"
	"github.com/v2fly/v2ray-core/v5/common/environment/envctx"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/task"
	"github.com/v2fly/v2ray-core/v5/features/dns/localdns"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

type connectionContext struct {
	rawConn net.PacketConn
	conn    *quic.Conn
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

func isActive(c *quic.Conn) bool {
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

func (c *clientConnections) Close() error {
	c.access.Lock()
	for _, cc := range c.conns {
		_ = cc.conn.CloseWithError(0, "")
		_ = cc.rawConn.Close()
	}
	clear(c.conns)
	_ = c.cleanup.Close()
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
		addr, err := localdns.New().LookupIP(dest.Address.Domain())
		if err != nil {
			return nil, err
		}
		destAddr = &net.UDPAddr{
			IP:   addr[0],
			Port: int(dest.Port),
		}
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

	var packetConn net.PacketConn
	switch rawConn := rawConn.(type) {
	case *internet.PacketConnWrapper:
		packetConn = rawConn.Conn
	case net.PacketConn:
		packetConn = rawConn
	default:
		packetConn = internet.NewConnWrapper(rawConn)
	}

	sysConn, err := wrapSysConn(packetConn, streamSettings.ProtocolSettings.(*Config))
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

type transportConnectionState struct {
	scopedDialerMap    map[dialerConf]*clientConnections
	scopedDialerAccess sync.Mutex
}

func (t *transportConnectionState) IsTransientStorageLifecycleReceiver() {
}

func (t *transportConnectionState) Close() error {
	t.scopedDialerAccess.Lock()
	for _, conn := range t.scopedDialerMap {
		_ = conn.Close()
	}
	clear(t.scopedDialerMap)
	t.scopedDialerAccess.Unlock()
	return nil
}

func Dial(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (internet.Connection, error) {
	dest.Network = net.Network_UDP
	transportEnvironment := envctx.EnvironmentFromContext(ctx).(environment.TransportEnvironment)
	state, err := transportEnvironment.TransientStorage().Get(ctx, "quic-transport-connection-state")
	if err != nil {
		state = &transportConnectionState{}
		transportEnvironment.TransientStorage().Put(ctx, "quic-transport-connection-state", state)
		state, err = transportEnvironment.TransientStorage().Get(ctx, "quic-transport-connection-state")
		if err != nil {
			return nil, newError("failed to get quic transport connection state").Base(err)
		}
	}
	stateTyped := state.(*transportConnectionState)
	stateTyped.scopedDialerAccess.Lock()
	if stateTyped.scopedDialerMap == nil {
		stateTyped.scopedDialerMap = make(map[dialerConf]*clientConnections)
	}
	client, found := stateTyped.scopedDialerMap[dialerConf{dest, streamSettings}]
	if !found {
		client = new(clientConnections)
		client.cleanup = &task.Periodic{
			Interval: time.Minute,
			Execute:  client.cleanConnections,
		}
		common.Must(client.cleanup.Start())
		stateTyped.scopedDialerMap[dialerConf{dest, streamSettings}] = client
	}
	stateTyped.scopedDialerAccess.Unlock()
	return client.openConnection(ctx, dest, streamSettings)
}

func init() {
	common.Must(internet.RegisterTransportDialer(protocolName, Dial))
}
