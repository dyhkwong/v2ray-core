package v4

import (
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/app/browserdialer"
)

type BrowserDialerConfig struct {
	ListenAddr string `json:"listenAddr"`
	ListenPort int32  `json:"listenPort"`
}

func (b *BrowserDialerConfig) Build() (proto.Message, error) {
	b.ListenAddr = strings.TrimSpace(b.ListenAddr)
	if b.ListenAddr != "" && b.ListenPort == 0 {
		b.ListenPort = 54322
	}
	return &browserdialer.Config{
		ListenAddr: b.ListenAddr,
		ListenPort: b.ListenPort,
	}, nil
}
