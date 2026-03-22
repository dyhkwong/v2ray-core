package splithttp

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/serial"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

const (
	PlacementQueryInHeader = "queryInHeader"
	PlacementCookie        = "cookie"
	PlacementHeader        = "header"
	PlacementQuery         = "query"
	PlacementPath          = "path"
	PlacementBody          = "body"
	PlacementAuto          = "auto"
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
		return config
	}
	from, to, err := parseRangeString(randRange)
	if err != nil || to == 0 {
		return config
	}
	config.From = int32(from)
	config.To = int32(to)
	return config
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
	return query
}

func (c *Config) GetRequestHeader() http.Header {
	header := http.Header{}
	for k, v := range c.Headers {
		header.Add(k, v)
	}
	return header
}

func (c *Config) GetRequestHeaderWithPayload(payload []byte) http.Header {
	header := c.GetRequestHeader()

	key := c.UplinkDataKey
	encodedData := base64.RawURLEncoding.EncodeToString(payload)

	for i := 0; len(encodedData) > 0; i++ {
		chunkSize := min(int(c.GetNormalizedUplinkChunkSize().rand()), len(encodedData))
		chunk := encodedData[:chunkSize]
		encodedData = encodedData[chunkSize:]
		headerKey := fmt.Sprintf("%s-%d", key, i)
		header.Set(headerKey, chunk)
	}

	return header
}

func (c *Config) WriteResponseHeader(writer http.ResponseWriter, requestMethod string, requestHeader http.Header) {
	// CORS headers for the browser dialer
	if origin := requestHeader.Get("Origin"); origin == "" {
		writer.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		// Chrome says: The value of the 'Access-Control-Allow-Origin' header in the response must not be the wildcard '*' when the request's credentials mode is 'include'.
		writer.Header().Set("Access-Control-Allow-Origin", origin)
	}

	if c.GetNormalizedSessionPlacement() == PlacementCookie ||
		c.GetNormalizedSeqPlacement() == PlacementCookie ||
		c.XPaddingPlacement == PlacementCookie ||
		c.GetNormalizedUplinkDataPlacement() == PlacementCookie {
		writer.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	if requestMethod == "OPTIONS" {
		requestedMethod := requestHeader.Get("Access-Control-Request-Method")
		if requestedMethod != "" {
			writer.Header().Set("Access-Control-Allow-Methods", requestedMethod)
		} else {
			writer.Header().Set("Access-Control-Allow-Methods", "*")
		}

		requestedHeaders := requestHeader.Get("Access-Control-Request-Headers")
		if requestedHeaders == "" {
			writer.Header().Set("Access-Control-Allow-Headers", "*")
		} else {
			writer.Header().Set("Access-Control-Allow-Headers", requestedHeaders)
		}
	}
}

func (c *Config) GetRequestCookiesWithPayload(payload []byte) []*http.Cookie {
	cookies := []*http.Cookie{}

	key := c.UplinkDataKey
	encodedData := base64.RawURLEncoding.EncodeToString(payload)

	for i := 0; len(encodedData) > 0; i++ {
		chunkSize := min(int(c.GetNormalizedUplinkChunkSize().rand()), len(encodedData))
		chunk := encodedData[:chunkSize]
		encodedData = encodedData[chunkSize:]
		cookieName := fmt.Sprintf("%s_%d", key, i)
		cookies = append(cookies, &http.Cookie{Name: cookieName, Value: chunk})
	}

	return cookies
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

func (c *Config) GetNormalizedUplinkHTTPMethod() string {
	if len(c.UplinkHTTPMethod) == 0 {
		return "POST"
	}
	return c.UplinkHTTPMethod
}

func (c *Config) GetNormalizedUplinkChunkSize() *RangeConfig {
	from, to, err := parseRangeString(c.UplinkChunkSize)
	if err != nil || to == 0 {
		switch c.UplinkDataPlacement {
		case PlacementCookie:
			return &RangeConfig{
				From: 2 * 1024, // 2 KiB
				To:   3 * 1024, // 3 KiB
			}
		case PlacementHeader:
			return &RangeConfig{
				From: 3 * 1000, // 3 KB
				To:   4 * 1000, // 4 KB
			}
		default:
			return c.GetNormalizedScMaxEachPostBytes()
		}
	}
	if from < 64 {
		return &RangeConfig{
			From: 64,
			To:   int32(max(64, to)),
		}
	}
	return &RangeConfig{
		From: int32(from),
		To:   int32(to),
	}
}

func (c *Config) GetNormalizedSessionPlacement() string {
	if c.SessionPlacement == "" {
		return PlacementPath
	}
	return c.SessionPlacement
}

func (c *Config) GetNormalizedSeqPlacement() string {
	if c.SeqPlacement == "" {
		return PlacementPath
	}
	return c.SeqPlacement
}

func (c *Config) GetNormalizedUplinkDataPlacement() string {
	if c.UplinkDataPlacement == "" {
		return PlacementBody
	}
	return c.UplinkDataPlacement
}

func (c *Config) GetNormalizedSessionKey() string {
	if c.SessionKey != "" {
		return c.SessionKey
	}
	switch c.GetNormalizedSessionPlacement() {
	case PlacementHeader:
		return "X-Session"
	case PlacementCookie, PlacementQuery:
		return "x_session"
	default:
		return ""
	}
}

func (c *Config) GetNormalizedSeqKey() string {
	if c.SeqKey != "" {
		return c.SeqKey
	}
	switch c.GetNormalizedSeqPlacement() {
	case PlacementHeader:
		return "X-Seq"
	case PlacementCookie, PlacementQuery:
		return "x_seq"
	default:
		return ""
	}
}

func (c *Config) ApplyMetaToRequest(req *http.Request, sessionId string, seqStr string) {
	sessionPlacement := c.GetNormalizedSessionPlacement()
	seqPlacement := c.GetNormalizedSeqPlacement()
	sessionKey := c.GetNormalizedSessionKey()
	seqKey := c.GetNormalizedSeqKey()

	if sessionId != "" {
		switch sessionPlacement {
		case PlacementPath:
			req.URL.Path = appendToPath(req.URL.Path, sessionId)
		case PlacementQuery:
			q := req.URL.Query()
			q.Set(sessionKey, sessionId)
			req.URL.RawQuery = q.Encode()
		case PlacementHeader:
			req.Header.Set(sessionKey, sessionId)
		case PlacementCookie:
			req.AddCookie(&http.Cookie{Name: sessionKey, Value: sessionId})
		}
	}

	if seqStr != "" {
		switch seqPlacement {
		case PlacementPath:
			req.URL.Path = appendToPath(req.URL.Path, seqStr)
		case PlacementQuery:
			q := req.URL.Query()
			q.Set(seqKey, seqStr)
			req.URL.RawQuery = q.Encode()
		case PlacementHeader:
			req.Header.Set(seqKey, seqStr)
		case PlacementCookie:
			req.AddCookie(&http.Cookie{Name: seqKey, Value: seqStr})
		}
	}
}

func (c *Config) FillStreamRequest(request *http.Request, sessionId string, seqStr string) {
	request.Header = c.GetRequestHeader()
	length := int(c.GetNormalizedXPaddingBytes().rand())
	config := &XPaddingConfig{Length: length}

	if c.XPaddingObfsMode {
		config.Placement = XPaddingPlacement{
			Placement: c.XPaddingPlacement,
			Key:       c.XPaddingKey,
			Header:    c.XPaddingHeader,
			RawURL:    request.URL.String(),
		}
		config.Method = PaddingMethod(c.XPaddingMethod)
	} else {
		config.Placement = XPaddingPlacement{
			Placement: PlacementQueryInHeader,
			Key:       "x_padding",
			Header:    "Referer",
			RawURL:    request.URL.String(),
		}
	}

	c.ApplyXPaddingToRequest(request, config)
	c.ApplyMetaToRequest(request, sessionId, "")

	if request.Body != nil && !c.NoGRPCHeader { // stream-up/one
		request.Header.Set("Content-Type", "application/grpc")
	}
}

func (c *Config) FillPacketRequest(request *http.Request, sessionId string, seqStr string, payload buf.MultiBuffer) error {
	dataPlacement := c.GetNormalizedUplinkDataPlacement()

	if dataPlacement == PlacementBody || dataPlacement == PlacementAuto {
		request.Header = c.GetRequestHeader()
		request.Body = io.NopCloser(&buf.MultiBufferContainer{MultiBuffer: payload})
		request.ContentLength = int64(payload.Len())
	} else {
		data := make([]byte, payload.Len())
		payload.Copy(data)
		buf.ReleaseMulti(payload)
		switch dataPlacement {
		case PlacementHeader:
			request.Header = c.GetRequestHeaderWithPayload(data)
		case PlacementCookie:
			request.Header = c.GetRequestHeader()
			for _, cookie := range c.GetRequestCookiesWithPayload(data) {
				request.AddCookie(cookie)
			}
		}
	}

	length := int(c.GetNormalizedXPaddingBytes().rand())
	config := &XPaddingConfig{Length: length}

	if c.XPaddingObfsMode {
		config.Placement = XPaddingPlacement{
			Placement: c.XPaddingPlacement,
			Key:       c.XPaddingKey,
			Header:    c.XPaddingHeader,
			RawURL:    request.URL.String(),
		}
		config.Method = PaddingMethod(c.XPaddingMethod)
	} else {
		config.Placement = XPaddingPlacement{
			Placement: PlacementQueryInHeader,
			Key:       "x_padding",
			Header:    "Referer",
			RawURL:    request.URL.String(),
		}
	}

	c.ApplyXPaddingToRequest(request, config)
	c.ApplyMetaToRequest(request, sessionId, seqStr)

	return nil
}

func (c *Config) ExtractMetaFromRequest(req *http.Request, path string) (sessionId string, seqStr string) {
	sessionPlacement := c.GetNormalizedSessionPlacement()
	seqPlacement := c.GetNormalizedSeqPlacement()
	sessionKey := c.GetNormalizedSessionKey()
	seqKey := c.GetNormalizedSeqKey()

	var subpath []string
	pathPart := 0
	if sessionPlacement == PlacementPath || seqPlacement == PlacementPath {
		subpath = strings.Split(req.URL.Path[len(path):], "/")
	}

	switch sessionPlacement {
	case PlacementPath:
		if len(subpath) > pathPart {
			sessionId = subpath[pathPart]
			pathPart += 1
		}
	case PlacementQuery:
		sessionId = req.URL.Query().Get(sessionKey)
	case PlacementHeader:
		sessionId = req.Header.Get(sessionKey)
	case PlacementCookie:
		if cookie, e := req.Cookie(sessionKey); e == nil {
			sessionId = cookie.Value
		}
	}

	switch seqPlacement {
	case PlacementPath:
		if len(subpath) > pathPart {
			seqStr = subpath[pathPart]
			pathPart += 1
		}
	case PlacementQuery:
		seqStr = req.URL.Query().Get(seqKey)
	case PlacementHeader:
		seqStr = req.Header.Get(seqKey)
	case PlacementCookie:
		if cookie, e := req.Cookie(seqKey); e == nil {
			seqStr = cookie.Value
		}
	}

	return sessionId, seqStr
}

func appendToPath(path, value string) string {
	if strings.HasSuffix(path, "/") {
		return path + value
	}
	return path + "/" + value
}

func (c *XmuxConfig) GetNormalizedMaxConcurrency() *RangeConfig {
	return newRandRangeConfig(0, 0, c.MaxConcurrency)
}

func (c *XmuxConfig) GetNormalizedMaxConnections() *RangeConfig {
	return newRandRangeConfig(0, 0, c.MaxConnections)
}

func (c *XmuxConfig) GetNormalizedCMaxReuseTimes() *RangeConfig {
	return newRandRangeConfig(0, 0, c.CMaxReuseTimes)
}

func (c *XmuxConfig) GetNormalizedHMaxRequestTimes() *RangeConfig {
	return newRandRangeConfig(0, 0, c.HMaxRequestTimes)
}

func (c *XmuxConfig) GetNormalizedHMaxReusableSecs() *RangeConfig {
	return newRandRangeConfig(0, 0, c.HMaxReusableSecs)
}

type memoryStreamConfig struct {
	ProtocolSettings any
	SecurityType     string
	SecuritySettings any
}

func toMemoryStreamConfig(s *DownloadConfig) (*memoryStreamConfig, error) {
	transportSettings := s.TransportSettings
	if transportSettings == nil {
		transportSettings = serial.ToTypedMessage(new(Config))
	}
	ets, err := serial.GetInstanceOf(transportSettings)
	if err != nil {
		return nil, err
	}
	mss := &memoryStreamConfig{
		ProtocolSettings: ets,
	}
	if len(s.SecurityType) > 0 {
		ess, err := serial.GetInstanceOf(s.SecuritySettings)
		if err != nil {
			return nil, err
		}
		mss.SecurityType = s.SecurityType
		mss.SecuritySettings = ess
	}
	return mss, nil
}

func (c *memoryStreamConfig) toInternetMemoryStreamConfig() *internet.MemoryStreamConfig {
	return &internet.MemoryStreamConfig{
		ProtocolName:     "splithttp",
		ProtocolSettings: c.ProtocolSettings,
		SecurityType:     c.SecurityType,
		SecuritySettings: c.SecuritySettings,
	}
}
