//go:build !(android && cgo)

package localdns

import (
	"context"
	"encoding/binary"
	"io"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/v2fly/v2ray-core/v4/common/buf"
	"github.com/v2fly/v2ray-core/v4/common/net"
)

var rawQueryFunc = func(request []byte) ([]byte, error) {
	message := new(dnsmessage.Message)
	err := message.Unpack(request)
	if err != nil {
		return nil, newError("failed to parse dns request").Base(err)
	}
	if message.Response || len(message.Answers) > 0 {
		return nil, newError("failed to parse dns request: not query")
	}

	dns := dnsReadConfig()

	dialer := new(net.Dialer)

	udpAddr := &net.UDPAddr{
		IP:   dns[0].AsSlice(),
		Port: 53,
		Zone: dns[0].Zone(),
	}
	udpCtx, udpCancel := context.WithTimeout(context.Background(), time.Second*5)
	defer udpCancel()
	udpConn, err := dialer.DialContext(udpCtx, "udp", udpAddr.String())
	if err != nil {
		return nil, err
	}
	defer udpConn.Close()

	udpConn.SetDeadline(time.Now().Add(time.Second * 5))

	if _, err := udpConn.Write(request); err != nil {
		return nil, err
	}

	response := make([]byte, buf.Size)
	n, err := udpConn.Read(response)
	if err != nil {
		return nil, err
	}

	if err = message.Unpack(response[:n]); err != nil {
		return nil, newError("failed to parse dns response").Base(err)
	}

	if !message.Truncated {
		return response[:n], nil
	}

	newError("truncated, retry over TCP").AtError().WriteToLog()

	tcpAddr := &net.TCPAddr{
		IP:   dns[0].AsSlice(),
		Port: 53,
		Zone: dns[0].Zone(),
	}
	tcpCtx, tcpCancel := context.WithTimeout(context.Background(), time.Second*5)
	defer tcpCancel()
	tcpConn, err := dialer.DialContext(tcpCtx, "tcp", tcpAddr.String())
	if err != nil {
		return nil, err
	}
	defer tcpConn.Close()

	tcpConn.SetDeadline(time.Now().Add(time.Second * 5))

	reqBuf := buf.NewWithSize(2 + int32(len(request)))
	defer reqBuf.Release()
	if err := binary.Write(reqBuf, binary.BigEndian, uint16(len(request))); err != nil {
		return nil, err
	}
	if _, err := reqBuf.Write(request); err != nil {
		return nil, err
	}
	if _, err := tcpConn.Write(reqBuf.Bytes()); err != nil {
		return nil, err
	}
	var length uint16
	if err := binary.Read(tcpConn, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	response = make([]byte, length)
	n, err = io.ReadFull(tcpConn, response)
	if err != nil {
		return nil, err
	}
	if err = message.Unpack(response[:n]); err != nil {
		return nil, newError("failed to parse dns response").Base(err)
	}
	return response[:n], nil
}
