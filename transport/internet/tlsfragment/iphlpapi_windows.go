package tlsfragment

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	TcpTableBasicListener uint32 = iota
	TcpTableBasicConnections
	TcpTableBasicAll
	TcpTableOwnerPidListener
	TcpTableOwnerPidConnections
	TcpTableOwnerPidAll
	TcpTableOwnerModuleListener
	TcpTableOwnerModuleConnections
	TcpTableOwnerModuleAll
)

const (
	TcpConnectionEstatsSynOpts uint32 = iota
	TcpConnectionEstatsData
	TcpConnectionEstatsSndCong
	TcpConnectionEstatsPath
	TcpConnectionEstatsSendBuff
	TcpConnectionEstatsRec
	TcpConnectionEstatsObsRec
	TcpConnectionEstatsBandwidth
	TcpConnectionEstatsFineRtt
	TcpConnectionEstatsMaximum
)

type MibTcpTable struct {
	DwNumEntries uint32
	Table        [1]MibTcpRow
}

type MibTcpRow struct {
	DwState      uint32
	DwLocalAddr  uint32
	DwLocalPort  uint32
	DwRemoteAddr uint32
	DwRemotePort uint32
}

type MibTcp6Table struct {
	DwNumEntries uint32
	Table        [1]MibTcp6Row
}

type MibTcp6Row struct {
	State         uint32
	LocalAddr     [16]byte
	LocalScopeId  uint32
	LocalPort     uint32
	RemoteAddr    [16]byte
	RemoteScopeId uint32
	RemotePort    uint32
}

type TcpEstatsSendBufferRodV0 struct {
	CurRetxQueue uint64
	MaxRetxQueue uint64
	CurAppWQueue uint64
	MaxAppWQueue uint64
}

type TcpEstatsSendBuffRwV0 struct {
	EnableCollection bool
}

const (
	offsetOfMibTcpTable            = unsafe.Offsetof(MibTcpTable{}.Table)
	offsetOfMibTcp6Table           = unsafe.Offsetof(MibTcp6Table{}.Table)
	sizeOfTcpEstatsSendBuffRwV0    = unsafe.Sizeof(TcpEstatsSendBuffRwV0{})
	sizeOfTcpEstatsSendBufferRodV0 = unsafe.Sizeof(TcpEstatsSendBufferRodV0{})
)

func GetTcpTable() ([]MibTcpRow, error) {
	var size uint32
	err := getTcpTable(nil, &size, false)
	if !errors.Is(err, windows.ERROR_INSUFFICIENT_BUFFER) {
		return nil, err
	}
	for {
		table := make([]byte, size)
		err = getTcpTable(&table[0], &size, false)
		if err != nil {
			if errors.Is(err, windows.ERROR_INSUFFICIENT_BUFFER) {
				continue
			}
			return nil, err
		}
		dwNumEntries := int(*(*uint32)(unsafe.Pointer(&table[0])))
		return unsafe.Slice((*MibTcpRow)(unsafe.Pointer(&table[offsetOfMibTcpTable])), dwNumEntries), nil
	}
}

func GetTcp6Table() ([]MibTcp6Row, error) {
	var size uint32
	err := getTcp6Table(nil, &size, false)
	if !errors.Is(err, windows.ERROR_INSUFFICIENT_BUFFER) {
		return nil, err
	}
	for {
		table := make([]byte, size)
		err = getTcp6Table(&table[0], &size, false)
		if err != nil {
			if errors.Is(err, windows.ERROR_INSUFFICIENT_BUFFER) {
				continue
			}
			return nil, err
		}
		dwNumEntries := int(*(*uint32)(unsafe.Pointer(&table[0])))
		return unsafe.Slice((*MibTcp6Row)(unsafe.Pointer(&table[offsetOfMibTcp6Table])), dwNumEntries), nil
	}
}

func GetPerTcpConnectionEStatsSendBuffer(row *MibTcpRow) (*TcpEstatsSendBufferRodV0, error) {
	var rod TcpEstatsSendBufferRodV0
	err := getPerTcpConnectionEStats(row,
		TcpConnectionEstatsSendBuff,
		0,
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&rod)),
		0,
		uint64(sizeOfTcpEstatsSendBufferRodV0),
	)
	if err != nil {
		return nil, err
	}
	return &rod, nil
}

func GetPerTcp6ConnectionEStatsSendBuffer(row *MibTcp6Row) (*TcpEstatsSendBufferRodV0, error) {
	var rod TcpEstatsSendBufferRodV0
	err := getPerTcp6ConnectionEStats(row,
		TcpConnectionEstatsSendBuff,
		0,
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&rod)),
		0,
		uint64(sizeOfTcpEstatsSendBufferRodV0),
	)
	if err != nil {
		return nil, err
	}
	return &rod, nil
}

func SetPerTcpConnectionEStatsSendBuffer(row *MibTcpRow, rw *TcpEstatsSendBuffRwV0) error {
	return setPerTcpConnectionEStats(row, TcpConnectionEstatsSendBuff, uintptr(unsafe.Pointer(&rw)), 0, uint64(sizeOfTcpEstatsSendBuffRwV0), 0)
}

func SetPerTcp6ConnectionEStatsSendBuffer(row *MibTcp6Row, rw *TcpEstatsSendBuffRwV0) error {
	return setPerTcp6ConnectionEStats(row, TcpConnectionEstatsSendBuff, uintptr(unsafe.Pointer(&rw)), 0, uint64(sizeOfTcpEstatsSendBuffRwV0), 0)
}
