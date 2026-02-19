package splithttp_test

import (
	"context"
	"testing"

	"github.com/v2fly/v2ray-core/v5/common"
	. "github.com/v2fly/v2ray-core/v5/transport/internet/splithttp"
)

type fakeRoundTripper struct{}

func (f *fakeRoundTripper) IsClosed() bool {
	return false
}

func TestMaxConnections(t *testing.T) {
	xmuxConfig := &XmuxConfig{
		MaxConnections: "4-4",
	}

	xmuxManager, err := NewXmuxManager(xmuxConfig, func() (XmuxConn, error) {
		return &fakeRoundTripper{}, nil
	})
	common.Must(err)

	xmuxClients := make(map[any]struct{})
	for range 8 {
		xmuxClient, err := xmuxManager.GetXmuxClient(context.Background())
		common.Must(err)
		xmuxClients[xmuxClient] = struct{}{}
	}

	if len(xmuxClients) != 4 {
		t.Error("did not get 4 distinct clients, got ", len(xmuxClients))
	}
}

func TestCMaxReuseTimes(t *testing.T) {
	xmuxConfig := &XmuxConfig{
		CMaxReuseTimes: "2-2",
	}

	xmuxManager, err := NewXmuxManager(xmuxConfig, func() (XmuxConn, error) {
		return &fakeRoundTripper{}, nil
	})
	common.Must(err)

	xmuxClients := make(map[any]struct{})
	for range 64 {
		xmuxClient, err := xmuxManager.GetXmuxClient(context.Background())
		common.Must(err)
		xmuxClients[xmuxClient] = struct{}{}
	}

	if len(xmuxClients) != 32 {
		t.Error("did not get 32 distinct clients, got ", len(xmuxClients))
	}
}

func TestMaxConcurrency(t *testing.T) {
	xmuxConfig := &XmuxConfig{
		MaxConcurrency: "2-2",
	}

	xmuxManager, err := NewXmuxManager(xmuxConfig, func() (XmuxConn, error) {
		return &fakeRoundTripper{}, nil
	})
	common.Must(err)

	xmuxClients := make(map[any]struct{})
	for range 64 {
		xmuxClient, err := xmuxManager.GetXmuxClient(context.Background())
		common.Must(err)
		xmuxClient.OpenUsage.Add(1)
		xmuxClients[xmuxClient] = struct{}{}
	}

	if len(xmuxClients) != 32 {
		t.Error("did not get 32 distinct clients, got ", len(xmuxClients))
	}
}
