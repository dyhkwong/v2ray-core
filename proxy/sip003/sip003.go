package sip003

import (
	"github.com/v2fly/v2ray-core/v5/common"
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
	Init(localHost string, localPort string, remoteHost string, remotePort string, pluginOpts string, pluginArgs []string, workingDir string) error
	common.Closable
}
