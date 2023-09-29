//go:build !wasm
// +build !wasm

package domainsocket

import (
	"context"
	gotls "crypto/tls"
	"os"
	"strings"

	goreality "github.com/xtls/reality"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/reality"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

type Listener struct {
	addr          *net.UnixAddr
	ln            net.Listener
	tlsConfig     *gotls.Config
	realityConfig *goreality.Config
	config        *Config
	addConn       internet.ConnHandler
	locker        *fileLocker
}

func Listen(ctx context.Context, address net.Address, port net.Port, streamSettings *internet.MemoryStreamConfig, handler internet.ConnHandler) (internet.Listener, error) {
	settings := streamSettings.ProtocolSettings.(*Config)
	addr, err := settings.GetUnixAddr()
	if err != nil {
		return nil, err
	}

	unixListener, err := net.ListenUnix("unix", addr)
	if err != nil {
		return nil, newError("failed to listen domain socket").Base(err).AtWarning()
	}

	ln := &Listener{
		addr:    addr,
		ln:      unixListener,
		config:  settings,
		addConn: handler,
	}

	if !settings.Abstract {
		ln.locker = &fileLocker{
			path: settings.Path + ".lock",
		}
		if err := ln.locker.Acquire(); err != nil {
			unixListener.Close()
			return nil, err
		}
	}

	if config := tls.ConfigFromStreamSettings(streamSettings); config != nil {
		ln.tlsConfig = config.GetTLSConfig()
	} else if config := reality.ConfigFromStreamSettings(streamSettings); config != nil {
		ln.realityConfig = config.GetREALITYConfig()
	}

	go ln.run()

	return ln, nil
}

func (ln *Listener) Addr() net.Addr {
	return ln.addr
}

func (ln *Listener) Close() error {
	if ln.locker != nil {
		ln.locker.Release()
	}
	return ln.ln.Close()
}

func (ln *Listener) run() {
	for {
		conn, err := ln.ln.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "closed") {
				break
			}
			newError("failed to accepted raw connections").Base(err).AtWarning().WriteToLog()
			continue
		}
		go func() {
			if ln.tlsConfig != nil {
				conn = tls.Server(conn, ln.tlsConfig)
			} else if ln.realityConfig != nil {
				if conn, err = reality.Server(conn, ln.realityConfig); err != nil {
					newError(err).AtInfo().WriteToLog()
					return
				}
			}
			ln.addConn(internet.Connection(conn))
		}()
	}
}

type fileLocker struct {
	path string
	file *os.File
}

func init() {
	common.Must(internet.RegisterTransportListener(protocolName, Listen))
}
