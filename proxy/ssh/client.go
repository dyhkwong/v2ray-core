package ssh

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/retry"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal"
	"github.com/v2fly/v2ray-core/v5/common/task"
	"github.com/v2fly/v2ray-core/v5/features/policy"
	"github.com/v2fly/v2ray-core/v5/proxy"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

func init() {
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		c := &Client{}
		return c, core.RequireFeatures(ctx, func(policyManager policy.Manager) error {
			return c.Init(config.(*Config), policyManager)
		})
	}))
}

var (
	_ proxy.Outbound  = (*Client)(nil)
	_ common.Closable = (*Client)(nil)
)

type Client struct {
	sync.Mutex
	config          *Config
	sessionPolicy   policy.Session
	server          net.Destination
	client          *ssh.Client
	auth            []ssh.AuthMethod
	hostKeyCallback ssh.HostKeyCallback
}

func (c *Client) Init(config *Config, policyManager policy.Manager) error {
	c.config = config
	c.sessionPolicy = policyManager.ForLevel(config.UserLevel)
	c.server = net.Destination{
		Network: net.Network_TCP,
		Address: config.Address.AsAddress(),
		Port:    net.Port(config.Port),
	}
	if config.User == "" {
		config.User = "root"
	}
	if config.HostKeyAlgorithms != nil && len(config.HostKeyAlgorithms) == 0 {
		config.HostKeyAlgorithms = nil
	}

	if config.PrivateKey != "" {
		var signer ssh.Signer
		var err error
		if config.Password == "" {
			signer, err = ssh.ParsePrivateKey([]byte(config.PrivateKey))
		} else {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(config.PrivateKey), []byte(config.Password))
		}
		if err != nil {
			return newError("parse private key").Base(err)
		}
		c.auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	} else if config.Password != "" {
		c.auth = []ssh.AuthMethod{ssh.Password(config.Password)}
	}

	var keys []ssh.PublicKey
	if config.PublicKey != "" {
		for str := range strings.SplitSeq(config.PublicKey, "\n") {
			str = strings.TrimSpace(str)
			if str == "" {
				continue
			}
			key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(str))
			if err != nil {
				return newError(err, "parse public key").Base(err)
			}
			keys = append(keys, key)
		}
	}
	if keys != nil {
		c.hostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			for _, pk := range keys {
				if bytes.Equal(key.Marshal(), pk.Marshal()) {
					return nil
				}
			}
			return newError("ssh host key mismatch, server send ", key.Type(), " ", base64.StdEncoding.EncodeToString(key.Marshal()))
		}
	} else {
		c.hostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			newError("please save server public key for verifying").AtWarning().WriteToLog()
			newError(key.Type(), " ", base64.StdEncoding.EncodeToString(key.Marshal())).AtWarning().WriteToLog()
			return nil
		}
	}
	return nil
}

func (c *Client) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified")
	}
	destination := outbound.Target
	if destination.Network != net.Network_TCP {
		return newError("only TCP is supported in SSH proxy")
	}

	newError("tunneling request to ", destination, " via ", c.server.NetAddr()).WriteToLog(session.ExportIDToError(ctx))

	client, err := c.connect(ctx, dialer)
	if err != nil {
		return err
	}

	dialCtx, dialCancel := context.WithDeadline(ctx, time.Now().Add(time.Second*5))
	defer dialCancel()
	conn, err := client.DialContext(dialCtx, "tcp", destination.NetAddr())
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			c.Lock()
			client.Close()
			c.client = nil
			c.Unlock()
		}
		return newError("failed to open ssh proxy connection").Base(err)
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(ctx)
	timer := signal.CancelAfterInactivity(ctx, cancel, c.sessionPolicy.Timeouts.ConnectionIdle)

	if err := task.Run(ctx, func() error {
		defer timer.SetTimeout(c.sessionPolicy.Timeouts.DownlinkOnly)
		return buf.Copy(link.Reader, buf.NewWriter(conn), buf.UpdateActivity(timer))
	}, func() error {
		defer timer.SetTimeout(c.sessionPolicy.Timeouts.UplinkOnly)
		return buf.Copy(buf.NewReader(conn), link.Writer, buf.UpdateActivity(timer))
	}); err != nil {
		return newError("connection ends").Base(err)
	}

	return nil
}

func (c *Client) connect(ctx context.Context, dialer internet.Dialer) (*ssh.Client, error) {
	c.Lock()
	if c.client != nil {
		c.Unlock()
		return c.client, nil
	}
	c.Unlock()

	newError("open connection to ", c.server).AtDebug().WriteToLog(session.ExportIDToError(ctx))
	var conn internet.Connection
	err := retry.ExponentialBackoff(5, 100).On(func() error {
		rawConn, err := dialer.Dial(ctx, c.server)
		if err != nil {
			return err
		}
		conn = rawConn
		return nil
	})
	if err != nil {
		return nil, err
	}

	clientConn, chans, reqs, err := ssh.NewClientConn(conn, c.server.Address.String(), &ssh.ClientConfig{
		User:              c.config.User,
		Auth:              c.auth,
		ClientVersion:     c.config.ClientVersion,
		HostKeyAlgorithms: c.config.HostKeyAlgorithms,
		HostKeyCallback:   c.hostKeyCallback,
		BannerCallback: func(message string) error {
			for line := range strings.SplitSeq(message, "\n") {
				newError("| ", line).AtDebug().WriteToLog(session.ExportIDToError(ctx))
			}
			return nil
		},
	})
	if err != nil {
		conn.Close()
		return nil, newError("failed to create ssh connection").Base(err)
	}

	client := ssh.NewClient(clientConn, chans, reqs)

	c.Lock()
	c.client = client
	c.Unlock()
	go func() {
		err := client.Wait()
		newError("ssh client closed").Base(err).AtDebug().WriteToLog()
		client.Close()
		c.Lock()
		c.client = nil
		c.Unlock()
	}()
	return client, nil
}

func (c *Client) Close() error {
	c.Lock()
	if c.client != nil {
		c.client.Close()
	}
	c.client = nil
	c.Unlock()
	return nil
}
