package hysteria2_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/environment"
	"github.com/v2fly/v2ray-core/v5/common/environment/deferredpersistentstorage"
	"github.com/v2fly/v2ray-core/v5/common/environment/envctx"
	"github.com/v2fly/v2ray-core/v5/common/environment/filesystemimpl"
	"github.com/v2fly/v2ray-core/v5/common/environment/systemnetworkimpl"
	"github.com/v2fly/v2ray-core/v5/common/environment/transientstorageimpl"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/protocol/tls/cert"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/testing/servers/udp"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/hysteria2"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

func TestTCP(t *testing.T) {
	port := udp.PickPort()

	listener, err := hysteria2.Listen(context.Background(), net.LocalHostIP, port, &internet.MemoryStreamConfig{
		ProtocolName:     "hysteria2",
		ProtocolSettings: &hysteria2.Config{Password: "123"},
		SecurityType:     "tls",
		SecuritySettings: &tls.Config{
			Certificate: []*tls.Certificate{
				tls.ParseCertificate(
					cert.MustGenerate(nil,
						cert.DNSNames("www.v2fly.org"),
					),
				),
			},
		},
	}, func(conn internet.Connection) {
		go func() {
			defer conn.Close()

			b := buf.New()
			defer b.Release()

			for {
				b.Clear()
				if _, err := b.ReadFrom(conn); err != nil {
					fmt.Println(err)
					return
				}
				common.Must2(conn.Write(b.Bytes()))
			}
		}()
	})
	common.Must(err)

	defer listener.Close()

	time.Sleep(time.Second)

	dctx := context.Background()
	defaultNetworkImpl := systemnetworkimpl.NewSystemNetworkDefault()
	defaultFilesystemImpl := filesystemimpl.NewDefaultFileSystemDefaultImpl()
	deferredPersistentStorageImpl := deferredpersistentstorage.NewDeferredPersistentStorage(dctx)
	rootEnv := environment.NewRootEnvImpl(dctx,
		transientstorageimpl.NewScopedTransientStorageImpl(), defaultNetworkImpl.Dialer(), defaultNetworkImpl.Listener(),
		defaultFilesystemImpl, deferredPersistentStorageImpl)
	proxyEnvironment := rootEnv.ProxyEnvironment("o")
	transportEnvironment, err := proxyEnvironment.NarrowScopeToTransport("hysteria2")
	if err != nil {
		t.Fatal(err)
	}
	dctx = envctx.ContextWithEnvironment(dctx, transportEnvironment)

	conn, err := hysteria2.Dial(dctx, net.TCPDestination(net.LocalHostIP, port), &internet.MemoryStreamConfig{
		ProtocolName:     "hysteria2",
		ProtocolSettings: &hysteria2.Config{Password: "123"},
		SecurityType:     "tls",
		SecuritySettings: &tls.Config{
			ServerName:    "www.v2fly.org",
			AllowInsecure: true,
		},
	})
	common.Must(err)
	defer conn.Close()

	const N = 1000
	b1 := make([]byte, N)
	common.Must2(rand.Read(b1))
	b2 := buf.New()

	common.Must2(conn.Write(b1))

	b2.Clear()
	common.Must2(b2.ReadFullFrom(conn, N))
	if r := cmp.Diff(b2.Bytes(), b1); r != "" {
		t.Error(r)
	}
}

func TestUDP(t *testing.T) {
	port := udp.PickPort()

	listener, err := hysteria2.Listen(context.Background(), net.LocalHostIP, port, &internet.MemoryStreamConfig{
		ProtocolName:     "hysteria2",
		ProtocolSettings: &hysteria2.Config{Password: "123", UseUdpExtension: true},
		SecurityType:     "tls",
		SecuritySettings: &tls.Config{
			Certificate: []*tls.Certificate{
				tls.ParseCertificate(
					cert.MustGenerate(nil,
						cert.DNSNames("www.v2fly.org"),
					),
				),
			},
		},
	}, func(conn internet.Connection) {
		fmt.Println("incoming")
		go func() {
			defer conn.Close()

			b := buf.New()
			defer b.Release()

			for {
				b.Clear()
				if _, err := b.ReadFrom(conn); err != nil {
					fmt.Println(err)
					return
				}
				common.Must2(conn.Write(b.Bytes()))
			}
		}()
	})
	common.Must(err)

	defer listener.Close()

	time.Sleep(time.Second)

	address, err := net.ParseDestination("udp:127.0.0.1:1180")
	common.Must(err)

	dctx := session.ContextWithOutbound(context.Background(), &session.Outbound{Target: address})
	defaultNetworkImpl := systemnetworkimpl.NewSystemNetworkDefault()
	defaultFilesystemImpl := filesystemimpl.NewDefaultFileSystemDefaultImpl()
	deferredPersistentStorageImpl := deferredpersistentstorage.NewDeferredPersistentStorage(dctx)
	rootEnv := environment.NewRootEnvImpl(dctx,
		transientstorageimpl.NewScopedTransientStorageImpl(), defaultNetworkImpl.Dialer(), defaultNetworkImpl.Listener(),
		defaultFilesystemImpl, deferredPersistentStorageImpl)
	proxyEnvironment := rootEnv.ProxyEnvironment("o")
	transportEnvironment, err := proxyEnvironment.NarrowScopeToTransport("hysteria2")
	if err != nil {
		t.Fatal(err)
	}
	dctx = envctx.ContextWithEnvironment(dctx, transportEnvironment)

	conn, err := hysteria2.Dial(dctx, net.TCPDestination(net.LocalHostIP, port), &internet.MemoryStreamConfig{
		ProtocolName:     "hysteria2",
		ProtocolSettings: &hysteria2.Config{Password: "123", UseUdpExtension: true},
		SecurityType:     "tls",
		SecuritySettings: &tls.Config{
			ServerName:    "www.v2fly.org",
			AllowInsecure: true,
		},
	})
	common.Must(err)
	defer conn.Close()

	const N = 1000
	b1 := make([]byte, N)
	common.Must2(rand.Read(b1))
	common.Must2(conn.Write(b1))

	b2 := buf.New()
	b2.Clear()
	common.Must2(b2.ReadFullFrom(conn, N))
	if r := cmp.Diff(b2.Bytes(), b1); r != "" {
		t.Error(r)
	}
}
