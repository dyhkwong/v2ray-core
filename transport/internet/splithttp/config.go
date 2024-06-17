package splithttp

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"strings"

	"github.com/v2fly/v2ray-core/v5/common/net"
)

const (
	scMaxEachPostBytes   = 1000000
	scMaxConcurrentPosts = 100
	scMinPostsIntervalMs = 30
	scMinXPaddingBytes   = 100
	scMaxXPaddingBytes   = 1000
)

func IsValidHTTPHost(request string, config string) bool {
	r := strings.ToLower(request)
	c := strings.ToLower(config)
	if strings.Contains(r, ":") {
		h, _, _ := net.SplitHostPort(r)
		return h == c
	}
	return r == c
}

func (c *Config) GetNormalizedPath() string {
	pathAndQuery := strings.SplitN(c.Path, "?", 2)
	path := pathAndQuery[0]

	if path == "" || path[0] != '/' {
		path = "/" + path
	}

	if path[len(path)-1] != '/' {
		path = path + "/"
	}

	return path
}

func (c *Config) GetNormalizedQuery() string {
	pathAndQuery := strings.SplitN(c.Path, "?", 2)
	query := ""

	if len(pathAndQuery) > 1 {
		query = pathAndQuery[1]
	}

	if query != "" {
		query += "&"
	}

	bigInt, _ := rand.Int(rand.Reader, big.NewInt(int64(scMaxXPaddingBytes-scMinXPaddingBytes)))
	paddingLen := scMinXPaddingBytes + int(bigInt.Int64())
	if paddingLen > 0 {
		query += "x_padding=" + strings.Repeat("0", int(paddingLen))
	}

	return query
}

func (c *Config) GetRequestHeader() http.Header {
	header := http.Header{}
	for k, v := range c.Header {
		header.Add(k, v)
	}

	return header
}

func (c *Config) WriteResponseHeader(writer http.ResponseWriter) {
	bigInt, _ := rand.Int(rand.Reader, big.NewInt(int64(scMaxXPaddingBytes-scMinXPaddingBytes)))
	paddingLen := scMinXPaddingBytes + int(bigInt.Int64())
	if paddingLen > 0 {
		writer.Header().Set("X-Padding", strings.Repeat("0", int(paddingLen)))
	}
}
