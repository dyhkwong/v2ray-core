//go:build !(android && cgo)

package localdns

import (
	"context"
	"encoding/binary"
	"io"
	"time"

	"github.com/miekg/dns"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
)

var rawQueryFunc = func(request []byte) ([]byte, error) {
	message := new(dns.Msg)
	err := message.Unpack(request)
	if err != nil {
		return nil, newError("failed to parse dns request").Base(err)
	}
	if message.Response || len(message.Answer) > 0 {
		return nil, newError("failed to parse dns request: not query")
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
	var n int
	n, err = udpConn.Read(response)
	if err != nil {
		return nil, err
	}
	err = message.Unpack(response[:n])
	if err != nil {
		return nil, newError("failed to parse dns response").Base(err)
	}

	if !message.Truncated {
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
	err = message.Unpack(response[:n])
	if err != nil {
		return nil, newError("failed to parse dns response").Base(err)
	}
	return response[:n], nil
}
