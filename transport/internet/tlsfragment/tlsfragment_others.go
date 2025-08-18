//go:build !(linux || darwin || windows)

package tlsfragment

import (
	"net"
	"time"
)

func writeAndWaitAck(conn *net.TCPConn, payload []byte, fallbackDelay time.Duration) error {
	if _, err := conn.Write(payload); err != nil {
		return err
	}
	time.Sleep(fallbackDelay)
	return nil
}
