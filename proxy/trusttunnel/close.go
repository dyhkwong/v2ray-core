package trusttunnel

import (
	"reflect"
	"sync"
	"unsafe"

	"golang.org/x/net/http2"
)

//go:linkname connPool golang.org/x/net/http2.(*Transport).connPool
func connPool(transport *http2.Transport) http2.ClientConnPool

func closeHTTP2Transport(transport *http2.Transport) {
	transport.CloseIdleConnections()

	defer func() {
		if r := recover(); r != nil {
			newError("panic recovered: ", r).AtError().WriteToLog()
		}
	}()

	v := reflect.ValueOf(connPool(transport)).Elem()
	mu := (*sync.Mutex)(unsafe.Pointer(v.FieldByName("mu").UnsafeAddr()))
	conns := *(*map[string][]*http2.ClientConn)(unsafe.Pointer(v.FieldByName("conns").UnsafeAddr()))

	mu.Lock()
	for _, conn := range conns {
		for _, c := range conn {
			c.Close()
		}
	}
	mu.Unlock()
}
