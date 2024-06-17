package splithttp_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"runtime"
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
	"github.com/v2fly/v2ray-core/v5/testing/servers/tcp"
	"github.com/v2fly/v2ray-core/v5/testing/servers/udp"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	. "github.com/v2fly/v2ray-core/v5/transport/internet/splithttp"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

func Test_listenSHAndDial(t *testing.T) {
	listenPort := tcp.PickPort()
	listen, err := ListenSH(context.Background(), net.LocalHostIP, listenPort, &internet.MemoryStreamConfig{
		ProtocolName: "splithttp",
		ProtocolSettings: &Config{
			Path: "/sh",
		},
	}, func(conn internet.Connection) {
		go func(c internet.Connection) {
			defer c.Close()

			var b [1024]byte
			c.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, err := c.Read(b[:])
			if err != nil {
				return
			}

			common.Must2(c.Write([]byte("Response")))
		}(conn)
	})
	common.Must(err)

	ctx := context.Background()
	defaultNetworkImpl := systemnetworkimpl.NewSystemNetworkDefault()
	defaultFilesystemImpl := filesystemimpl.NewDefaultFileSystemDefaultImpl()
	deferredPersistentStorageImpl := deferredpersistentstorage.NewDeferredPersistentStorage(ctx)
	rootEnv := environment.NewRootEnvImpl(ctx,
		transientstorageimpl.NewScopedTransientStorageImpl(), defaultNetworkImpl.Dialer(), defaultNetworkImpl.Listener(),
		defaultFilesystemImpl, deferredPersistentStorageImpl)
	proxyEnvironment := rootEnv.ProxyEnvironment("o")
	transportEnvironment, err := proxyEnvironment.NarrowScopeToTransport("splithttp")
	if err != nil {
		t.Fatal(err)
	}
	ctx = envctx.ContextWithEnvironment(ctx, transportEnvironment)

	streamSettings := &internet.MemoryStreamConfig{
		ProtocolName:     "splithttp",
		ProtocolSettings: &Config{Path: "sh"},
	}
	conn, err := Dial(ctx, net.TCPDestination(net.LocalHostIP, listenPort), streamSettings)

	common.Must(err)
	_, err = conn.Write([]byte("Test connection 1"))
	common.Must(err)

	var b [1024]byte
	fmt.Println("test2")
	n, _ := io.ReadFull(conn, b[:])
	fmt.Println("string is", n)
	if string(b[:n]) != "Response" {
		t.Error("response: ", string(b[:n]))
	}

	common.Must(conn.Close())
	conn, err = Dial(ctx, net.TCPDestination(net.LocalHostIP, listenPort), streamSettings)

	common.Must(err)
	_, err = conn.Write([]byte("Test connection 2"))
	common.Must(err)
	n, _ = io.ReadFull(conn, b[:])
	common.Must(err)
	if string(b[:n]) != "Response" {
		t.Error("response: ", string(b[:n]))
	}
	common.Must(conn.Close())

	common.Must(listen.Close())
}

func TestDialWithRemoteAddr(t *testing.T) {
	listenPort := tcp.PickPort()
	listen, err := ListenSH(context.Background(), net.LocalHostIP, listenPort, &internet.MemoryStreamConfig{
		ProtocolName: "splithttp",
		ProtocolSettings: &Config{
			Path: "sh",
		},
	}, func(conn internet.Connection) {
		go func(c internet.Connection) {
			defer c.Close()

			var b [1024]byte
			_, err := c.Read(b[:])
			// common.Must(err)
			if err != nil {
				return
			}

			_, err = c.Write([]byte(c.RemoteAddr().String()))
			common.Must(err)
		}(conn)
	})
	common.Must(err)

	ctx := context.Background()
	defaultNetworkImpl := systemnetworkimpl.NewSystemNetworkDefault()
	defaultFilesystemImpl := filesystemimpl.NewDefaultFileSystemDefaultImpl()
	deferredPersistentStorageImpl := deferredpersistentstorage.NewDeferredPersistentStorage(ctx)
	rootEnv := environment.NewRootEnvImpl(ctx,
		transientstorageimpl.NewScopedTransientStorageImpl(), defaultNetworkImpl.Dialer(), defaultNetworkImpl.Listener(),
		defaultFilesystemImpl, deferredPersistentStorageImpl)
	proxyEnvironment := rootEnv.ProxyEnvironment("o")
	transportEnvironment, err := proxyEnvironment.NarrowScopeToTransport("splithttp")
	if err != nil {
		t.Fatal(err)
	}
	ctx = envctx.ContextWithEnvironment(ctx, transportEnvironment)

	conn, err := Dial(ctx, net.TCPDestination(net.LocalHostIP, listenPort), &internet.MemoryStreamConfig{
		ProtocolName:     "splithttp",
		ProtocolSettings: &Config{Path: "sh", Headers: map[string]string{"X-Forwarded-For": "1.1.1.1"}},
	})

	common.Must(err)
	_, err = conn.Write([]byte("Test connection 1"))
	common.Must(err)

	var b [1024]byte
	n, _ := io.ReadFull(conn, b[:])
	if string(b[:n]) != "1.1.1.1:0" {
		t.Error("response: ", string(b[:n]))
	}

	common.Must(listen.Close())
}

func Test_listenSHAndDial_TLS(t *testing.T) {
	if runtime.GOARCH == "arm64" {
		return
	}

	listenPort := tcp.PickPort()

	start := time.Now()

	streamSettings := &internet.MemoryStreamConfig{
		ProtocolName: "splithttp",
		ProtocolSettings: &Config{
			Path: "shs",
		},
		SecurityType: "tls",
		SecuritySettings: &tls.Config{
			AllowInsecure: true,
			Certificate:   []*tls.Certificate{tls.ParseCertificate(cert.MustGenerate(nil, cert.CommonName("localhost")))},
		},
	}
	listen, err := ListenSH(context.Background(), net.LocalHostIP, listenPort, streamSettings, func(conn internet.Connection) {
		go func() {
			defer conn.Close()

			var b [1024]byte
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, err := conn.Read(b[:])
			if err != nil {
				return
			}

			common.Must2(conn.Write([]byte("Response")))
		}()
	})
	common.Must(err)
	defer listen.Close()

	ctx := context.Background()
	defaultNetworkImpl := systemnetworkimpl.NewSystemNetworkDefault()
	defaultFilesystemImpl := filesystemimpl.NewDefaultFileSystemDefaultImpl()
	deferredPersistentStorageImpl := deferredpersistentstorage.NewDeferredPersistentStorage(ctx)
	rootEnv := environment.NewRootEnvImpl(ctx,
		transientstorageimpl.NewScopedTransientStorageImpl(), defaultNetworkImpl.Dialer(), defaultNetworkImpl.Listener(),
		defaultFilesystemImpl, deferredPersistentStorageImpl)
	proxyEnvironment := rootEnv.ProxyEnvironment("o")
	transportEnvironment, err := proxyEnvironment.NarrowScopeToTransport("splithttp")
	if err != nil {
		t.Fatal(err)
	}
	ctx = envctx.ContextWithEnvironment(ctx, transportEnvironment)

	conn, err := Dial(ctx, net.TCPDestination(net.LocalHostIP, listenPort), streamSettings)
	common.Must(err)

	_, err = conn.Write([]byte("Test connection 1"))
	common.Must(err)

	var b [1024]byte
	n, _ := io.ReadFull(conn, b[:])
	if string(b[:n]) != "Response" {
		t.Error("response: ", string(b[:n]))
	}

	end := time.Now()
	if !end.Before(start.Add(time.Second * 5)) {
		t.Error("end: ", end, " start: ", start)
	}
}

func Test_listenSHAndDial_H2C(t *testing.T) {
	if runtime.GOARCH == "arm64" {
		return
	}

	listenPort := tcp.PickPort()

	streamSettings := &internet.MemoryStreamConfig{
		ProtocolName: "splithttp",
		ProtocolSettings: &Config{
			Path: "shs",
		},
	}
	listen, err := ListenSH(context.Background(), net.LocalHostIP, listenPort, streamSettings, func(conn internet.Connection) {
		go func() {
			_ = conn.Close()
		}()
	})
	common.Must(err)
	defer listen.Close()

	protocols := new(http.Protocols)
	protocols.SetUnencryptedHTTP2(true)
	client := http.Client{
		Transport: &http.Transport{
			Protocols: protocols,
		},
	}

	resp, err := client.Get("http://" + net.LocalHostIP.String() + ":" + listenPort.String())
	common.Must(err)

	if resp.StatusCode != 404 {
		t.Error("Expected 404 but got:", resp.StatusCode)
	}

	if resp.ProtoMajor != 2 {
		t.Error("Expected h2 but got:", resp.ProtoMajor)
	}
}

func Test_listenSHAndDial_QUIC(t *testing.T) {
	if runtime.GOARCH == "arm64" {
		return
	}

	listenPort := udp.PickPort()

	start := time.Now()

	streamSettings := &internet.MemoryStreamConfig{
		ProtocolName: "splithttp",
		ProtocolSettings: &Config{
			Path: "shs",
		},
		SecurityType: "tls",
		SecuritySettings: &tls.Config{
			AllowInsecure: true,
			Certificate:   []*tls.Certificate{tls.ParseCertificate(cert.MustGenerate(nil, cert.CommonName("localhost")))},
			NextProtocol:  []string{"h3"},
		},
	}
	serverClosed := false
	listen, err := ListenSH(context.Background(), net.LocalHostIP, listenPort, streamSettings, func(conn internet.Connection) {
		go func() {
			defer conn.Close()

			b := buf.New()
			defer b.Release()

			for {
				b.Clear()
				if _, err := b.ReadFrom(conn); err != nil {
					break
				}
				common.Must2(conn.Write(b.Bytes()))
			}

			serverClosed = true
		}()
	})
	common.Must(err)
	defer listen.Close()

	time.Sleep(time.Second)

	ctx := context.Background()
	defaultNetworkImpl := systemnetworkimpl.NewSystemNetworkDefault()
	defaultFilesystemImpl := filesystemimpl.NewDefaultFileSystemDefaultImpl()
	deferredPersistentStorageImpl := deferredpersistentstorage.NewDeferredPersistentStorage(ctx)
	rootEnv := environment.NewRootEnvImpl(ctx,
		transientstorageimpl.NewScopedTransientStorageImpl(), defaultNetworkImpl.Dialer(), defaultNetworkImpl.Listener(),
		defaultFilesystemImpl, deferredPersistentStorageImpl)
	proxyEnvironment := rootEnv.ProxyEnvironment("o")
	transportEnvironment, err := proxyEnvironment.NarrowScopeToTransport("splithttp")
	if err != nil {
		t.Fatal(err)
	}
	ctx = envctx.ContextWithEnvironment(ctx, transportEnvironment)

	conn, err := Dial(ctx, net.UDPDestination(net.LocalHostIP, listenPort), streamSettings)
	common.Must(err)

	const N = 1024
	b1 := make([]byte, N)
	common.Must2(rand.Read(b1))
	b2 := buf.New()

	common.Must2(conn.Write(b1))

	b2.Clear()
	common.Must2(b2.ReadFullFrom(conn, N))
	if r := cmp.Diff(b2.Bytes(), b1); r != "" {
		t.Error(r)
	}

	common.Must2(conn.Write(b1))

	b2.Clear()
	common.Must2(b2.ReadFullFrom(conn, N))
	if r := cmp.Diff(b2.Bytes(), b1); r != "" {
		t.Error(r)
	}

	conn.Close()
	time.Sleep(100 * time.Millisecond)
	if !serverClosed {
		t.Error("server did not get closed")
	}

	end := time.Now()
	if !end.Before(start.Add(time.Second * 5)) {
		t.Error("end: ", end, " start: ", start)
	}
}

func Test_listenSHAndDial_Unix(t *testing.T) {
	tempDir := t.TempDir()
	tempSocket := tempDir + "/server.sock"

	listen, err := ListenSH(context.Background(), net.DomainAddress(tempSocket), 0, &internet.MemoryStreamConfig{
		ProtocolName: "splithttp",
		ProtocolSettings: &Config{
			Path: "/sh",
		},
	}, func(conn internet.Connection) {
		go func(c internet.Connection) {
			defer c.Close()

			var b [1024]byte
			c.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, err := c.Read(b[:])
			if err != nil {
				return
			}

			common.Must2(c.Write([]byte("Response")))
		}(conn)
	})
	common.Must(err)

	ctx := context.Background()
	defaultNetworkImpl := systemnetworkimpl.NewSystemNetworkDefault()
	defaultFilesystemImpl := filesystemimpl.NewDefaultFileSystemDefaultImpl()
	deferredPersistentStorageImpl := deferredpersistentstorage.NewDeferredPersistentStorage(ctx)
	rootEnv := environment.NewRootEnvImpl(ctx,
		transientstorageimpl.NewScopedTransientStorageImpl(), defaultNetworkImpl.Dialer(), defaultNetworkImpl.Listener(),
		defaultFilesystemImpl, deferredPersistentStorageImpl)
	proxyEnvironment := rootEnv.ProxyEnvironment("o")
	transportEnvironment, err := proxyEnvironment.NarrowScopeToTransport("splithttp")
	if err != nil {
		t.Fatal(err)
	}
	ctx = envctx.ContextWithEnvironment(ctx, transportEnvironment)

	streamSettings := &internet.MemoryStreamConfig{
		ProtocolName: "splithttp",
		ProtocolSettings: &Config{
			Host: "example.com",
			Path: "sh",
		},
	}
	conn, err := Dial(ctx, net.UnixDestination(net.DomainAddress(tempSocket)), streamSettings)

	common.Must(err)
	_, err = conn.Write([]byte("Test connection 1"))
	common.Must(err)

	var b [1024]byte
	fmt.Println("test2")
	n, _ := io.ReadFull(conn, b[:])
	fmt.Println("string is", n)
	if string(b[:n]) != "Response" {
		t.Error("response: ", string(b[:n]))
	}

	common.Must(conn.Close())
	conn, err = Dial(ctx, net.UnixDestination(net.DomainAddress(tempSocket)), streamSettings)

	common.Must(err)
	_, err = conn.Write([]byte("Test connection 2"))
	common.Must(err)
	n, _ = io.ReadFull(conn, b[:])
	common.Must(err)
	if string(b[:n]) != "Response" {
		t.Error("response: ", string(b[:n]))
	}
	common.Must(conn.Close())

	common.Must(listen.Close())
}

func Test_queryString(t *testing.T) {
	listenPort := tcp.PickPort()
	listen, err := ListenSH(context.Background(), net.LocalHostIP, listenPort, &internet.MemoryStreamConfig{
		ProtocolName: "splithttp",
		ProtocolSettings: &Config{
			// this querystring does not have any effect, but sometimes people blindly copy it from websocket config. make sure the outbound doesn't break
			Path: "/sh?ed=2048",
		},
	}, func(conn internet.Connection) {
		go func(c internet.Connection) {
			defer c.Close()

			var b [1024]byte
			c.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, err := c.Read(b[:])
			if err != nil {
				return
			}

			common.Must2(c.Write([]byte("Response")))
		}(conn)
	})
	common.Must(err)

	ctx := context.Background()
	defaultNetworkImpl := systemnetworkimpl.NewSystemNetworkDefault()
	defaultFilesystemImpl := filesystemimpl.NewDefaultFileSystemDefaultImpl()
	deferredPersistentStorageImpl := deferredpersistentstorage.NewDeferredPersistentStorage(ctx)
	rootEnv := environment.NewRootEnvImpl(ctx,
		transientstorageimpl.NewScopedTransientStorageImpl(), defaultNetworkImpl.Dialer(), defaultNetworkImpl.Listener(),
		defaultFilesystemImpl, deferredPersistentStorageImpl)
	proxyEnvironment := rootEnv.ProxyEnvironment("o")
	transportEnvironment, err := proxyEnvironment.NarrowScopeToTransport("splithttp")
	if err != nil {
		t.Fatal(err)
	}
	ctx = envctx.ContextWithEnvironment(ctx, transportEnvironment)

	streamSettings := &internet.MemoryStreamConfig{
		ProtocolName:     "splithttp",
		ProtocolSettings: &Config{Path: "sh?ed=2048"},
	}
	conn, err := Dial(ctx, net.TCPDestination(net.LocalHostIP, listenPort), streamSettings)

	common.Must(err)
	_, err = conn.Write([]byte("Test connection 1"))
	common.Must(err)

	var b [1024]byte
	fmt.Println("test2")
	n, _ := io.ReadFull(conn, b[:])
	fmt.Println("string is", n)
	if string(b[:n]) != "Response" {
		t.Error("response: ", string(b[:n]))
	}

	common.Must(conn.Close())
	common.Must(listen.Close())
}

func Test_maxUpload(t *testing.T) {
	listenPort := tcp.PickPort()
	streamSettings := &internet.MemoryStreamConfig{
		ProtocolName: "splithttp",
		ProtocolSettings: &Config{
			Path: "/sh",
		},
	}

	uploadReceived := make([]byte, 10001)
	listen, err := ListenSH(context.Background(), net.LocalHostIP, listenPort, streamSettings, func(conn internet.Connection) {
		go func(c internet.Connection) {
			defer c.Close()
			c.SetReadDeadline(time.Now().Add(2 * time.Second))
			io.ReadFull(c, uploadReceived)

			common.Must2(c.Write([]byte("Response")))
		}(conn)
	})
	common.Must(err)

	ctx := context.Background()
	defaultNetworkImpl := systemnetworkimpl.NewSystemNetworkDefault()
	defaultFilesystemImpl := filesystemimpl.NewDefaultFileSystemDefaultImpl()
	deferredPersistentStorageImpl := deferredpersistentstorage.NewDeferredPersistentStorage(ctx)
	rootEnv := environment.NewRootEnvImpl(ctx,
		transientstorageimpl.NewScopedTransientStorageImpl(), defaultNetworkImpl.Dialer(), defaultNetworkImpl.Listener(),
		defaultFilesystemImpl, deferredPersistentStorageImpl)
	proxyEnvironment := rootEnv.ProxyEnvironment("o")
	transportEnvironment, err := proxyEnvironment.NarrowScopeToTransport("splithttp")
	if err != nil {
		t.Fatal(err)
	}
	ctx = envctx.ContextWithEnvironment(ctx, transportEnvironment)

	conn, err := Dial(ctx, net.TCPDestination(net.LocalHostIP, listenPort), streamSettings)
	common.Must(err)

	// send a slightly too large upload
	upload := make([]byte, 10001)
	rand.Read(upload)
	_, err = conn.Write(upload)
	common.Must(err)

	var b [buf.Size]byte
	n, _ := io.ReadFull(conn, b[:])
	fmt.Println("string is", n)
	if string(b[:n]) != "Response" {
		t.Error("response: ", string(b[:n]))
	}
	common.Must(conn.Close())

	if !bytes.Equal(upload, uploadReceived) {
		t.Error("incorrect upload", upload, uploadReceived)
	}

	common.Must(listen.Close())
}
