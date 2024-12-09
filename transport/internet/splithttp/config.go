package splithttp

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"strconv"
	"strings"
)

type RandRangeConfig struct {
	From int
	To   int
}

func (c *RandRangeConfig) roll() int {
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

func (c *Config) GetNormalizedQuery() (string, error) {
	pathAndQuery := strings.SplitN(c.Path, "?", 2)
	query := ""

	if len(pathAndQuery) > 1 {
		query = pathAndQuery[1]
	}

	if query != "" {
		query += "&"
	}

	normalizedXPaddingBytes, err := c.GetNormalizedXPaddingBytes()
	if err != nil {
		return "", err
	}
	paddingLen := normalizedXPaddingBytes.roll()
	if paddingLen > 0 {
		query += "x_padding=" + strings.Repeat("0", int(paddingLen))
	}

	return query, nil
}

func (c *Config) GetRequestHeader() http.Header {
	header := http.Header{}
	for k, v := range c.Header {
		header.Add(k, v)
	}

	return header
}

func (c *Config) WriteResponseHeader(writer http.ResponseWriter) error {
	// CORS headers for the browser dialer
	writer.Header().Set("Access-Control-Allow-Origin", "*")
	writer.Header().Set("Access-Control-Allow-Methods", "GET, POST")
	normalizedXPaddingBytes, err := c.GetNormalizedXPaddingBytes()
	if err != nil {
		return err
	}
	paddingLen := normalizedXPaddingBytes.roll()
	if paddingLen > 0 {
		writer.Header().Set("X-Padding", strings.Repeat("0", int(paddingLen)))
	}
	return nil
}

func (c *Config) GetNormalizedScMaxConcurrentPosts() (*RandRangeConfig, error) {
	if len(c.ScMaxConcurrentPosts) == 0 {
		return &RandRangeConfig{
			From: 100,
			To:   100,
		}, nil
	}
	from, to, err := parseRangeString(c.ScMaxConcurrentPosts)
	if err != nil {
		return nil, err
	}
	if to == 0 {
		return &RandRangeConfig{
			From: 100,
			To:   100,
		}, nil
	}
	return &RandRangeConfig{
		From: from,
		To:   to,
	}, nil
}

func (c *Config) GetNormalizedScMaxEachPostBytes() (*RandRangeConfig, error) {
	if len(c.ScMaxEachPostBytes) == 0 {
		return &RandRangeConfig{
			From: 1000000,
			To:   1000000,
		}, nil
	}
	from, to, err := parseRangeString(c.ScMaxEachPostBytes)
	if err != nil {
		return nil, err
	}
	if to == 0 {
		return &RandRangeConfig{
			From: 1000000,
			To:   1000000,
		}, nil
	}
	return &RandRangeConfig{
		From: from,
		To:   to,
	}, nil
}

func (c *Config) GetNormalizedScMinPostsIntervalMs() (*RandRangeConfig, error) {
	if len(c.ScMinPostsIntervalMs) == 0 {
		return &RandRangeConfig{
			From: 30,
			To:   30,
		}, nil
	}
	from, to, err := parseRangeString(c.ScMinPostsIntervalMs)
	if err != nil {
		return nil, err
	}
	if to == 0 {
		return &RandRangeConfig{
			From: 30,
			To:   30,
		}, nil
	}
	return &RandRangeConfig{
		From: from,
		To:   to,
	}, nil
}

func (c *Config) GetNormalizedXPaddingBytes() (*RandRangeConfig, error) {
	if len(c.XPaddingBytes) == 0 {
		return &RandRangeConfig{
			From: 100,
			To:   1000,
		}, nil
	}
	from, to, err := parseRangeString(c.XPaddingBytes)
	if err != nil {
		return nil, err
	}
	if to == 0 {
		return &RandRangeConfig{
			From: 100,
			To:   1000,
		}, nil
	}
	return &RandRangeConfig{
		From: from,
		To:   to,
	}, nil
}
