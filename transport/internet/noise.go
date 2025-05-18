package internet

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type noiseConfig struct {
	noise []byte
	delay uint64
	count uint64
}

func newNoiseConfig(config *SocketConfig_Noise) (*noiseConfig, error) {
	c := new(noiseConfig)
	var err, err2 error
	switch strings.ToLower(config.Type) {
	case "rand":
		randValue := strings.Split(config.Packet, "-")
		if len(randValue) > 2 {
			return nil, newError("Only 2 values are allowed for rand")
		}
		var lengthMin, lengthMax uint64
		if len(randValue) == 2 {
			lengthMin, err = strconv.ParseUint(randValue[0], 10, 64)
			lengthMax, err2 = strconv.ParseUint(randValue[1], 10, 64)
		}
		if len(randValue) == 1 {
			lengthMin, err = strconv.ParseUint(randValue[0], 10, 64)
			lengthMax = lengthMin
		}
		if err != nil {
			return nil, newError("invalid value for rand lengthMin").Base(err)
		}
		if err2 != nil {
			return nil, newError("invalid value for rand lengthMax").Base(err2)
		}
		if lengthMin > lengthMax {
			lengthMin, lengthMax = lengthMax, lengthMin
		}
		if lengthMin == 0 {
			return nil, newError("rand lengthMin or lengthMax cannot be 0")
		}
		c.noise, err = GenerateRandomBytes(randBetween(int64(lengthMin), int64(lengthMax)))
		if err != nil {
			return nil, err
		}
	case "str":
		c.noise = []byte(strings.TrimSpace(config.Packet))
	case "base64":
		c.noise, err = base64.RawURLEncoding.DecodeString(strings.NewReplacer("+", "-", "/", "_", "=", "").Replace(strings.TrimSpace(config.Packet)))
		if err != nil {
			return nil, newError("Invalid base64 string").Base(err)
		}
	case "hex":
		c.noise, err = hex.DecodeString(config.Packet)
		if err != nil {
			return nil, newError("Invalid hex string").Base(err)
		}
	default:
		return nil, newError("Invalid packet, only rand, str, base64, and hex are supported")
	}
	var delayMin, delayMax uint64
	if len(config.Delay) > 0 {
		d := strings.Split(strings.ToLower(config.Delay), "-")
		if len(d) > 2 {
			return nil, newError("Invalid delay value")
		}
		if len(d) == 2 {
			delayMin, err = strconv.ParseUint(d[0], 10, 64)
			delayMax, err2 = strconv.ParseUint(d[1], 10, 64)
		} else {
			delayMin, err = strconv.ParseUint(d[0], 10, 64)
			delayMax = delayMin
		}
		if err != nil {
			return nil, newError("Invalid value for delayMin").Base(err)
		}
		if err2 != nil {
			return nil, newError("Invalid value for delayMax").Base(err2)
		}
		if delayMin > delayMax {
			delayMin, delayMax = delayMax, delayMin
		}
		if delayMin == 0 {
			return nil, newError("delayMin or delayMax cannot be 0")
		}
		c.delay = uint64(randBetween(int64(delayMin), int64(delayMax)))
	}
	var countMin, countMax uint64
	if len(config.Count) > 0 {
		cnt := strings.Split(strings.ToLower(config.Count), "-")
		if len(cnt) > 2 {
			return nil, newError("Invalid count value")
		}
		if len(cnt) == 2 {
			countMin, err = strconv.ParseUint(cnt[0], 10, 64)
			countMax, err2 = strconv.ParseUint(cnt[1], 10, 64)
		} else {
			countMin, err = strconv.ParseUint(cnt[0], 10, 64)
			countMax = countMin
		}
		if err != nil {
			return nil, newError("Invalid value for countMin").Base(err)
		}
		if err2 != nil {
			return nil, newError("Invalid value for countMax").Base(err2)
		}
		if countMin > countMax {
			countMin, countMax = countMax, countMin
		}
		c.count = uint64(randBetween(int64(countMin), int64(countMax)))
		if c.count < 1 {
			c.count = 1
		}
		if c.count > 100 {
			c.count = 100
		}
	} else {
		c.count = 1
	}
	return c, nil
}

func NewNoisePacketConn(conn net.PacketConn, configs []*SocketConfig_Noise, keepalive uint64) (net.PacketConn, error) {
	newError("NOISE", configs).AtDebug().WriteToLog()
	return &noisePacketConn{
		PacketConn: conn,
		firstWrite: true,
		configs:    configs,
		keepalive:  keepalive,
	}, nil
}

type noisePacketConn struct {
	net.PacketConn
	addr       net.Addr
	firstWrite bool
	configs    []*SocketConfig_Noise
	keepalive  uint64
	stopChan   chan struct{}
	ticker     *time.Ticker
	closeOnce  *sync.Once
}

func (n *noisePacketConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	if n.firstWrite {
		n.firstWrite = false
		n.addr = addr
		for _, c := range n.configs {
			config, err := newNoiseConfig(c)
			if err != nil {
				return 0, err
			}
			for range config.count {
				_, _ = n.PacketConn.WriteTo(config.noise, addr)
				if config.delay > 0 {
					time.Sleep(time.Duration(config.delay) * time.Millisecond)
				}
			}
		}
		if n.keepalive > 0 {
			n.stopChan = make(chan struct{})
			n.ticker = time.NewTicker(time.Duration(n.keepalive) * time.Second)
			n.closeOnce = &sync.Once{}
			go n.keepNoiseAlive()
		}
	}
	return n.PacketConn.WriteTo(b, addr)
}

func (n *noisePacketConn) Close() error {
	if n.stopChan != nil {
		n.closeOnce.Do(func() {
			close(n.stopChan)
		})
	}
	return n.Close()
}

func (n *noisePacketConn) keepNoiseAlive() {
	for {
		select {
		case <-n.ticker.C:
			for _, c := range n.configs {
				config, err := newNoiseConfig(c)
				if err != nil {
					return
				}
				for range config.count {
					_, _ = n.PacketConn.WriteTo(config.noise, n.addr)
					if config.delay > 0 {
						time.Sleep(time.Duration(config.delay) * time.Millisecond)
					}
				}
			}
		case <-n.stopChan:
			n.ticker.Stop()
			return
		}
	}
}

func NewNoiseConn(conn net.Conn, configs []*SocketConfig_Noise, keepalive uint64) (net.Conn, error) {
	newError("NOISE", configs).AtDebug().WriteToLog()
	return &noiseConn{
		Conn:       conn,
		firstWrite: true,
		configs:    configs,
		keepalive:  keepalive,
	}, nil
}

type noiseConn struct {
	net.Conn
	firstWrite bool
	configs    []*SocketConfig_Noise
	keepalive  uint64
	stopChan   chan struct{}
	ticker     *time.Ticker
	closeOnce  *sync.Once
}

func (n *noiseConn) Write(b []byte) (int, error) {
	if n.firstWrite {
		n.firstWrite = false
		for _, c := range n.configs {
			config, err := newNoiseConfig(c)
			if err != nil {
				return 0, err
			}
			for range config.count {
				_, _ = n.Conn.Write(config.noise)
				if config.delay > 0 {
					time.Sleep(time.Duration(config.delay) * time.Millisecond)
				}
			}
		}
		if n.keepalive > 0 {
			n.stopChan = make(chan struct{})
			n.ticker = time.NewTicker(time.Duration(n.keepalive) * time.Second)
			n.closeOnce = &sync.Once{}
			go n.keepNoiseAlive()
		}
	}
	return n.Conn.Write(b)
}

func (n *noiseConn) Close() error {
	if n.stopChan != nil {
		n.closeOnce.Do(func() {
			close(n.stopChan)
		})
	}
	return n.Close()
}

func (n *noiseConn) keepNoiseAlive() {
	for {
		select {
		case <-n.ticker.C:
			for _, c := range n.configs {
				config, err := newNoiseConfig(c)
				if err != nil {
					return
				}
				for range config.count {
					_, _ = n.Conn.Write(config.noise)
					if config.delay > 0 {
						time.Sleep(time.Duration(config.delay) * time.Millisecond)
					}
				}
			}
		case <-n.stopChan:
			n.ticker.Stop()
			return
		}
	}
}

func GenerateRandomBytes(n int64) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}
