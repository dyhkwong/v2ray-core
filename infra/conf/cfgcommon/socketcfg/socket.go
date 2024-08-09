package socketcfg

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

type SocketConfig struct {
	Mark                 uint32    `json:"mark"`
	TFO                  *bool     `json:"tcpFastOpen"`
	TProxy               string    `json:"tproxy"`
	AcceptProxyProtocol  bool      `json:"acceptProxyProtocol"`
	TCPKeepAliveInterval int32     `json:"tcpKeepAliveInterval"`
	TCPKeepAliveIdle     int32     `json:"tcpKeepAliveIdle"`
	TFOQueueLength       uint32    `json:"tcpFastOpenQueueLength"`
	BindToDevice         string    `json:"bindToDevice"`
	RxBufSize            uint64    `json:"rxBufSize"`
	TxBufSize            uint64    `json:"txBufSize"`
	ForceBufSize         bool      `json:"forceBufSize"`
	MPTCP                *bool     `json:"mptcp"`
	DialerProxy          string    `json:"dialerProxy"`
	Fragment             *Fragment `json:"fragment"`
	Noises               []*Noise  `json:"noises"`
}

type Fragment struct {
	Packets     string `json:"packets"`
	Length      string `json:"length"`
	Interval    string `json:"interval"`
	Host1Header string `json:"host1_header"`
	Host1Domain string `json:"host1_domain"`
	Host2Header string `json:"host2_header"`
	Host2Domain string `json:"host2_domain"`
}

func (c *Fragment) Build() *internet.SocketConfig_Fragment {
	return &internet.SocketConfig_Fragment{
		Packets:     c.Packets,
		Length:      c.Length,
		Interval:    c.Interval,
		Host1Header: c.Host1Header,
		Host1Domain: c.Host1Domain,
		Host2Header: c.Host2Header,
		Host2Domain: c.Host2Domain,
	}
}

type Noise struct {
	Type   string          `json:"type"`
	Packet string          `json:"packet"`
	Delay  json.RawMessage `json:"delay"`
}

func (c *Noise) Build() *internet.SocketConfig_Noise {
	var delayStr string
	var delayInt int32
	var delay string
	if err := json.Unmarshal(c.Delay, &delayStr); err == nil {
		delay = delayStr
	} else if err := json.Unmarshal(c.Delay, &delayInt); err == nil {
		delay = fmt.Sprint(delayInt) + "-" + fmt.Sprint(delayInt)
	}
	return &internet.SocketConfig_Noise{
		Type:   c.Type,
		Packet: c.Packet,
		Delay:  delay,
	}
}

// Build implements Buildable.
func (c *SocketConfig) Build() (*internet.SocketConfig, error) {
	var tfoSettings internet.SocketConfig_TCPFastOpenState
	if c.TFO != nil {
		if *c.TFO {
			tfoSettings = internet.SocketConfig_Enable
		} else {
			tfoSettings = internet.SocketConfig_Disable
		}
	}

	tfoQueueLength := c.TFOQueueLength
	if tfoQueueLength == 0 {
		tfoQueueLength = 4096
	}

	var tproxy internet.SocketConfig_TProxyMode
	switch strings.ToLower(c.TProxy) {
	case "tproxy":
		tproxy = internet.SocketConfig_TProxy
	case "redirect":
		tproxy = internet.SocketConfig_Redirect
	default:
		tproxy = internet.SocketConfig_Off
	}

	var mptcpSettings internet.MPTCPState
	if c.MPTCP != nil {
		if *c.MPTCP {
			mptcpSettings = internet.MPTCPState_Enable
		} else {
			mptcpSettings = internet.MPTCPState_Disable
		}
	}

	config := &internet.SocketConfig{
		Mark:                 c.Mark,
		Tfo:                  tfoSettings,
		TfoQueueLength:       tfoQueueLength,
		Tproxy:               tproxy,
		AcceptProxyProtocol:  c.AcceptProxyProtocol,
		TcpKeepAliveInterval: c.TCPKeepAliveInterval,
		TcpKeepAliveIdle:     c.TCPKeepAliveIdle,
		RxBufSize:            int64(c.RxBufSize),
		TxBufSize:            int64(c.TxBufSize),
		ForceBufSize:         c.ForceBufSize,
		BindToDevice:         c.BindToDevice,
		Mptcp:                mptcpSettings,
		DialerProxy:          c.DialerProxy,
	}

	if c.Fragment != nil {
		config.Fragment = c.Fragment.Build()
	}
	if c.Noises != nil {
		for _, noise := range c.Noises {
			config.Noises = append(config.Noises, noise.Build())
		}
	}

	return config, nil
}
