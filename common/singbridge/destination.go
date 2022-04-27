package singbridge

import (
	"github.com/sagernet/sing/common/metadata"

	"github.com/v2fly/v2ray-core/v5/common/net"
)

func ToDestination(socksaddr metadata.Socksaddr, network net.Network) net.Destination {
	if socksaddr.IsIP() {
		return net.Destination{
			Network: network,
			Address: net.IPAddress(socksaddr.Addr.AsSlice()),
			Port:    net.Port(socksaddr.Port),
		}
	} else {
		return net.Destination{
			Network: network,
			Address: net.DomainAddress(socksaddr.Fqdn),
			Port:    net.Port(socksaddr.Port),
		}
	}
}

func ToSocksAddr(destination net.Destination) metadata.Socksaddr {
	var addr metadata.Socksaddr
	switch destination.Address.Family() {
	case net.AddressFamilyDomain:
		addr.Fqdn = destination.Address.Domain()
	default:
		addr.Addr = metadata.AddrFromIP(destination.Address.IP())
	}
	addr.Port = uint16(destination.Port)
	return addr
}
