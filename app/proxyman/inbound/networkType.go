package inbound

var (
	networkType string
	ssid        string
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
