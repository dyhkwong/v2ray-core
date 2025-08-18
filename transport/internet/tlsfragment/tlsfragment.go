package tlsfragment

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"net"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

func NewTLSFragmentConn(conn net.Conn, splitRecord, splitPacket bool) net.Conn {
	fragmentConn := &tlsFragmentConn{
		Conn:        conn,
		splitRecord: splitRecord,
		splitPacket: splitPacket,
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		fragmentConn.tcpConn = tcpConn
	}
	return fragmentConn
}

type tlsFragmentConn struct {
	net.Conn
	tcpConn            *net.TCPConn
	splitPacket        bool
	splitRecord        bool
	fallbackDelayMs    uint32
	firstPacketWritten bool
}

func (c *tlsFragmentConn) Write(b []byte) (int, error) {
	if c.firstPacketWritten {
		return c.Conn.Write(b)
	}
	c.firstPacketWritten = true
	serverName := indexTLSServerName(b)
	if serverName == nil {
		return c.Conn.Write(b)
	}
	if c.splitPacket && c.tcpConn != nil {
		c.tcpConn.SetNoDelay(true)
	}
	splits := strings.Split(serverName.serverName, ".")
	currentIndex := serverName.index
	if publicSuffix := publicsuffix.List.PublicSuffix(serverName.serverName); publicSuffix != "" {
		splits = splits[:len(splits)-strings.Count(serverName.serverName, ".")]
	}
	if len(splits) > 1 && splits[0] == "..." {
		currentIndex += len(splits[0]) + 1
		splits = splits[1:]
	}
	var splitIndexes []int
	for i, split := range splits {
		splitAt := rand.Intn(len(split))
		splitIndexes = append(splitIndexes, currentIndex+splitAt)
		currentIndex += len(split)
		if i != len(splits)-1 {
			currentIndex++
		}
	}
	var buffer bytes.Buffer
	for i := 0; i <= len(splitIndexes); i++ {
		var payload []byte
		if i == 0 {
			payload = b[:splitIndexes[i]]
			if c.splitRecord {
				payload = payload[recordLayerHeaderLen:]
			}
		} else if i == len(splitIndexes) {
			payload = b[splitIndexes[i-1]:]
		} else {
			payload = b[splitIndexes[i-1]:splitIndexes[i]]
		}
		if c.splitRecord {
			if c.splitPacket {
				buffer.Reset()
			}
			payloadLen := uint16(len(payload))
			buffer.Write(b[:3])
			binary.Write(&buffer, binary.BigEndian, payloadLen)
			buffer.Write(payload)
			if c.splitPacket {
				payload = buffer.Bytes()
			}
		}
		if c.splitPacket {
			if c.tcpConn != nil && i != len(splitIndexes) {
				if err := writeAndWaitAck(c.tcpConn, payload, time.Millisecond*time.Duration(20+rand.Intn(20))); err != nil {
					return 0, err
				}
			} else {
				if _, err := c.Conn.Write(payload); err != nil {
					return 0, err
				}
				if i != len(splitIndexes) {
					time.Sleep(time.Millisecond * time.Duration(20+rand.Intn(20)))
				}
			}
		}
	}
	if c.splitRecord && !c.splitPacket {
		if _, err := c.Conn.Write(buffer.Bytes()); err != nil {
			return 0, err
		}
	}
	return len(b), nil
}

const (
	recordLayerHeaderLen    int    = 5
	handshakeHeaderLen      int    = 6
	randomDataLen           int    = 32
	sessionIDHeaderLen      int    = 1
	cipherSuiteHeaderLen    int    = 2
	compressMethodHeaderLen int    = 1
	extensionsHeaderLen     int    = 2
	extensionHeaderLen      int    = 4
	sniExtensionHeaderLen   int    = 5
	contentType             uint8  = 22
	handshakeType           uint8  = 1
	sniExtensionType        uint16 = 0
	sniNameDNSHostnameType  uint8  = 0
	tlsVersionBitmask       uint16 = 0xFFFC
	tls13                   uint16 = 0x0304
)

type serverName struct {
	index      int
	length     int
	serverName string
}

func indexTLSServerName(payload []byte) *serverName {
	if len(payload) < recordLayerHeaderLen || payload[0] != contentType {
		return nil
	}
	segmentLen := binary.BigEndian.Uint16(payload[3:5])
	if len(payload) < recordLayerHeaderLen+int(segmentLen) {
		return nil
	}
	serverName := indexTLSServerNameFromHandshake(payload[recordLayerHeaderLen:])
	if serverName == nil {
		return nil
	}
	serverName.index += recordLayerHeaderLen
	return serverName
}

func indexTLSServerNameFromHandshake(handshake []byte) *serverName {
	if len(handshake) < handshakeHeaderLen+randomDataLen+sessionIDHeaderLen {
		return nil
	}
	if handshake[0] != handshakeType {
		return nil
	}
	handshakeLen := uint32(handshake[1])<<16 | uint32(handshake[2])<<8 | uint32(handshake[3])
	if len(handshake[4:]) != int(handshakeLen) {
		return nil
	}
	tlsVersion := uint16(handshake[4])<<8 | uint16(handshake[5])
	if tlsVersion&tlsVersionBitmask != 0x0300 && tlsVersion != tls13 {
		return nil
	}
	sessionIDLen := handshake[38]
	currentIndex := handshakeHeaderLen + randomDataLen + sessionIDHeaderLen + int(sessionIDLen)
	if len(handshake) < currentIndex {
		return nil
	}
	cipherSuites := handshake[currentIndex:]
	if len(cipherSuites) < cipherSuiteHeaderLen {
		return nil
	}
	csLen := uint16(cipherSuites[0])<<8 | uint16(cipherSuites[1])
	if len(cipherSuites) < cipherSuiteHeaderLen+int(csLen)+compressMethodHeaderLen {
		return nil
	}
	compressMethodLen := uint16(cipherSuites[cipherSuiteHeaderLen+int(csLen)])
	currentIndex += cipherSuiteHeaderLen + int(csLen) + compressMethodHeaderLen + int(compressMethodLen)
	if len(handshake) < currentIndex {
		return nil
	}
	serverName := indexTLSServerNameFromExtensions(handshake[currentIndex:])
	if serverName == nil {
		return nil
	}
	serverName.index += currentIndex
	return serverName
}

func indexTLSServerNameFromExtensions(exs []byte) *serverName {
	if len(exs) == 0 {
		return nil
	}
	if len(exs) < extensionsHeaderLen {
		return nil
	}
	exsLen := uint16(exs[0])<<8 | uint16(exs[1])
	exs = exs[extensionsHeaderLen:]
	if len(exs) < int(exsLen) {
		return nil
	}
	for currentIndex := extensionsHeaderLen; len(exs) > 0; {
		if len(exs) < extensionHeaderLen {
			return nil
		}
		exType := uint16(exs[0])<<8 | uint16(exs[1])
		exLen := uint16(exs[2])<<8 | uint16(exs[3])
		if len(exs) < extensionHeaderLen+int(exLen) {
			return nil
		}
		sex := exs[extensionHeaderLen : extensionHeaderLen+int(exLen)]

		switch exType {
		case sniExtensionType:
			if len(sex) < sniExtensionHeaderLen {
				return nil
			}
			sniType := sex[2]
			if sniType != sniNameDNSHostnameType {
				return nil
			}
			sniLen := uint16(sex[3])<<8 | uint16(sex[4])
			sex = sex[sniExtensionHeaderLen:]

			return &serverName{
				index:      currentIndex + extensionHeaderLen + sniExtensionHeaderLen,
				length:     int(sniLen),
				serverName: string(sex),
			}
		}
		exs = exs[4+exLen:]
		currentIndex += 4 + int(exLen)
	}
	return nil
}
