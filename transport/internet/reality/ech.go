package reality

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/miekg/dns"
	utls "github.com/refraction-networking/utls"

	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
)

func ApplyECH(c *Config, config *utls.Config) error {
	if len(c.EchConfig) > 0 {
		config.EncryptedClientHelloConfigList = c.EchConfig
		return nil
	}
	// ECH config > DOH lookup
	var domain string
	if c.EchQueryDomain == "" {
		domain = config.ServerName
	} else {
		domain = c.EchQueryDomain
	}
	addr := net.ParseAddress(domain)
	if !addr.Family().IsDomain() {
		return newError("Using DOH for ECH needs SNI")
	}
	echConfig, err := QueryRecord(addr.Domain(), c.Ech_DOHserver)
	if err != nil {
		return err
	}
	if len(echConfig) == 0 {
		return newError("no ech record found")
	}
	config.EncryptedClientHelloConfigList = echConfig
	return nil
}

type record struct {
	record []byte
	expire time.Time
}

var (
	dnsCache = make(map[string]*record)
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
	record, ttl, err := dohQuery(server, domain)
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

func dohQuery(server string, domain string) ([]byte, uint32, error) {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dns.TypeHTTPS)
	m.Id = 0
	msg, err := m.Pack()
	if err != nil {
		return nil, 0, err
	}
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
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: tr,
	}
	req, err := http.NewRequest("POST", server, bytes.NewReader(msg))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/dns-message")
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, 0, newError("query failed with response code:", resp.StatusCode)
	}
	respMsg := new(dns.Msg)
	if err = respMsg.Unpack(respBody); err != nil {
		return nil, 0, err
	}
	if len(respMsg.Answer) > 0 {
		for _, answer := range respMsg.Answer {
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
