package inbound

import (
	"syscall"

	"github.com/v2fly/v2ray-core/v5/app/proxyman/inbound/procfs"
	"github.com/v2fly/v2ray-core/v5/common/net"
)

var (
	uidDumper         UidDumper
	useProcfs         bool
	ErrNotInitialized = newError("uidDumper not initialized")
)

type UidDumper interface {
	DumpUid(ipProto int32, srcIp string, srcPort int32, destIp string, destPort int32) (int32, error)
	GetPackageName(uid int32) (string, error)
}

func SetUidDumper(newUidDumper UidDumper, newUseProcfs bool) {
	uidDumper = newUidDumper
	useProcfs = newUseProcfs
}

func GetUidDumper() (UidDumper, error) {
	if uidDumper == nil {
		return nil, ErrNotInitialized
	}
	return uidDumper, nil
}

func DumpUid(source net.Destination, destination net.Destination) (int32, error) {
	if useProcfs {
		return procfs.QuerySocketUidFromProcFs(source, destination), nil
	}
	if uidDumper == nil {
		return -1, ErrNotInitialized
	}
	var ipProto int32
	if destination.Network == net.Network_TCP {
		ipProto = syscall.IPPROTO_TCP
	} else {
		ipProto = syscall.IPPROTO_UDP
	}
	return uidDumper.DumpUid(ipProto, source.Address.IP().String(), int32(source.Port), destination.Address.IP().String(), int32(destination.Port))
}
