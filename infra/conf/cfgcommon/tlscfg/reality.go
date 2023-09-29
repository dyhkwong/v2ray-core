package tlscfg

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/url"
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
	Show        bool            `json:"show"`
	Dest        json.RawMessage `json:"dest"`
	Target      json.RawMessage `json:"target"`
	Type        string          `json:"type"`
	Xver        uint64          `json:"xver"`
	ServerNames []string        `json:"serverNames"`
	PrivateKey  string          `json:"privateKey"`
	ShortIds    []string        `json:"shortIds"`

	Fingerprint string `json:"fingerprint"`
	ServerName  string `json:"serverName"`
	PublicKey   string `json:"publicKey"`
	ShortId     string `json:"shortId"`
	SpiderX     string `json:"spiderX"`
	Version     string `json:"version"`
}

// Build implements Buildable.

func (c *REALITYConfig) Build() (proto.Message, error) {
	config := new(reality.Config)
	config.Show = c.Show
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
		if c.PrivateKey == "" {
			return nil, newError(`empty "privateKey"`)
		}
		if config.PrivateKey, err = base64.RawURLEncoding.DecodeString(c.PrivateKey); err != nil || len(config.PrivateKey) != 32 {
			return nil, newError(`invalid "privateKey": `, c.PrivateKey)
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
		if c.Fingerprint == "" {
			c.Fingerprint = "chrome"
		}
		if config.Fingerprint = strings.ToLower(c.Fingerprint); reality.GetFingerprint(config.Fingerprint) == nil {
			return nil, newError(`unknown "fingerprint": `, config.Fingerprint)
		}
		if config.Fingerprint == "hellogolang" {
			return nil, newError(`invalid "fingerprint": `, config.Fingerprint)
		}
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
		if c.SpiderX == "" {
			c.SpiderX = "/"
		}
		if c.SpiderX[0] != '/' {
			return nil, newError(`invalid "spiderX": `, c.SpiderX)
		}
		config.SpiderY = make([]int64, 10)
		u, _ := url.Parse(c.SpiderX)
		q := u.Query()
		parse := func(param string, index int) {
			if q.Get(param) != "" {
				s := strings.Split(q.Get(param), "-")
				if len(s) == 1 {
					config.SpiderY[index], _ = strconv.ParseInt(s[0], 10, 64)
					config.SpiderY[index+1], _ = strconv.ParseInt(s[0], 10, 64)
				} else {
					config.SpiderY[index], _ = strconv.ParseInt(s[0], 10, 64)
					config.SpiderY[index+1], _ = strconv.ParseInt(s[1], 10, 64)
				}
			}
			q.Del(param)
		}
		parse("p", 0) // padding
		parse("c", 2) // concurrency
		parse("t", 4) // times
		parse("i", 6) // interval
		parse("r", 8) // return
		u.RawQuery = q.Encode()
		config.SpiderX = u.String()
		config.ServerName = c.ServerName

		config.Version = make([]byte, 3)
		if c.Version != "" {
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
		} else {
			config.Version[0] = 24 // Version_x
			config.Version[1] = 10 // Version_y
			config.Version[2] = 31 // Version_z
		}
	}
	return config, nil
}
