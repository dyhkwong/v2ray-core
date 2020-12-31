//go:build !linux
// +build !linux

package dokodemo

import (
	"fmt"
	"net"
)

func fakeUDP(addr *net.UDPAddr, mark int) (net.PacketConn, error) {
	return nil, &net.OpError{Op: "fake", Err: fmt.Errorf("!linux")}
}
