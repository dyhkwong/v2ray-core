package sip003

import (
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

var (
	PluginLoader func(plugin string) Plugin
	Plugins      map[string]func() Plugin
)

func init() {
	Plugins = make(map[string]func() Plugin)
}

func SetPluginLoader(creator func(plugin string) Plugin) {
	PluginLoader = creator
}

func RegisterPlugin(name string, creator func() Plugin) {
	Plugins[name] = creator
}

type Plugin interface {
	Init(localHost string, localPort string, remoteHost string, remotePort string, pluginOpts string, pluginArgs []string) error
	common.Closable
}

// SagerNet private
type StreamPlugin interface {
	Plugin
	InitStreamPlugin(remotePort string, pluginOpts string) error
	StreamConn(conn internet.Connection) internet.Connection
}

type ProtocolConn struct {
	buf.Reader
	buf.Writer
	ProtocolReader buf.Reader
	ProtocolWriter buf.Writer
}

// SagerNet private
type ProtocolPlugin interface {
	StreamPlugin
	InitProtocolPlugin(remoteHost string, remotePort string, pluginArgs []string, key []byte, ivSize int) error
	ProtocolConn(conn *ProtocolConn, iv []byte)
	EncodePacket(buffer *buf.Buffer, ivLen int32) (*buf.Buffer, error)
	DecodePacket(buffer *buf.Buffer) (*buf.Buffer, error)
}
