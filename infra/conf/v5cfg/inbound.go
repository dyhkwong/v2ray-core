package v5cfg

import (
	"context"

	"github.com/golang/protobuf/proto"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/app/proxyman"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/serial"
	"github.com/v2fly/v2ray-core/v5/proxy/dokodemo"
	"github.com/v2fly/v2ray-core/v5/proxy/socks"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

func (c InboundConfig) BuildV5(ctx context.Context) (proto.Message, error) {
	receiverSettings := &proxyman.ReceiverConfig{}

	if c.ListenOn == nil {
		// Listen on anyip, must set PortRange
		if c.PortRange == nil {
			return nil, newError("Listen on AnyIP but no Port(s) set in InboundDetour.")
		}
		receiverSettings.PortRange = c.PortRange.Build()
	} else {
		// Listen on specific IP or Unix Domain Socket
		receiverSettings.Listen = c.ListenOn.Build()
		listenDS := c.ListenOn.Family().IsDomain() && (c.ListenOn.Domain()[0] == '/' || c.ListenOn.Domain()[0] == '@')
		listenIP := c.ListenOn.Family().IsIP() || (c.ListenOn.Family().IsDomain() && c.ListenOn.Domain() == "localhost")
		switch {
		case listenIP:
			// Listen on specific IP, must set PortRange
			if c.PortRange == nil {
				return nil, newError("Listen on specific ip without port in InboundDetour.")
			}
			// Listen on IP:Port
			receiverSettings.PortRange = c.PortRange.Build()
		case listenDS:
			if c.PortRange != nil {
				// Listen on Unix Domain Socket, PortRange should be nil
				receiverSettings.PortRange = nil
			}
		default:
			return nil, newError("unable to listen on domain address: ", c.ListenOn.Domain())
		}
	}

	if c.StreamSetting != nil {
		ss, err := c.StreamSetting.BuildV5(ctx)
		if err != nil {
			return nil, err
		}
		receiverSettings.StreamSettings = ss.(*internet.StreamConfig)
	}

	if c.SniffingConfig != nil {
		s, err := c.SniffingConfig.Build()
		if err != nil {
			return nil, newError("failed to build sniffing config").Base(err)
		}
		receiverSettings.SniffingSettings = s
	}

	if c.Settings == nil {
		c.Settings = []byte("{}")
	}

	inboundConfigPack, err := loadHeterogeneousConfigFromRawJSON("inbound", c.Protocol, c.Settings)
	if err != nil {
		return nil, newError("unable to load inbound protocol config").Base(err)
	}

	if content, ok := inboundConfigPack.(*dokodemo.SimplifiedConfig); ok && content.FollowRedirect {
		receiverSettings.ReceiveOriginalDestination = true
		if receiverSettings.StreamSettings == nil {
			receiverSettings.StreamSettings = &internet.StreamConfig{}
		}
		if receiverSettings.StreamSettings.SocketSettings == nil {
			receiverSettings.StreamSettings.SocketSettings = &internet.SocketConfig{}
		}
		receiverSettings.StreamSettings.SocketSettings.ReceiveOriginalDestAddress = true
		if receiverSettings.StreamSettings.SocketSettings.Tproxy == internet.SocketConfig_Off {
			receiverSettings.StreamSettings.SocketSettings.Tproxy = internet.SocketConfig_Redirect
		}
	}
	if content, ok := inboundConfigPack.(*dokodemo.Config); ok && content.FollowRedirect {
		receiverSettings.ReceiveOriginalDestination = true
		if receiverSettings.StreamSettings == nil {
			receiverSettings.StreamSettings = &internet.StreamConfig{}
		}
		if receiverSettings.StreamSettings.SocketSettings == nil {
			receiverSettings.StreamSettings.SocketSettings = &internet.SocketConfig{}
		}
		receiverSettings.StreamSettings.SocketSettings.ReceiveOriginalDestAddress = true
		if receiverSettings.StreamSettings.SocketSettings.Tproxy == internet.SocketConfig_Off {
			receiverSettings.StreamSettings.SocketSettings.Tproxy = internet.SocketConfig_Redirect
		}
	}
	if content, ok := inboundConfigPack.(*socks.ServerConfig); ok && content.UdpEnabled &&
		(receiverSettings.Listen.AsAddress() == net.AnyIP || receiverSettings.Listen.AsAddress() == net.AnyIPv6) {
		receiverSettings.ReceiveOriginalDestination = true
		if receiverSettings.StreamSettings == nil {
			receiverSettings.StreamSettings = &internet.StreamConfig{}
		}
		if receiverSettings.StreamSettings.SocketSettings == nil {
			receiverSettings.StreamSettings.SocketSettings = &internet.SocketConfig{}
		}
		receiverSettings.StreamSettings.SocketSettings.ReceiveOriginalDestAddress = true
	}

	return &core.InboundHandlerConfig{
		Tag:              c.Tag,
		ReceiverSettings: serial.ToTypedMessage(receiverSettings),
		ProxySettings:    serial.ToTypedMessage(inboundConfigPack),
	}, nil
}
