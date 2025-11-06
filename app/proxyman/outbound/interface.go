package outbound

import (
	"container/list"
	"sync"
)

var (
	interfaceUpdateCallbackMutex sync.Mutex
	interfaceUpdateCallBackList  list.List
)

func RegisterInterfaceUpdateCallback(callback func()) *list.Element {
	interfaceUpdateCallbackMutex.Lock()
	elem := interfaceUpdateCallBackList.PushBack(callback)
	interfaceUpdateCallbackMutex.Unlock()
	return elem
}

func UnRegisterInterfaceUpdateCallback(elem *list.Element) {
	interfaceUpdateCallbackMutex.Lock()
	interfaceUpdateCallBackList.Remove(elem)
	interfaceUpdateCallbackMutex.Unlock()
}

func InterfaceUpdate() {
	interfaceUpdateCallbackMutex.Lock()
	for elem := interfaceUpdateCallBackList.Front(); elem != nil; elem = elem.Next() {
		callback := elem.Value.(func())
		callback()
	}
	interfaceUpdateCallbackMutex.Unlock()
}
