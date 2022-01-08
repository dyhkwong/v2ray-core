package httpfetcher

import (
	"context"

	"github.com/v2fly/v2ray-core/v5/app/subscription"
	"github.com/v2fly/v2ray-core/v5/app/subscription/documentfetcher"
	"github.com/v2fly/v2ray-core/v5/common"
)

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

func newHTTPFetcher() *httpFetcher {
	return &httpFetcher{}
}

func init() {
	common.Must(documentfetcher.RegisterFetcher("http", newHTTPFetcher()))
}

type httpFetcher struct{}

func (h *httpFetcher) DownloadDocument(ctx context.Context, source *subscription.ImportSource, opts ...documentfetcher.FetcherOptions) ([]byte, error) {
	return nil, newError("unsupported: remote config may lead to RCE vulnerabilities")
}
