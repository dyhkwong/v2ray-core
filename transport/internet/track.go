package internet

import (
	"container/list"
	"net"
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

func NewTrackedConn(conn net.Conn, pool *ConnectionPool) *TrackedConn {
	pool.Lock()
	defer pool.Unlock()
	return &TrackedConn{
		Conn: conn,
		elem: pool.PushBack(conn),
		pool: pool,
	}
}

func UpdateTrackedConn(trackedConn *TrackedConn, conn net.Conn) *TrackedConn {
	trackedConn.pool.Lock()
	trackedConn.pool.Remove(trackedConn.elem)
	trackedConn.Conn = conn
	trackedConn.elem = trackedConn.pool.PushBack(conn)
	trackedConn.pool.Unlock()
	return trackedConn
}

type TrackedConn struct {
	net.Conn
	elem *list.Element
	pool *ConnectionPool
}

func (c *TrackedConn) Close() error {
	c.pool.Lock()
	c.pool.Remove(c.elem)
	c.pool.Unlock()
	return c.Conn.Close()
}
