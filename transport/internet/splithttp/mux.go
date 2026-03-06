package splithttp

import (
	"context"
	"crypto/rand"
	"math"
	"math/big"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/v2fly/v2ray-core/v5/common/session"
)

type XmuxConn interface {
	IsClosed() bool
}

type XmuxClient struct {
	XmuxConn     XmuxConn
	OpenUsage    atomic.Int32
	leftUsage    int32
	LeftRequests atomic.Int32
	UnreusableAt time.Time
}

type XmuxManager struct {
	xmuxConfig  *XmuxConfig
	concurrency int32
	connections int32
	newConnFunc func() XmuxConn
	xmuxClients []*XmuxClient
}

func NewXmuxManager(xmuxConfig *XmuxConfig, newConnFunc func() XmuxConn) (*XmuxManager, error) {
	var config *XmuxConfig
	if xmuxConfig == nil || reflect.DeepEqual(xmuxConfig, &XmuxConfig{}) {
		config = &XmuxConfig{
			MaxConcurrency:   "1-1",
			HMaxRequestTimes: "600-900",
			HMaxReusableSecs: "1800-3000",
		}
	} else {
		config = &XmuxConfig{
			MaxConcurrency:   xmuxConfig.MaxConcurrency,
			MaxConnections:   xmuxConfig.MaxConnections,
			CMaxReuseTimes:   xmuxConfig.CMaxReuseTimes,
			HMaxRequestTimes: xmuxConfig.HMaxRequestTimes,
			HMaxReusableSecs: xmuxConfig.HMaxReusableSecs,
		}
	}
	maxConcurrency := config.GetNormalizedMaxConcurrency()
	maxConnections := config.GetNormalizedMaxConnections()
	if maxConnections.To > 0 && maxConcurrency.To > 0 {
		return nil, newError("maxConnections cannot be specified together with maxConcurrency")
	}
	return &XmuxManager{
		xmuxConfig:  config,
		concurrency: maxConcurrency.rand(),
		connections: maxConnections.rand(),
		newConnFunc: newConnFunc,
		xmuxClients: make([]*XmuxClient, 0),
	}, nil
}

func (m *XmuxManager) newXmuxClient() *XmuxClient {
	xmuxClient := &XmuxClient{
		XmuxConn:  m.newConnFunc(),
		leftUsage: -1,
	}
	if x := m.xmuxConfig.GetNormalizedCMaxReuseTimes().rand(); x > 0 {
		xmuxClient.leftUsage = x - 1
	}
	xmuxClient.LeftRequests.Store(math.MaxInt32)
	if x := m.xmuxConfig.GetNormalizedHMaxRequestTimes().rand(); x > 0 {
		xmuxClient.LeftRequests.Store(x)
	}
	if x := m.xmuxConfig.GetNormalizedHMaxReusableSecs().rand(); x > 0 {
		xmuxClient.UnreusableAt = time.Now().Add(time.Duration(x) * time.Second)
	}
	m.xmuxClients = append(m.xmuxClients, xmuxClient)
	return xmuxClient
}

func (m *XmuxManager) GetXmuxClient(ctx context.Context) *XmuxClient { // when locking
	for i := 0; i < len(m.xmuxClients); {
		xmuxClient := m.xmuxClients[i]
		if xmuxClient.XmuxConn.IsClosed() ||
			xmuxClient.leftUsage == 0 ||
			xmuxClient.LeftRequests.Load() <= 0 ||
			(xmuxClient.UnreusableAt != time.Time{} && time.Now().After(xmuxClient.UnreusableAt)) {
			newError("XMUX: removing xmuxClient, IsClosed() = ", xmuxClient.XmuxConn.IsClosed(),
				", OpenUsage = ", xmuxClient.OpenUsage.Load(),
				", leftUsage = ", xmuxClient.leftUsage,
				", LeftRequests = ", xmuxClient.LeftRequests.Load(),
				", UnreusableAt = ", xmuxClient.UnreusableAt,
			).AtDebug().WriteToLog(session.ExportIDToError(ctx))
			m.xmuxClients = append(m.xmuxClients[:i], m.xmuxClients[i+1:]...)
		} else {
			i++
		}
	}

	if len(m.xmuxClients) == 0 {
		newError("XMUX: creating xmuxClient because xmuxClients is empty").AtDebug().WriteToLog(session.ExportIDToError(ctx))
		return m.newXmuxClient()
	}

	if m.connections > 0 && len(m.xmuxClients) < int(m.connections) {
		newError("XMUX: creating xmuxClient because maxConnections was not hit, xmuxClients = ", len(m.xmuxClients)).AtDebug().WriteToLog(session.ExportIDToError(ctx))
		return m.newXmuxClient()
	}

	xmuxClients := make([]*XmuxClient, 0)
	if m.concurrency > 0 {
		for _, xmuxClient := range m.xmuxClients {
			if xmuxClient.OpenUsage.Load() < m.concurrency {
				xmuxClients = append(xmuxClients, xmuxClient)
			}
		}
	} else {
		xmuxClients = m.xmuxClients
	}

	if len(xmuxClients) == 0 {
		newError("XMUX: creating xmuxClient because maxConcurrency was hit, xmuxClients = ", len(m.xmuxClients)).AtDebug().WriteToLog(session.ExportIDToError(ctx))
		return m.newXmuxClient()
	}

	i, _ := rand.Int(rand.Reader, big.NewInt(int64(len(xmuxClients))))
	xmuxClient := xmuxClients[i.Int64()]
	if xmuxClient.leftUsage > 0 {
		xmuxClient.leftUsage -= 1
	}
	return xmuxClient
}
