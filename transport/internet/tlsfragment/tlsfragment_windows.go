package tlsfragment

import (
	"encoding/binary"
	"errors"
	"net"
	"net/netip"
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func writeAndWaitAck(conn *net.TCPConn, payload []byte, fallbackDelay time.Duration) error {
	start := time.Now()
	if err := writeAndWaitAckInternal(conn, payload); err != nil {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			if _, err := conn.Write(payload); err != nil {
				return err
			}
			time.Sleep(fallbackDelay)
			return nil
		}
		return err
	}
	if time.Since(start) <= 20*time.Millisecond {
		time.Sleep(fallbackDelay)
	}
	return nil
}

func writeAndWaitAckInternal(conn *net.TCPConn, payload []byte) error {
	var source, destination netip.AddrPort
	if tcpAddr, ok := conn.LocalAddr().(*net.TCPAddr); ok {
		source = tcpAddr.AddrPort()
	} else {
		return os.ErrInvalid
	}
	if tcpAddr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		destination = tcpAddr.AddrPort()
	} else {
		return os.ErrInvalid
	}
	if source.Addr().Is4() {
		tcpTable, err := GetTcpTable()
		if err != nil {
			return err
		}
		var tcpRow *MibTcpRow
		for _, row := range tcpTable {
			if source == netip.AddrPortFrom(netip.AddrFrom4(*(*[4]byte)(unsafe.Pointer(&row.DwLocalAddr))), binary.BigEndian.Uint16((*[4]byte)(unsafe.Pointer(&row.DwLocalPort))[:])) ||
				destination == netip.AddrPortFrom(netip.AddrFrom4(*(*[4]byte)(unsafe.Pointer(&row.DwRemoteAddr))), binary.BigEndian.Uint16((*[4]byte)(unsafe.Pointer(&row.DwRemotePort))[:])) {
				tcpRow = &row
				break
			}
		}
		if tcpRow == nil {
			return errors.New("row not found for: " + source.String())
		}
		if err := SetPerTcpConnectionEStatsSendBuffer(tcpRow, &TcpEstatsSendBuffRwV0{
			EnableCollection: true,
		}); err != nil {
			return os.NewSyscallError("SetPerTcpConnectionEStatsSendBufferV0", err)
		}
		defer SetPerTcpConnectionEStatsSendBuffer(tcpRow, &TcpEstatsSendBuffRwV0{
			EnableCollection: false,
		})
		if _, err := conn.Write(payload); err != nil {
			return err
		}
		for {
			eStatsSendBuffer, err := GetPerTcpConnectionEStatsSendBuffer(tcpRow)
			if err != nil {
				return err
			}
			if eStatsSendBuffer.CurRetxQueue == 0 {
				return nil
			}
			time.Sleep(10 * time.Millisecond)
		}
	} else {
		tcpTable, err := GetTcp6Table()
		if err != nil {
			return err
		}
		var tcpRow *MibTcp6Row
		for _, row := range tcpTable {
			if source == netip.AddrPortFrom(netip.AddrFrom16(row.LocalAddr), binary.BigEndian.Uint16((*[4]byte)(unsafe.Pointer(&row.LocalPort))[:])) ||
				destination == netip.AddrPortFrom(netip.AddrFrom16(row.RemoteAddr), binary.BigEndian.Uint16((*[4]byte)(unsafe.Pointer(&row.RemotePort))[:])) {
				tcpRow = &row
				break
			}
		}
		if tcpRow == nil {
			return errors.New("row not found for: " + source.String())
		}
		if err := SetPerTcp6ConnectionEStatsSendBuffer(tcpRow, &TcpEstatsSendBuffRwV0{
			EnableCollection: true,
		}); err != nil {
			return os.NewSyscallError("SetPerTcpConnectionEStatsSendBufferV0", err)
		}
		defer SetPerTcp6ConnectionEStatsSendBuffer(tcpRow, &TcpEstatsSendBuffRwV0{
			EnableCollection: false,
		})
		if _, err := conn.Write(payload); err != nil {
			return err
		}
		for {
			eStatsSendBuffer, err := GetPerTcp6ConnectionEStatsSendBuffer(tcpRow)
			if err != nil {
				return err
			}
			if eStatsSendBuffer.CurRetxQueue == 0 {
				return nil
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}
