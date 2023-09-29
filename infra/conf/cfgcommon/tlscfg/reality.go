package tlscfg

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/transport/internet/reality"
)

type REALITYConfig struct {
	Dest        json.RawMessage `json:"dest"`
	Target      json.RawMessage `json:"target"`
	Type        string          `json:"type"`
	Xver        uint64          `json:"xver"`
	ServerNames []string        `json:"serverNames"`
	Password    string          `json:"password"`
	PrivateKey  string          `json:"privateKey"`
	ShortIds    []string        `json:"shortIds"`

	Fingerprint              string `json:"fingerprint"`
	ServerName               string `json:"serverName"`
	PublicKey                string `json:"publicKey"`
	ShortId                  string `json:"shortId"`
	Version                  string `json:"version"`
	DisableX25519MLKEM768    bool   `json:"disableX25519MLKEM768"`
	ReenableCHACHA20POLY1305 bool   `json:"reenableCHACHA20POLY1305"`
}

// Build implements Buildable.

func (c *REALITYConfig) Build() (proto.Message, error) {
	config := new(reality.Config)
	var err error
	if c.Dest == nil {
		c.Dest = c.Target
	}
	if c.Dest != nil {
		var i uint16
		var s string
		if err = json.Unmarshal(c.Dest, &i); err == nil {
			s = strconv.Itoa(int(i))
		} else {
			_ = json.Unmarshal(c.Dest, &s)
		}
		if c.Type == "" && s != "" {
			switch {
			case s[0] == '@', filepath.IsAbs(s):
				c.Type = "unix"
				if s[0] == '@' && len(s) > 1 && s[1] == '@' && (runtime.GOOS == "linux" || runtime.GOOS == "android") {
					fullAddr := make([]byte, len(syscall.RawSockaddrUnix{}.Path)) // may need padding to work with haproxy
					copy(fullAddr, s[1:])
					s = string(fullAddr)
				}
			default:
				if _, err = strconv.Atoi(s); err == nil {
					s = "127.0.0.1:" + s
				}
				if _, _, err = net.SplitHostPort(s); err == nil {
					c.Type = "tcp"
				}
			}
		}
		if c.Type == "" {
			return nil, newError(`please fill in a valid value for "dest"`)
		}
		if c.Xver > 2 {
			return nil, newError(`invalid PROXY protocol version, "xver" only accepts 0, 1, 2`)
		}
		if len(c.ServerNames) == 0 {
			return nil, newError(`empty "serverNames"`)
		}
		if c.Password != "" {
			c.PublicKey = c.Password
		}
		if c.PrivateKey == "" {
			return nil, newError(`empty "password"`)
		}
		if config.PrivateKey, err = base64.RawURLEncoding.DecodeString(c.PrivateKey); err != nil || len(config.PrivateKey) != 32 {
			return nil, newError(`invalid "password": `, c.PrivateKey)
		}
		if len(c.ShortIds) == 0 {
			c.ShortIds = []string{""}
		}
		config.ShortIds = make([][]byte, len(c.ShortIds))
		for i, s := range c.ShortIds {
			config.ShortIds[i] = make([]byte, 8)
			if _, err = hex.Decode(config.ShortIds[i], []byte(s)); err != nil {
				return nil, newError(`invalid "shortIds[`, i, `]": `, s)
			}
		}
		config.Dest = s
		config.Type = c.Type
		config.Xver = c.Xver
		config.ServerNames = c.ServerNames
	} else {
		config.Fingerprint = strings.ToLower(c.Fingerprint)
		if len(c.ServerNames) != 0 {
			return nil, newError(`non-empty "serverNames", please use "serverName" instead`)
		}
		if c.PublicKey == "" {
			return nil, newError(`empty "publicKey"`)
		}
		if config.PublicKey, err = base64.RawURLEncoding.DecodeString(c.PublicKey); err != nil || len(config.PublicKey) != 32 {
			return nil, newError(`invalid "publicKey": `, c.PublicKey)
		}
		if len(c.ShortIds) != 0 {
			return nil, newError(`non-empty "shortIds", please use "shortId" instead`)
		}
		config.ShortId = make([]byte, 8)
		if _, err = hex.Decode(config.ShortId, []byte(c.ShortId)); err != nil {
			return nil, newError(`invalid "shortId": `, c.ShortId)
		}
		config.ServerName = c.ServerName

		if c.Version != "" {
			config.Version = make([]byte, 3)
			var u uint64
			for i, s := range strings.Split(c.Version, ".") {
				if i == 3 {
					return nil, newError(`invalid "verion": `, c.Version)
				}
				if u, err = strconv.ParseUint(s, 10, 8); err != nil {
					return nil, newError(`"verion[`, i, `]" should be lesser than 256`)
				} else {
					config.Version[i] = byte(u)
				}
			}
		}
		config.DisableX25519Mlkem768 = c.DisableX25519MLKEM768
		config.ReenableChacha20Poly1305 = c.ReenableCHACHA20POLY1305
	}
	return config, nil
}
