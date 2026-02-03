package tls

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

func ApplyECH(c *Config, config *tls.Config) error {
	var echConfig []byte
	var err error
	var domain string

	switch {
	case len(c.EchConfig) > 0:
		echConfig = c.EchConfig
	case len(c.Ech_DOHserver) > 0:
		// ECH config > DOH lookup
		if len(c.EchQueryDomain) == 0 {
			domain = config.ServerName
		} else {
			domain = c.EchQueryDomain
		}
		if len(domain) == 0 {
			return newError("Using DOH for ECH needs SNI")
		}
		addr := net.ParseAddress(domain)
		if !addr.Family().IsDomain() {
			return newError("Using DOH for ECH needs SNI")
		}
		echConfig, err = QueryRecord(addr.Domain(), c.Ech_DOHserver)
		if err != nil {
			return err
		}
		if len(echConfig) == 0 {
			return newError("no ech record found")
		}
	case len(c.EchConfigList) > 0:
		if net.ParseAddress(config.ServerName).Family().IsDomain() {
			domain = config.ServerName
		}
		if strings.Contains(c.EchConfigList, "://") {
			var server string
			parts := strings.Split(c.EchConfigList, "+")
			switch {
			case len(parts) == 2:
				domain = parts[0]
				server = parts[1]
			case len(parts) == 1:
				server = parts[0]
			default:
				return newError("Invalid ECH DNS server format: ", c.EchConfigList)
			}
			if len(domain) == 0 {
				return newError("Using DNS for ECH Config needs serverName or use Server format example.com+https://1.1.1.1/dns-query")
			}
			echConfig, err = QueryRecord(domain, server)
			if err != nil {
				return newError("Failed to query ECH DNS record for domain: ", domain, " at server: ", server).Base(err)
			}
		} else {
			echConfig, err = base64.StdEncoding.DecodeString(c.EchConfigList)
			if err != nil {
				return newError("Failed to unmarshal echConfigList: ", err)
			}
		}
	default:
		panic("ech error")
	}

	config.EncryptedClientHelloConfigList = echConfig
	return nil
}

type record struct {
	record []byte
	expire time.Time
}

var (
	dnsCache = make(map[string]record)
	mutex    sync.RWMutex
)

func QueryRecord(domain string, server string) ([]byte, error) {
	mutex.Lock()
	rec, found := dnsCache[domain]
	if found {
		if rec.expire.After(time.Now()) {
			mutex.Unlock()
			return rec.record, nil
		}
		delete(dnsCache, domain)
	}
	mutex.Unlock()

	newError("Trying to query ECH config for domain: ", domain, " with ECH server: ", server).AtDebug().WriteToLog()
	record, ttl, err := dnsQuery(server, domain)
	if err != nil {
		return nil, err
	}

	if ttl > 0 {
		mutex.Lock()
		defer mutex.Unlock()
		rec.record = record
		rec.expire = time.Now().Add(time.Second * time.Duration(ttl))
		dnsCache[domain] = rec
	}

	return record, nil
}

func dnsQuery(server string, domain string) ([]byte, uint32, error) {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dns.TypeHTTPS)
	if strings.HasPrefix(strings.ToLower(server), "https://") || strings.HasPrefix(strings.ToLower(server), "h2c://") {
		m.Id = 0
	}
	msg, err := m.Pack()
	if err != nil {
		return nil, 0, err
	}
	var respMsg *dns.Msg
	if strings.HasPrefix(strings.ToLower(server), "udp://") {
		respMsg, err = udpQuery(server, msg)
	} else {
		respMsg, err = dohQuery(server, msg)
	}
	if err != nil {
		return nil, 0, err
	}
	for _, answer := range respMsg.Answer {
		if answer.Header().Class == dns.ClassINET && answer.Header().Header().Rrtype == dns.TypeHTTPS {
			if https, ok := answer.(*dns.HTTPS); ok {
				for _, v := range https.Value {
					if echConfig, ok := v.(*dns.SVCBECHConfig); ok {
						newError(context.Background(), "Get ECH config:", echConfig.String(), " TTL:", respMsg.Answer[0].Header().Ttl).AtDebug().WriteToLog()
						return echConfig.ECH, answer.Header().Ttl, nil
					}
				}
			}
		}
	}
	if respMsg.Rcode == dns.RcodeSuccess || respMsg.Rcode == dns.RcodeNameError {
		for _, ns := range respMsg.Ns {
			if soa, ok := ns.(*dns.SOA); ok {
				return nil, min(ns.Header().Ttl, soa.Minttl), nil
			}
		}
	}
	return nil, 0, newError("no ech record found")
}

func dohQuery(server string, msg []byte) (*dns.Msg, error) {
	tr := &http.Transport{
		IdleConnTimeout:   90 * time.Second,
		ForceAttemptHTTP2: true,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dest, err := net.ParseDestination(network + ":" + addr)
			if err != nil {
				return nil, err
			}
			conn, err := internet.DialSystem(ctx, dest, nil)
			if err != nil {
				return nil, err
			}
			return conn, nil
		},
	}
	if strings.HasPrefix(strings.ToLower(server), "h2c://") {
		protocols := new(http.Protocols)
		protocols.SetUnencryptedHTTP2(true)
		protocols.SetHTTP1(false)
		protocols.SetHTTP2(false)
		tr.Protocols = protocols
	}
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: tr,
	}
	defer client.CloseIdleConnections()
	req, err := http.NewRequest("POST", server, bytes.NewReader(msg))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/dns-message")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newError("query failed with response code: ", resp.StatusCode)
	}
	respMsg := new(dns.Msg)
	err = respMsg.Unpack(respBody)
	if err != nil {
		return nil, err
	}
	return respMsg, nil
}

func udpQuery(server string, msg []byte) (*dns.Msg, error) {
	server = strings.TrimPrefix(server, "udp://")
	if _, _, err := net.SplitHostPort(server); err != nil {
		if _, err := netip.ParseAddr(server); err == nil {
			server = net.JoinHostPort(server, "53")
		} else if !strings.Contains(server, ":") {
			// blame Xray for the bad parsing
			server += ":53"
		}
	}
	dest, err := net.ParseDestination("udp:" + server)
	if err != nil {
		return nil, err
	}

	udpCtx, udpCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer udpCancel()
	udpConn, err := internet.DialSystem(udpCtx, dest, nil)
	if err != nil {
		return nil, err
	}
	defer udpConn.Close()

	udpConn.SetDeadline(time.Now().Add(5 * time.Second))
	_, err = udpConn.Write(msg)
	if err != nil {
		return nil, err
	}

	response := make([]byte, buf.Size)
	n, err := udpConn.Read(response)
	if err != nil {
		return nil, err
	}
	respMsg := new(dns.Msg)
	err = respMsg.Unpack(response[:n])
	if err != nil {
		return nil, err
	}

	if !respMsg.Truncated {
		return respMsg, nil
	}

	newError("truncated, retry over TCP").AtError().WriteToLog()

	dest, err = net.ParseDestination("tcp:" + server)
	if err != nil {
		return nil, err
	}

	tcpCtx, tcpCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer tcpCancel()
	tcpConn, err := internet.DialSystem(tcpCtx, dest, nil)
	if err != nil {
		return nil, err
	}
	defer tcpConn.Close()

	tcpConn.SetDeadline(time.Now().Add(5 * time.Second))
	reqBuf := buf.NewWithSize(2 + int32(len(msg)))
	defer reqBuf.Release()
	binary.Write(reqBuf, binary.BigEndian, uint16(len(msg)))
	reqBuf.Write(msg)
	_, err = tcpConn.Write(reqBuf.Bytes())
	if err != nil {
		return nil, err
	}

	var length uint16
	err = binary.Read(tcpConn, binary.BigEndian, &length)
	if err != nil {
		return nil, err
	}
	response = make([]byte, length)
	n, err = io.ReadFull(tcpConn, response)
	if err != nil {
		return nil, err
	}
	err = respMsg.Unpack(response[:n])
	if err != nil {
		return nil, err
	}
	return respMsg, nil
}
