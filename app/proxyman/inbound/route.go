package inbound

var (
	networkType string
	ssid        string
	uidDumper   UidDumper
)

func SetNetworkType(newNetworkType string) {
	if newNetworkType != networkType {
		newError("updated network type: ", newNetworkType).AtDebug().WriteToLog()
		networkType = newNetworkType
	}
}

func GetNetworkType() string {
	return networkType
}

func SetSSID(newSSID string) {
	if newSSID != ssid {
		newError("updated SSID: ", newSSID).AtDebug().WriteToLog()
		ssid = newSSID
	}
}

func GetSSID() string {
	return ssid
}

type UidDumper interface {
	DumpUid(ipProto int32, srcIP string, srcPort int32, destIP string, destPort int32) (int32, error)
	GetPackageName(uid int32) (string, error)
}

func SetUidDumper(newUidDumper UidDumper) {
	uidDumper = newUidDumper
}

func GetUidDumper() (UidDumper, error) {
	if uidDumper == nil {
		return nil, newError("uidDumper not initialized")
	}
	return uidDumper, nil
}
