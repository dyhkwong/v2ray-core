package httpupgrade

import (
	"context"
	"io"
	"time"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/serial"
)

type Connection struct {
	Conn       net.Conn
	reader     io.Reader
	remoteAddr net.Addr

	shouldWait        bool
	delayedDialFinish context.Context
	finishedDial      context.CancelFunc
	dialer            delayedDialer
}

type delayedDialer func(ctx context.Context, earlyData []byte) (conn net.Conn, earlyReply io.Reader, err error)

func newConnectionWithPendingRead(conn net.Conn, remoteAddr net.Addr, earlyReplyReader io.Reader) *Connection {
	return &Connection{
		Conn:       conn,
		remoteAddr: remoteAddr,
		reader:     earlyReplyReader,
	}
}

func newConnectionWithDelayedDial(ctx context.Context, dialer delayedDialer) *Connection {
	ctx, cancel := context.WithCancel(ctx)
	return &Connection{
		shouldWait:        true,
		delayedDialFinish: ctx,
		finishedDial:      cancel,
		dialer:            dialer,
	}
}

// Read implements net.Conn.Read()
func (c *Connection) Read(b []byte) (int, error) {
	if c.shouldWait {
		<-c.delayedDialFinish.Done()
		if c.Conn == nil {
			return 0, newError("unable to read delayed dial websocket connection as it do not exist")
		}
	}

	if c.reader != nil {
		n, err := c.reader.Read(b)
		if err == io.EOF {
			c.reader = nil
			return c.Conn.Read(b)
		}
		return n, err
	}
	return c.Conn.Read(b)
}

// Write implements io.Writer.
func (c *Connection) Write(b []byte) (int, error) {
	if c.shouldWait {
		conn, earlyReply, err := c.dialer(c.delayedDialFinish, b)
		if err != nil {
			c.finishedDial()
			return 0, newError("Unable to proceed with delayed write").Base(err)
		}
		if earlyReply != nil {
			c.reader = earlyReply
		}
		c.Conn = conn
		c.remoteAddr = c.Conn.RemoteAddr()
		c.shouldWait = false
		c.finishedDial()
		return len(b), nil
	}
	return c.Conn.Write(b)
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
	var closeErrors []interface{}
	if err := c.Conn.Close(); err != nil {
		closeErrors = append(closeErrors, err)
	}
	if len(closeErrors) > 0 {
		return newError("failed to close connection").Base(newError(serial.Concat(closeErrors...)))
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
			newError("httpupgrade transport is not materialized when SetReadDeadline() is called").AtWarning().WriteToLog()
			return nil
		}
	}
	return c.Conn.SetReadDeadline(t)
}

func (c *Connection) SetWriteDeadline(t time.Time) error {
	if c.shouldWait {
		<-c.delayedDialFinish.Done()
		if c.Conn == nil {
			newError("httpupgrade transport is not materialized when SetWriteDeadline() is called").AtWarning().WriteToLog()
			return nil
		}
	}
	return c.Conn.SetWriteDeadline(t)
}
