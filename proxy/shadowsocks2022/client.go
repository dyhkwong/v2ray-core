package shadowsocks2022

import (
	"context"
	gonet "net"
	"strconv"
	"sync"
	"time"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/environment"
	"github.com/v2fly/v2ray-core/v5/common/environment/envctx"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/net/packetaddr"
	"github.com/v2fly/v2ray-core/v5/common/retry"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal"
	"github.com/v2fly/v2ray-core/v5/common/task"
	"github.com/v2fly/v2ray-core/v5/proxy/sip003"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/udp"
)

type Client struct {
	config *ClientConfig
	ctx    context.Context

	plugin         sip003.Plugin
	pluginOverride net.Destination
}

func (c *Client) Close() error {
	if c.plugin != nil {
		return c.plugin.Close()
	}
	return nil
}

const UDPConnectionState = "UDPConnectionState"

type ClientUDPConnState struct {
	session  *ClientUDPSession
	initOnce *sync.Once
}

func (c *ClientUDPConnState) GetOrCreateSession(create func() (*ClientUDPSession, error)) (*ClientUDPSession, error) {
	var errOuter error
	c.initOnce.Do(func() {
		sessionState, err := create()
		if err != nil {
			errOuter = newError("failed to create UDP session").Base(err)
			return
		}
		c.session = sessionState
	})
	if errOuter != nil {
		return nil, newError("failed to initialize UDP State").Base(errOuter)
	}
	return c.session, nil
}

func NewClientUDPConnState() (*ClientUDPConnState, error) {
	return &ClientUDPConnState{initOnce: &sync.Once{}}, nil
}

func (c *Client) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified")
	}
	destination := outbound.Target
	network := destination.Network

	keyDerivation := newBLAKE3KeyDerivation()
	var method Method
	switch c.config.Method {
	case "2022-blake3-aes-128-gcm":
		method = newAES128GCMMethod()
	case "2022-blake3-aes-256-gcm":
		method = newAES256GCMMethod()
	case "2022-blake3-chacha20-poly1305":
		method = newChacha20Poly1305Method()
	default:
		return newError("unknown method: ", c.config.Method)
	}

	effectivePsk := c.config.Psk

	ctx, cancel := context.WithCancel(ctx)
	timer := signal.CancelAfterInactivity(ctx, cancel, time.Minute)

	if packetConn, err := packetaddr.ToPacketAddrConn(link, destination); err == nil {
		udpSession, err := c.getUDPSession(c.ctx, network, dialer, method, keyDerivation)
		if err != nil {
			return newError("failed to get UDP udpSession").Base(err)
		}
		requestDone := func() error {
			return udp.CopyPacketConn(udpSession, packetConn, udp.UpdateActivity(timer))
		}
		responseDone := func() error {
			return udp.CopyPacketConn(packetConn, udpSession, udp.UpdateActivity(timer))
		}
		responseDoneAndCloseWriter := task.OnSuccess(responseDone, task.Close(link.Writer))
		if err := task.Run(ctx, requestDone, responseDoneAndCloseWriter); err != nil {
			return newError("connection ends").Base(err)
		}
		return nil
	}

	if network == net.Network_TCP {
		var conn internet.Connection
		err := retry.ExponentialBackoff(5, 100).On(func() error {
			var dest net.Destination
			if network == net.Network_TCP && c.plugin != nil {
				dest = c.pluginOverride
			} else {
				dest = net.TCPDestination(c.config.Address.AsAddress(), net.Port(c.config.Port))
			}
			dest.Network = network
			rawConn, err := dialer.Dial(ctx, dest)
			if err != nil {
				return err
			}
			conn = rawConn

			return nil
		})
		if err != nil {
			return newError("failed to find an available destination").AtWarning().Base(err)
		}
		newError("tunneling request to ", destination, " via ", network, ":", net.TCPDestination(c.config.Address.AsAddress(), net.Port(c.config.Port)).NetAddr()).WriteToLog(session.ExportIDToError(ctx))
		defer conn.Close()

		request := &TCPRequest{
			keyDerivation: keyDerivation,
			method:        method,
		}
		TCPRequestBuffer := buf.New()
		defer TCPRequestBuffer.Release()
		err = request.EncodeTCPRequestHeader(effectivePsk, c.config.Ipsk, destination.Address,
			int(destination.Port), nil, TCPRequestBuffer)
		if err != nil {
			return newError("failed to encode TCP request header").Base(err)
		}
		_, err = conn.Write(TCPRequestBuffer.Bytes())
		if err != nil {
			return newError("failed to write TCP request header").Base(err)
		}
		requestDone := func() error {
			encodedWriter := request.CreateClientC2SWriter(conn)
			return buf.Copy(link.Reader, encodedWriter, buf.UpdateActivity(timer))
		}
		responseDone := func() error {
			err = request.DecodeTCPResponseHeader(effectivePsk, conn)
			if err != nil {
				return newError("failed to decode TCP response header").Base(err)
			}
			if err = request.CheckC2SConnectionConstraint(); err != nil {
				return newError("C2S connection constraint violation").Base(err)
			}
			initialPayload := buf.NewWithSize(65535)
			encodedReader, err := request.CreateClientS2CReader(conn, initialPayload)
			if err != nil {
				initialPayload.Release()
				return newError("failed to create client S2C reader").Base(err)
			}
			err = link.Writer.WriteMultiBuffer(buf.MultiBuffer{initialPayload})
			if err != nil {
				return newError("failed to write initial payload").Base(err)
			}
			return buf.Copy(encodedReader, link.Writer, buf.UpdateActivity(timer))
		}
		responseDoneAndCloseWriter := task.OnSuccess(responseDone, task.Close(link.Writer))
		if err := task.Run(ctx, requestDone, responseDoneAndCloseWriter); err != nil {
			return newError("connection ends").Base(err)
		}
		return nil
	} else {
		udpSession, err := c.getUDPSession(c.ctx, network, dialer, method, keyDerivation)
		if err != nil {
			return newError("failed to get UDP udpSession").Base(err)
		}
		var addr net.Addr
		if !destination.Address.Family().IsDomain() {
			addr = &gonet.UDPAddr{IP: destination.Address.IP(), Port: int(destination.Port)}
		} else {
			addr = &udp.MonoDestUDPAddr{Address: destination.Address, Port: destination.Port}
		}
		monoDestUDPConn := udp.NewMonoDestUDPConn(udpSession, addr)
		requestDone := func() error {
			return buf.Copy(link.Reader, monoDestUDPConn, buf.UpdateActivity(timer))
		}
		responseDone := func() error {
			return buf.Copy(monoDestUDPConn, link.Writer, buf.UpdateActivity(timer))
		}
		responseDoneAndCloseWriter := task.OnSuccess(responseDone, task.Close(link.Writer))
		if err := task.Run(ctx, requestDone, responseDoneAndCloseWriter); err != nil {
			return newError("connection ends").Base(err)
		}
		return nil
	}
}

func (c *Client) getUDPSession(ctx context.Context, network net.Network, dialer internet.Dialer, method Method, keyDerivation *BLAKE3KeyDerivation) (internet.AbstractPacketConn, error) {
	storage := envctx.EnvironmentFromContext(ctx).(environment.ProxyEnvironment).TransientStorage()
	clientUDPStateIfce, err := storage.Get(ctx, UDPConnectionState)
	if err != nil {
		return nil, newError("failed to get UDP connection state").Base(err)
	}
	clientUDPState, ok := clientUDPStateIfce.(*ClientUDPConnState)
	if !ok {
		return nil, newError("failed to cast UDP connection state")
	}

	sessionState, err := clientUDPState.GetOrCreateSession(func() (*ClientUDPSession, error) {
		var conn internet.Connection
		err := retry.ExponentialBackoff(5, 100).On(func() error {
			dest := net.TCPDestination(c.config.Address.AsAddress(), net.Port(c.config.Port))
			dest.Network = network
			rawConn, err := dialer.Dial(ctx, dest)
			if err != nil {
				return err
			}
			conn = rawConn

			return nil
		})
		if err != nil {
			return nil, newError("failed to find an available destination").AtWarning().Base(err)
		}
		newError("creating udp session to ", network, ":", net.UDPDestination(c.config.Address.AsAddress(), net.Port(c.config.Port)).NetAddr()).WriteToLog(session.ExportIDToError(ctx))
		packetProcessor, err := method.GetUDPClientProcessor(c.config.Ipsk, c.config.Psk, keyDerivation)
		if err != nil {
			return nil, newError("failed to create UDP client packet processor").Base(err)
		}
		return NewClientUDPSession(ctx, conn, packetProcessor), nil
	})
	if err != nil {
		return nil, newError("failed to create UDP session").Base(err)
	}
	sessionConn, err := sessionState.NewSessionConn()
	if err != nil {
		return nil, newError("failed to create UDP session connection").Base(err)
	}
	return sessionConn, nil
}

func NewClient(ctx context.Context, config *ClientConfig) (*Client, error) {
	storage := envctx.EnvironmentFromContext(ctx).(environment.ProxyEnvironment).TransientStorage()

	udpState, err := NewClientUDPConnState()
	if err != nil {
		return nil, newError("failed to create UDP connection state").Base(err)
	}
	storage.Put(ctx, UDPConnectionState, udpState)

	c := &Client{
		config: config,
		ctx:    ctx,
	}

	if config.Plugin != "" {
		var plugin sip003.Plugin
		if pc := sip003.Plugins[config.Plugin]; pc != nil {
			plugin = pc()
		} else if sip003.PluginLoader == nil {
			return nil, newError("plugin loader not registered")
		} else {
			plugin = sip003.PluginLoader(config.Plugin)
		}
		port, err := net.GetFreePort()
		if err != nil {
			return nil, newError("failed to get free port for sip003 plugin").Base(err)
		}
		c.pluginOverride = net.Destination{
			Network: net.Network_TCP,
			Address: net.LocalHostIP,
			Port:    net.Port(port),
		}
		if err := plugin.Init(net.LocalHostIP.String(), strconv.Itoa(port), config.Address.AsAddress().String(), net.Port(config.Port).String(), config.PluginOpts, config.PluginArgs, config.PluginWorkingDir); err != nil {
			return nil, newError("failed to start plugin").Base(err)
		}
		c.plugin = plugin
	}

	return c, nil
}

func init() {
	common.Must(common.RegisterConfig((*ClientConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		clientConfig, ok := config.(*ClientConfig)
		if !ok {
			return nil, newError("not a ClientConfig")
		}
		return NewClient(ctx, clientConfig)
	}))
}
