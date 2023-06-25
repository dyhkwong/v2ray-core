package websocket

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/gorilla/websocket"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/errors"
	"github.com/v2fly/v2ray-core/v5/common/serial"
)

var _ buf.Writer = (*Connection)(nil)

// Connection is a wrapper for net.Conn over WebSocket Connection.
type Connection struct {
	Conn       *websocket.Conn
	reader     io.Reader
	remoteAddr net.Addr

	shouldWait        bool
	delayedDialFinish context.Context
	finishedDial      context.CancelFunc
	dialer            DelayedDialer
}

type DelayedDialer interface {
	Dial(ctx context.Context, earlyData []byte) (*websocket.Conn, error)
}

func newConnection(conn *websocket.Conn, remoteAddr net.Addr) *Connection {
	return &Connection{
		Conn:       conn,
		remoteAddr: remoteAddr,
	}
}

func newConnectionWithEarlyData(conn *websocket.Conn, remoteAddr net.Addr, earlyData io.Reader) *Connection {
	return &Connection{
		Conn:       conn,
		remoteAddr: remoteAddr,
		reader:     earlyData,
	}
}

func newConnectionWithDelayedDial(ctx context.Context, dialer DelayedDialer) *Connection {
	delayedDialContext, cancelFunc := context.WithCancel(ctx)
	return &Connection{
		shouldWait:        true,
		delayedDialFinish: delayedDialContext,
		finishedDial:      cancelFunc,
		dialer:            dialer,
	}
}

func newRelayedConnectionWithDelayedDial(ctx context.Context, dialer DelayedDialerForwarded) *connectionForwarder {
	delayedDialContext, cancelFunc := context.WithCancel(ctx)
	return &connectionForwarder{
		shouldWait:        true,
		delayedDialFinish: delayedDialContext,
		finishedDial:      cancelFunc,
		dialer:            dialer,
	}
}

func newRelayedConnection(conn io.ReadWriteCloser) *connectionForwarder {
	return &connectionForwarder{
		ReadWriteCloser: conn,
		shouldWait:      false,
	}
}

// Read implements net.Conn.Read()
func (c *Connection) Read(b []byte) (int, error) {
	for {
		reader, err := c.getReader()
		if err != nil {
			return 0, err
		}

		nBytes, err := reader.Read(b)
		if errors.Cause(err) == io.EOF {
			c.reader = nil
			continue
		}
		return nBytes, err
	}
}

func (c *Connection) getReader() (io.Reader, error) {
	if c.shouldWait {
		<-c.delayedDialFinish.Done()
		if c.Conn == nil {
			return nil, newError("unable to read delayed dial websocket connection as it do not exist")
		}
	}
	if c.reader != nil {
		return c.reader, nil
	}

	_, reader, err := c.Conn.NextReader()
	if err != nil {
		return nil, err
	}
	c.reader = reader
	return reader, nil
}

// Write implements io.Writer.
func (c *Connection) Write(b []byte) (int, error) {
	if c.shouldWait {
		conn, err := c.dialer.Dial(c.delayedDialFinish, b)
		if err != nil {
			c.finishedDial()
			return 0, newError("Unable to proceed with delayed write").Base(err)
		}
		c.Conn = conn
		c.remoteAddr = c.Conn.RemoteAddr()
		c.shouldWait = false
		c.finishedDial()
		return len(b), nil
	}
	if err := c.Conn.WriteMessage(websocket.BinaryMessage, b); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *Connection) WriteMultiBuffer(mb buf.MultiBuffer) error {
	mb = buf.Compact(mb)
	mb, err := buf.WriteMultiBuffer(c, mb)
	buf.ReleaseMulti(mb)
	return err
}

func (c *Connection) Close() error {
	if c.shouldWait {
		select {
		case <-c.delayedDialFinish.Done():
		default:
			c.finishedDial()
		}
		if c.Conn == nil {
			return newError("unable to close delayed dial websocket connection as it do not exist")
		}
	}
	var errors []interface{}
	if err := c.Conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second*5)); err != nil {
		errors = append(errors, err)
	}
	if err := c.Conn.Close(); err != nil {
		errors = append(errors, err)
	}
	if len(errors) > 0 {
		return newError("failed to close connection").Base(newError(serial.Concat(errors...)))
	}
	return nil
}

func (c *Connection) LocalAddr() net.Addr {
	if c.shouldWait {
		<-c.delayedDialFinish.Done()
		if c.Conn == nil {
			newError("websocket transport is not materialized when LocalAddr() is called").AtWarning().WriteToLog()
			return &net.UnixAddr{
				Name: "@placeholder",
				Net:  "unix",
			}
		}
	}
	return c.Conn.LocalAddr()
}

func (c *Connection) RemoteAddr() net.Addr {
	if c.shouldWait {
		<-c.delayedDialFinish.Done()
		if c.Conn == nil {
			newError("websocket transport is not materialized when RemoteAddr() is called").AtWarning().WriteToLog()
			return &net.UnixAddr{
				Name: "@placeholder",
				Net:  "unix",
			}
		}
	}
	return c.remoteAddr
}

func (c *Connection) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

func (c *Connection) SetReadDeadline(t time.Time) error {
	if c.shouldWait {
		<-c.delayedDialFinish.Done()
		if c.Conn == nil {
			newError("websocket transport is not materialized when SetReadDeadline() is called").AtWarning().WriteToLog()
			return nil
		}
	}
	return c.Conn.SetReadDeadline(t)
}

func (c *Connection) SetWriteDeadline(t time.Time) error {
	if c.shouldWait {
		<-c.delayedDialFinish.Done()
		if c.Conn == nil {
			newError("websocket transport is not materialized when SetWriteDeadline() is called").AtWarning().WriteToLog()
			return nil
		}
	}
	return c.Conn.SetWriteDeadline(t)
}
