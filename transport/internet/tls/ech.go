//go:build !confonly

package tls

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/v2fly/v2ray-core/v4/common/net"
	"github.com/v2fly/v2ray-core/v4/transport/internet"
)

// from "github.com/v2fly/v2ray-core/v5/app/dns", import cycle
func fqdn(domain string) string {
	if len(domain) > 0 && strings.HasSuffix(domain, ".") {
		return domain
	}
	return domain + "."
}

func ApplyECH(c *Config, config *tls.Config) error {
	var echConfig []byte
	var err error
	var domain string

	if len(c.EchConfig) > 0 {
		echConfig = c.EchConfig
	} else { // ECH config > DOH lookup
		if c.EchQueryDomain == "" {
			domain = config.ServerName
		} else {
			domain = c.EchQueryDomain
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
	m := new(dnsmessage.Message)
	m.Questions = []dnsmessage.Question{
		{
			Name:  dnsmessage.MustNewName(fqdn(domain)),
			Type:  dnsmessage.TypeHTTPS,
			Class: dnsmessage.ClassINET,
		},
	}
	m.RecursionDesired = true
	m.ID = 0
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
		return nil, 0, newError("query failed with response code: ", resp.StatusCode)
	}
	respMsg := new(dnsmessage.Message)
	err = respMsg.Unpack(respBody)
	if err != nil {
		return nil, 0, err
	}
	for _, answer := range respMsg.Answers {
		if answer.Header.Class == dnsmessage.ClassINET && answer.Header.Type == dnsmessage.TypeHTTPS {
			if https, ok := answer.Body.(*dnsmessage.HTTPSResource); ok {
				for _, param := range https.Params {
					if param.Key == dnsmessage.SVCParamECH {
						newError("Get ECH config: ", base64.StdEncoding.EncodeToString(param.Value), " TTL: ", answer.Header.TTL).AtDebug().WriteToLog()
						return param.Value, answer.Header.TTL, nil
					}
				}
			}
		}
	}
	if respMsg.RCode == dnsmessage.RCodeSuccess || respMsg.RCode == dnsmessage.RCodeNameError {
		for _, authority := range respMsg.Authorities {
			if authority.Header.Class == dnsmessage.ClassINET {
				if soa, ok := authority.Body.(*dnsmessage.SOAResource); ok {
					return nil, min(authority.Header.TTL, soa.MinTTL), nil
				}
			}
		}
	}
	return nil, 0, newError("no ech record found")
}
