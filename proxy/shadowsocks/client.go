package shadowsocks

import (
	"context"
	"crypto/rand"
	"strconv"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/net/packetaddr"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
	"github.com/v2fly/v2ray-core/v5/common/retry"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal"
	"github.com/v2fly/v2ray-core/v5/common/task"
	"github.com/v2fly/v2ray-core/v5/features/policy"
	"github.com/v2fly/v2ray-core/v5/proxy"
	"github.com/v2fly/v2ray-core/v5/proxy/sip003"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/udp"
)

// Client is a inbound handler for Shadowsocks protocol
type Client struct {
	serverPicker  protocol.ServerPicker
	policyManager policy.Manager

	plugin         sip003.Plugin
	pluginOverride net.Destination

	streamPlugin   sip003.StreamPlugin
	protocolPlugin sip003.ProtocolPlugin
}

func (c *Client) Close() error {
	if c.plugin != nil {
		return c.plugin.Close()
	}
	return nil
}

// NewClient create a new Shadowsocks client.
func NewClient(ctx context.Context, config *ClientConfig) (*Client, error) {
	serverList := protocol.NewServerList()
	for _, rec := range config.Server {
		s, err := protocol.NewServerSpecFromPB(rec)
		if err != nil {
			return nil, newError("failed to parse server spec").Base(err)
		}
		serverList.AddServer(s)
	}
	if serverList.Size() == 0 {
		return nil, newError("0 server")
	}

	v := core.MustFromContext(ctx)
	client := &Client{
		serverPicker:  protocol.NewRoundRobinServerPicker(serverList),
		policyManager: v.GetFeature(policy.ManagerType()).(policy.Manager),
	}

	if config.Plugin != "" {
		s := client.serverPicker.PickServer()
		var plugin sip003.Plugin
		if pc := sip003.Plugins[config.Plugin]; pc != nil {
			plugin = pc()
		} else if sip003.PluginLoader == nil {
			return nil, newError("plugin loader not registered")
		} else {
			plugin = sip003.PluginLoader(config.Plugin)
		}

		if streamPlugin, ok := plugin.(sip003.StreamPlugin); ok {
			client.streamPlugin = streamPlugin
			var err error
			if protocolPlugin, ok := plugin.(sip003.ProtocolPlugin); ok {
				client.protocolPlugin = protocolPlugin
				account := s.PickUser().Account.(*MemoryAccount)
				err = protocolPlugin.InitProtocolPlugin(s.Destination().Address.String(), s.Destination().Port.String(), config.PluginArgs, account.Key, int(account.Cipher.IVSize()))
			} else {
				err = streamPlugin.InitStreamPlugin(s.Destination().Port.String(), config.PluginOpts)
			}
			if err != nil {
				return nil, newError("failed to start plugin").Base(err)
			}
			return client, nil
		}

		port, err := net.GetFreePort()
		if err != nil {
			return nil, newError("failed to get free port for sip003 plugin").Base(err)
		}
		client.pluginOverride = net.Destination{
			Network: net.Network_TCP,
			Address: net.LocalHostIP,
			Port:    net.Port(port),
		}
		if err := plugin.Init(net.LocalHostIP.String(), strconv.Itoa(port), s.Destination().Address.String(), s.Destination().Port.String(), config.PluginOpts, config.PluginArgs); err != nil {
			return nil, newError("failed to start plugin").Base(err)
		}
		client.plugin = plugin
	}

	return client, nil
}

// Process implements OutboundHandler.Process().
func (c *Client) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified")
	}
	destination := outbound.Target
	network := destination.Network

	var server *protocol.ServerSpec
	var conn internet.Connection
	var user *protocol.MemoryUser

	err := retry.ExponentialBackoff(5, 100).On(func() error {
		server = c.serverPicker.PickServer()
		user = server.PickUser()
		_, ok := user.Account.(*MemoryAccount)
		if !ok {
			return newError("user account is not valid")
		}

		var dest net.Destination
		if network == net.Network_TCP && c.plugin != nil {
			dest = c.pluginOverride
		} else {
			dest = server.Destination()
			dest.Network = network
		}

		rawConn, err := dialer.Dial(ctx, dest)
		if err != nil {
			return err
		}

		if network == net.Network_TCP && c.streamPlugin != nil {
			conn = c.streamPlugin.StreamConn(rawConn)
		} else {
			conn = rawConn
		}

		return nil
	})
	if err != nil {
		return newError("failed to find an available destination").AtWarning().Base(err)
	}
	newError("tunneling request to ", destination, " via ", network, ":", server.Destination().NetAddr()).WriteToLog(session.ExportIDToError(ctx))

	defer conn.Close()

	request := &protocol.RequestHeader{
		Version: Version,
		Address: destination.Address,
		Port:    destination.Port,
	}
	if destination.Network == net.Network_TCP {
		request.Command = protocol.RequestCommandTCP
	} else {
		request.Command = protocol.RequestCommandUDP
	}

	request.User = user

	sessionPolicy := c.policyManager.ForLevel(user.Level)
	ctx, cancel := context.WithCancel(ctx)
	timer := signal.CancelAfterInactivity(ctx, cancel, sessionPolicy.Timeouts.ConnectionIdle)

	var protocolConn *sip003.ProtocolConn
	var iv []byte
	account := user.Account.(*MemoryAccount)
	if account.Cipher.IVSize() > 0 {
		iv = make([]byte, account.Cipher.IVSize())
		common.Must2(rand.Read(iv))
		if account.ReducedIVEntropy {
			remapToPrintable(iv[:6])
		}
		if ivError := account.CheckIV(iv); ivError != nil {
			return newError("failed to mark outgoing iv").Base(ivError)
		}
	}

	if c.protocolPlugin != nil {
		protocolConn = &sip003.ProtocolConn{}
		c.protocolPlugin.ProtocolConn(protocolConn, iv)
	}

	if packetConn, err := packetaddr.ToPacketAddrConn(link, destination); err == nil {
		requestDone := func() error {
			protocolWriter := &UDPWriter{
				Writer:  conn,
				Request: request,
				Plugin:  c.protocolPlugin,
			}
			return udp.CopyPacketConn(protocolWriter, packetConn, udp.UpdateActivity(timer))
		}
		responseDone := func() error {
			protocolReader := &UDPReader{
				Reader: conn,
				User:   user,
				Plugin: c.protocolPlugin,
			}
			return udp.CopyPacketConn(packetConn, protocolReader, udp.UpdateActivity(timer))
		}
		responseDoneAndCloseWriter := task.OnSuccess(responseDone, task.Close(link.Writer))
		if err := task.Run(ctx, requestDone, responseDoneAndCloseWriter); err != nil {
			return newError("connection ends").Base(err)
		}
		return nil
	}

	if request.Command == protocol.RequestCommandTCP {
		requestDone := func() error {
			defer timer.SetTimeout(sessionPolicy.Timeouts.DownlinkOnly)
			bufferedWriter := buf.NewBufferedWriter(buf.NewWriter(conn))
			bodyWriter, err := WriteTCPRequest(request, bufferedWriter, iv, protocolConn)
			if err != nil {
				return newError("failed to write request").Base(err)
			}

			if err = buf.CopyOnceTimeout(link.Reader, bodyWriter, proxy.FirstPayloadTimeout); err != nil && err != buf.ErrNotTimeoutReader && err != buf.ErrReadTimeout {
				return newError("failed to write A request payload").Base(err).AtWarning()
			}

			if err := bufferedWriter.SetBuffered(false); err != nil {
				return err
			}

			return buf.Copy(link.Reader, bodyWriter, buf.UpdateActivity(timer))
		}

		responseDone := func() error {
			defer timer.SetTimeout(sessionPolicy.Timeouts.UplinkOnly)

			responseReader, err := ReadTCPResponse(user, conn, protocolConn)
			if err != nil {
				return err
			}

			return buf.Copy(responseReader, link.Writer, buf.UpdateActivity(timer))
		}

		responseDoneAndCloseWriter := task.OnSuccess(responseDone, task.Close(link.Writer))
		if err := task.Run(ctx, requestDone, responseDoneAndCloseWriter); err != nil {
			return newError("connection ends").Base(err)
		}

		return nil
	}

	if request.Command == protocol.RequestCommandUDP {
		writer := &UDPWriter{
			Writer:  conn,
			Request: request,
			Plugin:  c.protocolPlugin,
		}

		requestDone := func() error {
			defer timer.SetTimeout(sessionPolicy.Timeouts.DownlinkOnly)

			if err := buf.Copy(link.Reader, writer, buf.UpdateActivity(timer)); err != nil {
				return newError("failed to transport all UDP request").Base(err)
			}
			return nil
		}

		responseDone := func() error {
			defer timer.SetTimeout(sessionPolicy.Timeouts.UplinkOnly)

			reader := &UDPReader{
				Reader: conn,
				User:   user,
				Plugin: c.protocolPlugin,
			}

			if err := buf.Copy(reader, link.Writer, buf.UpdateActivity(timer)); err != nil {
				return newError("failed to transport all UDP response").Base(err)
			}
			return nil
		}

		responseDoneAndCloseWriter := task.OnSuccess(responseDone, task.Close(link.Writer))
		if err := task.Run(ctx, requestDone, responseDoneAndCloseWriter); err != nil {
			return newError("connection ends").Base(err)
		}

		return nil
	}

	return nil
}

func init() {
	common.Must(common.RegisterConfig((*ClientConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewClient(ctx, config.(*ClientConfig))
	}))
}
