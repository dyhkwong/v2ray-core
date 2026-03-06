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
	defer interfaceUpdateCallbackMutex.Unlock()
	return interfaceUpdateCallBackList.PushBack(callback)
}

func UnRegisterInterfaceUpdateCallback(callback *list.Element) {
	interfaceUpdateCallbackMutex.Lock()
	interfaceUpdateCallBackList.Remove(callback)
	interfaceUpdateCallbackMutex.Unlock()
}

func InterfaceUpdate() {
	interfaceUpdateCallbackMutex.Lock()
	for element := interfaceUpdateCallBackList.Front(); element != nil; element = element.Next() {
		callback := element.Value.(func())
		callback()
	}
	interfaceUpdateCallbackMutex.Unlock()
}
