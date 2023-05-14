package quic

import (
	"sync"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/bytespool"
)

var pool *sync.Pool

func init() {
	pool = bytespool.GetPool(buf.Size)
}

func getBuffer() []byte {
	return pool.Get().([]byte)
}

func putBuffer(p []byte) {
	pool.Put(p) // nolint: staticcheck
}
