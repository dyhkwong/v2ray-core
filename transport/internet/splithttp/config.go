package splithttp

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"strconv"
	"strings"
)

type RangeConfig struct {
	From int
	To   int
}

func newRandRangeConfig(defaultFrom, defaultTo int, randRange string) (config *RangeConfig) {
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
	config.From = from
	config.To = to
	return
}

func (c *RangeConfig) rand() int {
	if c.From == c.To {
		return c.From
	}
	bigInt, _ := rand.Int(rand.Reader, big.NewInt(int64(c.To-c.From)))
	return c.From + int(bigInt.Int64())
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
	paddingLen := c.GetNormalizedXPaddingBytes().rand()
	if paddingLen > 0 {
		query += "x_padding=" + strings.Repeat("0", int(paddingLen))
	}
	return query
}

func (c *Config) GetRequestHeader() http.Header {
	header := http.Header{}
	for k, v := range c.Headers {
		header.Add(k, v)
	}
	return header
}

func (c *Config) WriteResponseHeader(writer http.ResponseWriter) {
	// CORS headers for the browser dialer
	writer.Header().Set("Access-Control-Allow-Origin", "*")
	writer.Header().Set("Access-Control-Allow-Methods", "GET, POST")
	paddingLen := c.GetNormalizedXPaddingBytes().rand()
	if paddingLen > 0 {
		writer.Header().Set("X-Padding", strings.Repeat("0", int(paddingLen)))
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
