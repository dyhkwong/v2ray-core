//go:build !linux && !confonly
// +build !linux,!confonly

package dokodemo

import (
	"fmt"
	"net"
)

func DialUDP(addr *net.UDPAddr, mark int) (net.PacketConn, error) {
	return nil, &net.OpError{Op: "tproxy", Err: fmt.Errorf("!linux")}
}
