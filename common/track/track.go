package track

import (
	"container/list"
	"sync"

	"github.com/v2fly/v2ray-core/v5/common"
)

type ConnectionPool struct {
	list.List
	sync.Mutex
}

func NewConnectionPool() *ConnectionPool {
	return new(ConnectionPool)
}

func (p *ConnectionPool) ResetConnections() {
	p.Lock()
	for elem := p.Front(); elem != nil; elem = elem.Next() {
		common.Close(elem.Value)
	}
	p.Init()
	p.Unlock()
}
