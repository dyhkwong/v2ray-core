//go:build !android

package localdns

import (
	"context"
	"encoding/binary"
	"io"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
)

var defaultRawQueryFunc = func(request []byte) ([]byte, error) {
	requestMsg := new(dnsmessage.Message)
	err := requestMsg.Unpack(request)
	if err != nil {
		return nil, newError("failed to parse dns request").Base(err)
	}

	dns := dnsReadConfig()

	done := make(chan any)
	var udpConn *net.UDPConn
	go func() {
		udpConn, err = net.DialUDP("udp", nil, &net.UDPAddr{
			IP:   dns[0].AsSlice(),
			Port: 53,
			Zone: dns[0].Zone(),
		})
		close(done)
	}()
	select {
	case <-time.After(time.Second * 5):
		return nil, context.DeadlineExceeded
	case <-done:
		if err != nil {
			return nil, err
		}
		defer udpConn.Close()
	}

	if _, err := udpConn.Write(request); err != nil {
		return nil, err
	}

	response := make([]byte, buf.Size)
	n, err := udpConn.Read(response)
	if err != nil {
		return nil, err
	}
	if n < 12 {
		return nil, newError("response too short")
	}
	if binary.BigEndian.Uint16(response[:2]) != requestMsg.ID {
		return nil, newError("DNS message ID mismatch")
	}
	if response[3]&0x02 < 0x02 {
		return response[:n], nil
	}

	newError("truncated, retry over TCP").AtError().WriteToLog()
	done = make(chan any)
	var tcpConn *net.TCPConn
	go func() {
		tcpConn, err = net.DialTCP("tcp", nil, &net.TCPAddr{
			IP:   dns[0].AsSlice(),
			Port: 53,
			Zone: dns[0].Zone(),
		})
		close(done)
	}()
	select {
	case <-time.After(time.Second * 5):
		return nil, context.DeadlineExceeded
	case <-done:
		if err != nil {
			return nil, err
		}
		defer tcpConn.Close()
	}

	reqBuf := buf.NewWithSize(2 + int32(len(request)))
	defer reqBuf.Release()
	binary.Write(reqBuf, binary.BigEndian, uint16(len(request)))
	reqBuf.Write(request)
	if _, err := tcpConn.Write(reqBuf.Bytes()); err != nil {
		return nil, err
	}
	var length uint16
	if err := binary.Read(tcpConn, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	response = make([]byte, length)
	if n, err = io.ReadFull(tcpConn, response); err != nil {
		return nil, err
	}
	if binary.BigEndian.Uint16(response[:2]) != requestMsg.ID {
		return nil, newError("DNS message ID mismatch")
	}
	return response[:n], nil
}
