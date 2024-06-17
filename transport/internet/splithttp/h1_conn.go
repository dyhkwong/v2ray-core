package splithttp

import (
	"bufio"
	"net"
)

type H1Conn struct {
	net.Conn
	UnreadedResponsesCount int
	RespBufReader          *bufio.Reader
}

func NewH1Conn(conn net.Conn) *H1Conn {
	return &H1Conn{
		Conn:          conn,
		RespBufReader: bufio.NewReader(conn),
	}
}
