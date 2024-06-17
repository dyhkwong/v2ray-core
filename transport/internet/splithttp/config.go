package splithttp

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type RangeConfig struct {
	From int32
	To   int32
}

func newRandRangeConfig(defaultFrom, defaultTo int32, randRange string) (config *RangeConfig) {
	config = &RangeConfig{
		From: defaultFrom,
		To:   defaultTo,
	}
	if len(randRange) == 0 {
		return
	}
	from, to, err := parseRangeString(randRange)
	if err != nil || to == 0 {
		return
	}
	config.From = int32(from)
	config.To = int32(to)
	return
}

func (c *RangeConfig) rand() int32 {
	if c.From == c.To {
		return c.From
	}
	bigInt, _ := rand.Int(rand.Reader, big.NewInt(int64(c.To-c.From)))
	return c.From + int32(bigInt.Int64())
}

func parseRangeString(str string) (int, int, error) {
	// for number in string format like "114" or "-1"
	if value, err := strconv.Atoi(str); err == nil {
		return value, value, nil
	}
	// for empty "", we treat it as 0
	if str == "" {
		return 0, 0, nil
	}
	// for range value, like "114-514"
	var pair []string
	// Process sth like "-114-514" "-1919--810"
	if strings.HasPrefix(str, "-") {
		pair = splitFromSecondDash(str)
	} else {
		pair = strings.SplitN(str, "-", 2)
	}
	if len(pair) == 2 {
		left, err := strconv.Atoi(pair[0])
		right, err2 := strconv.Atoi(pair[1])
		if err == nil && err2 == nil {
			return left, right, nil
		}
	}
	return 0, 0, newError("invalid range string: ", str)
}

func splitFromSecondDash(s string) []string {
	parts := strings.SplitN(s, "-", 3)
	if len(parts) < 3 {
		return []string{s}
	}
	return []string{parts[0] + "-" + parts[1], parts[2]}
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
	query += "x_padding=" + strings.Repeat("X", int(c.GetNormalizedXPaddingBytes().From))
	return query
}

func (c *Config) GetRequestHeader(rawURL string) (http.Header, error) {
	header := http.Header{}
	for k, v := range c.Headers {
		header.Add(k, v)
	}
	if paddingLen := c.GetNormalizedXPaddingBytes().rand(); paddingLen > 0 {
		u, err := url.Parse(rawURL)
		if err != nil {
			return nil, err
		}
		// https://www.rfc-editor.org/rfc/rfc7541.html#appendix-B
		// h2's HPACK Header Compression feature employs a huffman encoding using a static table.
		// 'X' is assigned an 8 bit code, so HPACK compression won't change actual padding length on the wire.
		// https://www.rfc-editor.org/rfc/rfc9204.html#section-4.1.2-2
		// h3's similar QPACK feature uses the same huffman table.
		u.RawQuery = "x_padding=" + strings.Repeat("X", int(c.GetNormalizedXPaddingBytes().rand()))
		header.Set("Referer", u.String())
	}
	return header, nil
}

func (c *Config) WriteResponseHeader(writer http.ResponseWriter) {
	// CORS headers for the browser dialer
	writer.Header().Set("Access-Control-Allow-Origin", "*")
	writer.Header().Set("Access-Control-Allow-Methods", "GET, POST")
	if paddingLen := c.GetNormalizedXPaddingBytes().rand(); paddingLen > 0 {
		writer.Header().Set("X-Padding", strings.Repeat("X", int(paddingLen)))
	}
}

func (c *Config) GetNormalizedScMaxBufferedPosts() int {
	if c.ScMaxBufferedPosts == 0 {
		return 30
	}
	return int(c.ScMaxBufferedPosts)
}

func (c *Config) GetNormalizedScMaxEachPostBytes() *RangeConfig {
	return newRandRangeConfig(1000000, 1000000, c.ScMaxEachPostBytes)
}

func (c *Config) GetNormalizedScMinPostsIntervalMs() *RangeConfig {
	return newRandRangeConfig(30, 30, c.ScMinPostsIntervalMs)
}

func (c *Config) GetNormalizedXPaddingBytes() *RangeConfig {
	return newRandRangeConfig(100, 1000, c.XPaddingBytes)
}
