package internet

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type fragmentConfig struct {
	packetsFrom uint64
	packetsTo   uint64
	lengthMin   uint64
	lengthMax   uint64
	intervalMin uint64
	intervalMax uint64
	fakeHost    bool
	host1Header string
	host1Domain string
	host2Header string
	host2Domain string
}

func newFragmentConfig(config *SocketConfig_Fragment) (*fragmentConfig, error) {
	c := new(fragmentConfig)
	var err, err2 error
	switch strings.ToLower(config.Packets) {
	case "tlshello":
		// TLS Hello Fragmentation (into multiple handshake messages)
		c.packetsFrom = 0
		c.packetsTo = 1
	case "fakehost":
		// fake host header with no fragmentation
		c.packetsFrom = 1
		c.packetsTo = 1
		c.fakeHost = true
	case "":
		// TCP Segmentation (all packets)
		c.packetsFrom = 0
		c.packetsTo = 0
	default:
		// TCP Segmentation (range)
		packetsFromTo := strings.Split(config.Packets, "-")
		if len(packetsFromTo) == 2 {
			c.packetsFrom, err = strconv.ParseUint(packetsFromTo[0], 10, 64)
			c.packetsTo, err2 = strconv.ParseUint(packetsFromTo[1], 10, 64)
		} else {
			c.packetsFrom, err = strconv.ParseUint(packetsFromTo[0], 10, 64)
			c.packetsTo = c.packetsFrom
		}
		if err != nil {
			return nil, newError("Invalid packetsFrom").Base(err)
		}
		if err2 != nil {
			return nil, newError("Invalid packetsTo").Base(err2)
		}
		if c.packetsFrom > c.packetsTo {
			c.packetsFrom, c.packetsTo = c.packetsTo, c.packetsFrom
		}
		if c.packetsFrom == 0 {
			return nil, newError("packetsFrom can't be 0")
		}
	}

	if len(config.Length) == 0 {
		return nil, newError("length can't be empty")
	}
	lengthMinMax := strings.Split(config.Length, "-")
	if len(lengthMinMax) == 2 {
		c.lengthMin, err = strconv.ParseUint(lengthMinMax[0], 10, 64)
		c.lengthMax, err2 = strconv.ParseUint(lengthMinMax[1], 10, 64)
	} else {
		c.lengthMin, err = strconv.ParseUint(lengthMinMax[0], 10, 64)
		c.lengthMax = c.lengthMin
	}
	if err != nil {
		return nil, newError("Invalid lengthMin").Base(err)
	}
	if err2 != nil {
		return nil, newError("Invalid lengthMax").Base(err2)
	}
	if c.lengthMin > c.lengthMax {
		c.lengthMin, c.lengthMax = c.lengthMax, c.lengthMin
	}
	if c.lengthMin == 0 {
		return nil, newError("lengthMin can't be 0")
	}

	if len(config.Interval) == 0 {
		return nil, newError("interval can't be empty")
	}
	intervalMinMax := strings.Split(config.Interval, "-")
	if len(intervalMinMax) == 2 {
		c.intervalMin, err = strconv.ParseUint(intervalMinMax[0], 10, 64)
		c.intervalMax, err2 = strconv.ParseUint(intervalMinMax[1], 10, 64)
	} else {
		c.intervalMin, err = strconv.ParseUint(intervalMinMax[0], 10, 64)
		c.intervalMax = c.intervalMin
	}
	if err != nil {
		return nil, newError("Invalid intervalMin").Base(err)
	}
	if err2 != nil {
		return nil, newError("Invalid intervalMax").Base(err2)
	}
	if c.intervalMin > c.intervalMax {
		c.intervalMin, c.intervalMax = c.intervalMax, c.intervalMin
	}

	if len(config.Host1Header) == 0 {
		c.host1Header = "Host : "
	} else {
		c.host1Header = config.Host1Header
	}
	if len(config.Host1Domain) == 0 {
		c.host1Domain = "cloudflare.com"
	} else {
		c.host1Domain = config.Host1Domain
	}
	if len(config.Host2Header) == 0 {
		c.host2Header = "Host:   "
	} else {
		c.host2Header = config.Host2Header
	}
	if len(config.Host2Domain) == 0 {
		c.host2Domain = "cloudflare.com"
	} else {
		c.host2Domain = config.Host2Domain
	}

	newError("FRAGMENT",
		c.packetsFrom, c.packetsTo,
		c.lengthMin, c.lengthMax,
		c.intervalMin, c.intervalMax).AtDebug().WriteToLog()
	return c, nil
}

func NewFragmentConn(conn net.Conn, cfg *SocketConfig_Fragment) (net.Conn, error) {
	config, err := newFragmentConfig(cfg)
	if err != nil {
		return nil, err
	}
	fragmentConn := &fragmentConn{
		Conn:   conn,
		config: config,
	}

	return fragmentConn, nil
}

type fragmentConn struct {
	net.Conn
	count  uint64
	config *fragmentConfig
}

func (f *fragmentConn) Write(b []byte) (int, error) {
	f.count++

	if f.config.fakeHost {
		if f.count == 1 {
			h1_header := f.config.host1Header
			h1_domain := f.config.host1Domain
			h2_header := f.config.host2Header
			h2_domain := f.config.host2Domain

			// find the old host case-insensitive
			re := regexp.MustCompile("(?i)(\r\nHost:.*\r\n)")
			firstMatch := re.FindSubmatch(b)
			var new_b []byte
			if len(firstMatch) > 1 {
				old_h := firstMatch[1]
				new_h := []byte("\r\n" + h1_header + h1_domain + string(old_h) + h2_header + h2_domain + "\r\n")
				new_b = bytes.Replace(b, old_h, new_h, 1)
			} else {
				new_b = b
			}
			return f.Conn.Write(new_b)

		} else {
			return f.Conn.Write(b)
		}
	}

	if f.config.packetsFrom == 0 && f.config.packetsTo == 1 {
		if f.count != 1 || len(b) <= 5 || b[0] != 22 {
			return f.Conn.Write(b)
		}
		recordLen := 5 + ((int(b[3]) << 8) | int(b[4]))
		if len(b) < recordLen { // maybe already fragmented somehow
			return f.Conn.Write(b)
		}
		data := b[5:recordLen]
		buf := make([]byte, 1024)
		queue := make([]byte, 2048)
		n_queue := int(randBetween(int64(2), int64(4)))
		L_queue := 0
		c_queue := 0
		for from := 0; ; {
			to := from + int(randBetween(int64(f.config.lengthMin), int64(f.config.lengthMax)))
			if to > len(data) {
				to = len(data)
			}
			copy(buf[:3], b)
			copy(buf[5:], data[from:to])
			l := to - from
			from = to
			buf[3] = byte(l >> 8)
			buf[4] = byte(l)
			if c_queue < n_queue {
				if l > 0 {
					copy(queue[L_queue:], buf[:5+l])
					L_queue = L_queue + 5 + l
				}
				c_queue = c_queue + 1
			} else {
				if l > 0 {
					copy(queue[L_queue:], buf[:5+l])
					L_queue = L_queue + 5 + l
				}
				if L_queue > 0 {
					_, err := f.Conn.Write(queue[:L_queue])
					time.Sleep(time.Duration(randBetween(int64(f.config.intervalMin), int64(f.config.intervalMax))) * time.Millisecond)
					if err != nil {
						return 0, err
					}
				}
				L_queue = 0
				c_queue = 0
			}
			if from == len(data) {
				if L_queue > 0 {
					_, err := f.Conn.Write(queue[:L_queue])
					time.Sleep(time.Duration(randBetween(int64(f.config.intervalMin), int64(f.config.intervalMax))) * time.Millisecond)
					L_queue = 0
					c_queue = 0
					if err != nil {
						return 0, err
					}
				}
				if len(b) > recordLen {
					n, err := f.Conn.Write(b[recordLen:])
					if err != nil {
						return recordLen + n, err
					}
				}
				return len(b), nil
			}
		}
	}

	if f.config.packetsFrom != 0 && (f.count < f.config.packetsFrom || f.count > f.config.packetsTo) {
		return f.Conn.Write(b)
	}
	for from := 0; ; {
		to := from + int(randBetween(int64(f.config.lengthMin), int64(f.config.lengthMax)))
		if to > len(b) {
			to = len(b)
		}
		n, err := f.Conn.Write(b[from:to])
		from += n
		time.Sleep(time.Duration(randBetween(int64(f.config.intervalMin), int64(f.config.intervalMax))) * time.Millisecond)
		if err != nil {
			return from, err
		}
		if from >= len(b) {
			return from, nil
		}
	}
}

func randBetween(left int64, right int64) int64 {
	if left == right {
		return left
	}
	bigInt, _ := rand.Int(rand.Reader, big.NewInt(right-left+1))
	return left + bigInt.Int64()
}
