package tls

import (
	"context"
	"crypto/tls"
	"sync"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/crypto/cryptobyte"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common/net"
	feature_dns "github.com/v2fly/v2ray-core/v5/features/dns"
)

func (c *Config) applyECH(ctx context.Context, config *tls.Config) error {
	if len(c.Ech.Config) > 0 {
		config.EncryptedClientHelloConfigList = c.Ech.Config
		return nil
	}
	domain := config.ServerName
	if len(c.Ech.QueryDomain) > 0 {
		domain = c.Ech.QueryDomain
	}
	if len(domain) == 0 {
		return newError("ech requires dnsname")
	}
	if addr := net.ParseAddress(domain); !addr.Family().IsDomain() {
		return newError("ech requires dnsname")
	}
	echConfig, err := queryECH(ctx, domain)
	if err != nil {
		return err
	}
	config.EncryptedClientHelloConfigList = echConfig
	return nil
}

type ech struct {
	raw    []byte
	expire time.Time
}

var (
	echCache      = make(map[string]*ech)
	echCacheMutex sync.RWMutex
)

func queryECH(ctx context.Context, domain string) ([]byte, error) {
	echCacheMutex.Lock()
	if ech, found := echCache[domain]; found {
		if ech.expire.After(time.Now()) {
			echCacheMutex.Unlock()
			if ech.raw == nil {
				return nil, newError("ech config not found")
			}
			return ech.raw, nil
		}
		delete(echCache, domain)
	}
	echCacheMutex.Unlock()

	instance := core.FromContext(ctx)
	if instance == nil {
		return nil, newError("nil instance")
	}
	dnsClient, ok := instance.GetFeature(feature_dns.ClientType()).(feature_dns.Client)
	if !ok || dnsClient == nil {
		return nil, newError("nil dns client")
	}
	dnsRawClient, ok := dnsClient.(feature_dns.RawQuery)
	if !ok {
		return nil, newError("dns client does not support raw query")
	}

	requestMsg := new(dns.Msg)
	requestMsg.SetQuestion(dns.Fqdn(domain), dns.TypeHTTPS)
	request, err := requestMsg.Pack()
	if err != nil {
		return nil, err
	}
	response, err := dnsRawClient.QueryRaw(request)
	if err != nil {
		return nil, err
	}
	responseMsg := new(dns.Msg)
	if err := responseMsg.Unpack(response); err != nil {
		return nil, err
	}
	for _, answer := range responseMsg.Answer {
		if answer.Header().Class == dns.ClassINET && answer.Header().Header().Rrtype == dns.TypeHTTPS {
			if https, ok := answer.(*dns.HTTPS); ok {
				ttl := answer.Header().Ttl
				for _, v := range https.Value {
					if echConfig, ok := v.(*dns.SVCBECHConfig); ok {
						if ttl > 0 {
							echCacheMutex.Lock()
							echCache[domain] = &ech{
								raw:    echConfig.ECH,
								expire: time.Now().Add(time.Second * time.Duration(ttl)),
							}
							echCacheMutex.Unlock()
						}
						return echConfig.ECH, nil
					}
				}
				if ttl > 0 {
					echCacheMutex.Lock()
					echCache[domain] = &ech{
						raw:    nil,
						expire: time.Now().Add(time.Second * time.Duration(ttl)),
					}
					echCacheMutex.Unlock()
					return nil, newError("ech config not found")
				}
			}
		}
	}
	if responseMsg.Rcode == dns.RcodeSuccess || responseMsg.Rcode == dns.RcodeNameError {
		for _, ns := range responseMsg.Ns {
			if soa, ok := ns.(*dns.SOA); ok {
				if ttl := min(ns.Header().Ttl, soa.Minttl); ttl > 0 {
					echCacheMutex.Lock()
					echCache[domain] = &ech{
						raw:    nil,
						expire: time.Now().Add(time.Second * time.Duration(ttl)),
					}
					echCacheMutex.Unlock()
					return nil, newError("ech config not found")
				}
			}
		}
	}
	return nil, newError("ech config not found")
}

func unmarshalECHKeys(raw []byte) ([]tls.EncryptedClientHelloKey, error) {
	var keys []tls.EncryptedClientHelloKey
	rawString := cryptobyte.String(raw)
	for !rawString.Empty() {
		var key tls.EncryptedClientHelloKey
		if !rawString.ReadUint16LengthPrefixed((*cryptobyte.String)(&key.PrivateKey)) {
			return nil, newError("parse ech keys private key error")
		}
		if !rawString.ReadUint16LengthPrefixed((*cryptobyte.String)(&key.Config)) {
			return nil, newError("parsing ech keys config error")
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return nil, newError("empty ech keys")
	}
	return keys, nil
}
