package outbound

import (
	"sync"
)

var (
	globalHandlerIdx       uint64
	interfaceUpdatedFuncMu sync.Mutex
	interfaceUpdatedFunc   = make(map[uint64]func())
)

func RegisterInterfaceUpdateFunc(fn func()) uint64 {
	interfaceUpdatedFuncMu.Lock()
	defer interfaceUpdatedFuncMu.Unlock()
	globalHandlerIdx++
	interfaceUpdatedFunc[globalHandlerIdx] = fn
	return globalHandlerIdx
}

func UnRegisterInterfaceUpdateFunc(idx uint64) {
	interfaceUpdatedFuncMu.Lock()
	delete(interfaceUpdatedFunc, idx)
	interfaceUpdatedFuncMu.Unlock()
}

func InterfaceUpdate() {
	interfaceUpdatedFuncMu.Lock()
	for _, fn := range interfaceUpdatedFunc {
		fn()
	}
	interfaceUpdatedFuncMu.Unlock()
}
