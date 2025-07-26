package outbound

import (
	"context"
	"sync"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/app/proxyman"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/environment"
	"github.com/v2fly/v2ray-core/v5/common/environment/envctx"
	"github.com/v2fly/v2ray-core/v5/common/mux"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/net/cnc"
	"github.com/v2fly/v2ray-core/v5/common/net/packetaddr"
	"github.com/v2fly/v2ray-core/v5/common/serial"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/features/dns"
	"github.com/v2fly/v2ray-core/v5/features/outbound"
	"github.com/v2fly/v2ray-core/v5/features/policy"
	"github.com/v2fly/v2ray-core/v5/features/stats"
	"github.com/v2fly/v2ray-core/v5/proxy"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/security"
	"github.com/v2fly/v2ray-core/v5/transport/pipe"
)

func getStatCounter(v *core.Instance, tag string) (stats.Counter, stats.Counter) {
	var uplinkCounter stats.Counter
	var downlinkCounter stats.Counter

	policy := v.GetFeature(policy.ManagerType()).(policy.Manager)
	if len(tag) > 0 && policy.ForSystem().Stats.OutboundUplink {
		statsManager := v.GetFeature(stats.ManagerType()).(stats.Manager)
		name := "outbound>>>" + tag + ">>>traffic>>>uplink"
		c, _ := stats.GetOrRegisterCounter(statsManager, name)
		if c != nil {
			uplinkCounter = c
		}
	}
	if len(tag) > 0 && policy.ForSystem().Stats.OutboundDownlink {
		statsManager := v.GetFeature(stats.ManagerType()).(stats.Manager)
		name := "outbound>>>" + tag + ">>>traffic>>>downlink"
		c, _ := stats.GetOrRegisterCounter(statsManager, name)
		if c != nil {
			downlinkCounter = c
		}
	}

	return uplinkCounter, downlinkCounter
}

// Handler is an implements of outbound.Handler.
type Handler struct {
	ctx               context.Context
	tag               string
	senderSettings    *proxyman.SenderConfig
	streamSettings    *internet.MemoryStreamConfig
	proxy             proxy.Outbound
	outboundManager   outbound.Manager
	mux               *mux.ClientManager
	uplinkCounter     stats.Counter
	downlinkCounter   stats.Counter
	dns               dns.Client
	fakedns           dns.FakeDNSEngine
	muxPacketEncoding packetaddr.PacketAddrType
	pool              *internet.ConnectionPool
	closed            bool
}

// NewHandler create a new Handler based on the given configuration.
func NewHandler(ctx context.Context, config *core.OutboundHandlerConfig) (outbound.Handler, error) {
	v := core.MustFromContext(ctx)
	uplinkCounter, downlinkCounter := getStatCounter(v, config.Tag)
	h := &Handler{
		ctx:             ctx,
		tag:             config.Tag,
		outboundManager: v.GetFeature(outbound.ManagerType()).(outbound.Manager),
		uplinkCounter:   uplinkCounter,
		downlinkCounter: downlinkCounter,
		pool:            internet.NewConnectionPool(),
	}

	if config.SenderSettings != nil {
		senderSettings, err := serial.GetInstanceOf(config.SenderSettings)
		if err != nil {
			return nil, err
		}
		switch s := senderSettings.(type) {
		case *proxyman.SenderConfig:
			h.senderSettings = s
			mss, err := internet.ToMemoryStreamConfig(s.StreamSettings)
			if err != nil {
				return nil, newError("failed to parse stream settings").Base(err).AtWarning()
			}
			h.streamSettings = mss
		default:
			return nil, newError("settings is not SenderConfig")
		}
	}

	proxyConfig, err := serial.GetInstanceOf(config.ProxySettings)
	if err != nil {
		return nil, err
	}

	rawProxyHandler, err := common.CreateObject(ctx, proxyConfig)
	if err != nil {
		return nil, err
	}

	proxyHandler, ok := rawProxyHandler.(proxy.Outbound)
	if !ok {
		return nil, newError("not an outbound handler")
	}

	if h.senderSettings != nil && h.senderSettings.MultiplexSettings != nil {
		config := h.senderSettings.MultiplexSettings
		if config.Concurrency < 1 || config.Concurrency > 1024 {
			return nil, newError("invalid mux concurrency: ", config.Concurrency).AtWarning()
		}
		h.muxPacketEncoding = h.senderSettings.MultiplexSettings.PacketEncoding
		h.mux = &mux.ClientManager{
			Enabled: h.senderSettings.MultiplexSettings.Enabled,
			Picker: &mux.IncrementalWorkerPicker{
				Factory: mux.NewDialingWorkerFactory(
					ctx,
					proxyHandler,
					h,
					mux.ClientStrategy{
						MaxConcurrency: config.Concurrency,
						MaxConnection:  128,
					},
				),
			},
		}
	}

	if h.senderSettings != nil && h.senderSettings.DomainStrategy != proxyman.SenderConfig_AS_IS {
		err := core.RequireFeatures(ctx, func(d dns.Client) error {
			h.dns = d
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	_ = core.RequireFeatures(ctx, func(fakedns dns.FakeDNSEngine) error {
		h.fakedns = fakedns
		return nil
	})

	h.proxy = proxyHandler
	return h, nil
}

// Tag implements outbound.Handler.
func (h *Handler) Tag() string {
	return h.tag
}

// Dispatch implements proxy.Outbound.Dispatch.
func (h *Handler) Dispatch(ctx context.Context, link *transport.Link) {
	outbound := session.OutboundFromContext(ctx)
	if outbound == nil {
		outbound = new(session.Outbound)
		ctx = session.ContextWithOutbound(ctx, outbound)
	}
	if outbound.Target.Network != net.Network_UDP && h.senderSettings != nil && h.senderSettings.DomainStrategy != proxyman.SenderConfig_AS_IS {
		if outbound.Target.Address != nil && outbound.Target.Address.Family().IsDomain() {
			if addr := h.resolveIP(ctx, outbound.Target.Address.Domain(), h.Address()); addr != nil {
				outbound.Target.Address = addr
			}
		}
	}
	if outbound.Target.Network == net.Network_UDP {
		if h.senderSettings != nil && h.senderSettings.DomainStrategy != proxyman.SenderConfig_AS_IS {
			if outbound.Target.Address != nil && outbound.Target.Address.Family().IsDomain() {
				if addr := h.resolveIP(ctx, outbound.Target.Address.Domain(), h.Address()); addr != nil {
					outbound.Target.Address = addr
				}
			}
		}
		reader := &EndpointOverrideReader{
			Reader:       link.Reader,
			Dest:         outbound.Target.Address,
			OriginalDest: outbound.OriginalTarget.Address,
		}
		writer := &EndpointOverrideWriter{
			Writer:       link.Writer,
			Dest:         outbound.Target.Address,
			OriginalDest: outbound.OriginalTarget.Address,
		}
		if h.fakedns != nil {
			reader.fakedns = h.fakedns
			writer.fakedns = h.fakedns
			usedFakeIPs := new(sync.Map)
			reader.usedFakeIPs = usedFakeIPs
			writer.usedFakeIPs = usedFakeIPs
		}
		if h.senderSettings != nil && h.senderSettings.DomainStrategy != proxyman.SenderConfig_AS_IS {
			writer.resolveIP = func(domain string) net.Address {
				return h.resolveIP(h.ctx, domain, h.Address())
			}
			ipToDomain := new(sync.Map)
			reader.ipToDomain = ipToDomain
			writer.ipToDomain = ipToDomain
		}
		link.Reader = reader
		link.Writer = writer
	}
	if h.mux != nil && (h.mux.Enabled || session.MuxPreferedFromContext(ctx)) {
		if outbound.Target.Network == net.Network_UDP {
			switch h.muxPacketEncoding {
			case packetaddr.PacketAddrType_None:
				link.Reader = &buf.EndpointErasureReader{Reader: link.Reader}
				link.Writer = &buf.EndpointErasureWriter{Writer: link.Writer}
			case packetaddr.PacketAddrType_XUDP:
				break
			case packetaddr.PacketAddrType_Packet:
				link.Reader = packetaddr.NewReversePacketReader(link.Reader, outbound.Target)
				link.Writer = packetaddr.NewReversePacketWriter(link.Writer)
				outbound.Target = net.Destination{
					Network: net.Network_UDP,
					Address: net.DomainAddress(packetaddr.SeqPacketMagicAddress),
					Port:    0,
				}
			}
		}
		if err := h.mux.Dispatch(ctx, link); err != nil {
			err := newError("failed to process mux outbound traffic").Base(err)
			session.SubmitOutboundErrorToOriginator(ctx, err)
			err.WriteToLog(session.ExportIDToError(ctx))
			common.Interrupt(link.Writer)
		}
	} else {
		if err := h.proxy.Process(ctx, link, h); err != nil {
			// Ensure outbound ray is properly closed.
			err := newError("failed to process outbound traffic").Base(err)
			session.SubmitOutboundErrorToOriginator(ctx, err)
			err.WriteToLog(session.ExportIDToError(ctx))
			common.Interrupt(link.Writer)
		} else {
			common.Must(common.Close(link.Writer))
		}
		common.Interrupt(link.Reader)
	}
}

// Address implements internet.Dialer.
func (h *Handler) Address() net.Address {
	if h.senderSettings == nil || h.senderSettings.Via == nil {
		return nil
	}
	return h.senderSettings.Via.AsAddress()
}

// Dial implements internet.Dialer.
func (h *Handler) Dial(ctx context.Context, dest net.Destination) (internet.Connection, error) {
	if h.closed {
		return nil, newError("handler closed")
	}
	if h.senderSettings != nil {
		if h.senderSettings.ProxySettings.HasTag() && !h.senderSettings.ProxySettings.TransportLayerProxy {
			tag := h.senderSettings.ProxySettings.Tag
			handler := h.outboundManager.GetHandler(tag)
			if handler != nil {
				newError("proxying to ", tag, " for dest ", dest).AtDebug().WriteToLog(session.ExportIDToError(ctx))
				ctx = session.ContextWithOutbound(ctx, &session.Outbound{
					Target: dest,
				})

				opts := pipe.OptionsFromContext(ctx)
				uplinkReader, uplinkWriter := pipe.New(opts...)
				downlinkReader, downlinkWriter := pipe.New(opts...)

				go handler.Dispatch(ctx, &transport.Link{Reader: uplinkReader, Writer: downlinkWriter})
				conn := cnc.NewConnection(cnc.ConnectionInputMulti(uplinkWriter), cnc.ConnectionOutputMulti(downlinkReader))

				securityEngine, err := security.CreateSecurityEngineFromSettings(ctx, h.streamSettings)
				if err != nil {
					return nil, newError("unable to create security engine").Base(err)
				}

				if securityEngine != nil {
					conn, err = securityEngine.Client(conn, security.OptionWithDestination{Dest: dest})
					if err != nil {
						return nil, newError("unable to create security protocol client from security engine").Base(err)
					}
				}

				return internet.NewTrackedConn(h.getStatCouterConnection(conn), h.pool), nil
			}

			newError("failed to get outbound handler with tag: ", tag).AtWarning().WriteToLog(session.ExportIDToError(ctx))
		}

		if h.senderSettings.Via != nil {
			outbound := session.OutboundFromContext(ctx)
			if outbound == nil {
				outbound = new(session.Outbound)
				ctx = session.ContextWithOutbound(ctx, outbound)
			}
			outbound.Gateway = h.senderSettings.Via.AsAddress()
		}

		if h.senderSettings.DialDomainStrategy != proxyman.SenderConfig_AS_IS {
			outbound := session.OutboundFromContext(ctx)
			if outbound == nil {
				outbound = new(session.Outbound)
				ctx = session.ContextWithOutbound(ctx, outbound)
			}
			outbound.Resolver = func(ctx context.Context, domain string) net.Address {
				return h.resolveIP(ctx, domain, h.Address())
			}
		}
	}

	enablePacketAddrCapture := true
	if h.senderSettings != nil && h.senderSettings.ProxySettings != nil && h.senderSettings.ProxySettings.HasTag() && h.senderSettings.ProxySettings.TransportLayerProxy {
		tag := h.senderSettings.ProxySettings.Tag
		newError("transport layer proxying to ", tag, " for dest ", dest).AtDebug().WriteToLog(session.ExportIDToError(ctx))
		ctx = session.SetTransportLayerProxyTagToContext(ctx, tag)
		enablePacketAddrCapture = false
	}

	if h.streamSettings != nil && h.streamSettings.SocketSettings != nil && len(h.streamSettings.SocketSettings.DialerProxy) > 0 {
		tag := h.streamSettings.SocketSettings.DialerProxy
		newError("transport layer proxying to ", tag, " for dest ", dest).AtDebug().WriteToLog(session.ExportIDToError(ctx))
		ctx = session.SetTransportLayerProxyTagToContext(ctx, tag)
		enablePacketAddrCapture = false
	}

	if isStream, err := packetaddr.GetDestinationSubsetOf(dest); err == nil && enablePacketAddrCapture {
		packetConn, err := internet.ListenSystemPacket(ctx, &net.UDPAddr{IP: net.AnyIP.IP(), Port: 0}, h.streamSettings.SocketSettings)
		if err != nil {
			return nil, newError("unable to listen socket").Base(err)
		}
		conn := packetaddr.ToPacketAddrConnWrapper(packetConn, isStream)
		return internet.NewTrackedConn(h.getStatCouterConnection(conn), h.pool), nil
	}

	proxyEnvironment := envctx.EnvironmentFromContext(h.ctx).(environment.ProxyEnvironment)
	transportEnvironment, err := proxyEnvironment.NarrowScopeToTransport("transport")
	if err != nil {
		return nil, newError("unable to narrow environment to transport").Base(err)
	}
	ctx = envctx.ContextWithEnvironment(ctx, transportEnvironment)
	conn, err := internet.Dial(ctx, dest, h.streamSettings)
	if err != nil {
		return nil, err
	}
	return internet.NewTrackedConn(h.getStatCouterConnection(conn), h.pool), err
}

func (h *Handler) resolveIP(ctx context.Context, domain string, localAddr net.Address) net.Address {
	strategy := h.senderSettings.DomainStrategy
	ips, err := dns.LookupIPWithOption(h.dns, domain, dns.IPOption{
		IPv4Enable: strategy != proxyman.SenderConfig_USE_IP6 || (localAddr != nil && localAddr.Family().IsIPv4()),
		IPv6Enable: strategy != proxyman.SenderConfig_USE_IP4 || (localAddr != nil && localAddr.Family().IsIPv6()),
		FakeEnable: false,
	})
	if err != nil {
		newError("failed to get IP address for domain ", domain).Base(err).WriteToLog(session.ExportIDToError(ctx))
	}
	if len(ips) == 0 {
		return nil
	}
	if strategy == proxyman.SenderConfig_PREFER_IP4 || strategy == proxyman.SenderConfig_PREFER_IP6 {
		var addr net.Address
		for _, ip := range ips {
			addr = net.IPAddress(ip)
			if addr.Family().IsIPv4() == (strategy == proxyman.SenderConfig_PREFER_IP4) {
				return addr
			}
		}
	}
	return net.IPAddress(ips[0])
}

func (h *Handler) getStatCouterConnection(conn internet.Connection) internet.Connection {
	if h.uplinkCounter != nil || h.downlinkCounter != nil {
		return &internet.StatCouterConnection{
			Connection:   conn,
			ReadCounter:  h.downlinkCounter,
			WriteCounter: h.uplinkCounter,
		}
	}
	return conn
}

// GetOutbound implements proxy.GetOutbound.
func (h *Handler) GetOutbound() proxy.Outbound {
	return h.proxy
}

// Start implements common.Runnable.
func (h *Handler) Start() error {
	return nil
}

// Close implements common.Closable.
func (h *Handler) Close() error {
	h.closed = true
	h.pool.ResetConnections()

	common.Close(h.mux)

	if closableProxy, ok := h.proxy.(common.Closable); ok {
		if err := closableProxy.Close(); err != nil {
			return newError("unable to close proxy").Base(err)
		}
	}
	return nil
}
