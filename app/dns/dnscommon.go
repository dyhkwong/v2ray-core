//go:build !confonly

package dns

import (
	"strings"
	"time"

	"github.com/miekg/dns"

	"github.com/v2fly/v2ray-core/v4/common/errors"
	"github.com/v2fly/v2ray-core/v4/common/net"
	dns_feature "github.com/v2fly/v2ray-core/v4/features/dns"
)

var errTruncated = newError("truncated")

// Fqdn normalizes domain make sure it ends with '.'
func Fqdn(domain string) string {
	if len(domain) > 0 && strings.HasSuffix(domain, ".") {
		return domain
	}
	return domain + "."
}

type record struct {
	A    *IPRecord
	AAAA *IPRecord
}

// IPRecord is a cacheable item for a resolved domain
type IPRecord struct {
	ReqID  uint16
	IP     []net.Address
	Expire time.Time
	RCode  int
	TTL    uint32
}

func (r *IPRecord) getIPs() ([]net.Address, time.Time, error) {
	if r == nil || r.TTL > 0 && r.Expire.Before(time.Now()) {
		return nil, time.Time{}, errRecordNotFound
	}
	if r.RCode != dns.RcodeSuccess {
		return nil, r.Expire, dns_feature.RCodeError(r.RCode)
	}
	return r.IP, r.Expire, nil
}

func isNewer(baseRec *IPRecord, newRec *IPRecord) bool {
	if newRec == nil {
		return false
	}
	if baseRec == nil {
		return true
	}
	return baseRec.Expire.Before(newRec.Expire)
}

var errRecordNotFound = errors.New("record not found")

type dnsRequest struct {
	reqType uint16
	domain  string
	start   time.Time
	expire  time.Time
	msg     *dns.Msg
}

func genEDNS0Subnet(clientIP net.IP) dns.EDNS0 {
	if len(clientIP) == 0 {
		return nil
	}
	var netmask uint8
	var family uint16
	if len(clientIP) == 4 {
		family = 1
		netmask = 24 // 24 for IPV4, 96 for IPv6
	} else {
		family = 2
		netmask = 96
	}
	return &dns.EDNS0_SUBNET{
		Code:          dns.EDNS0SUBNET,
		Family:        family,
		Address:       clientIP,
		SourceNetmask: netmask,
		SourceScope:   0,
	}
}

func genEDNS0Options(clientIP net.IP) dns.RR {
	if len(clientIP) == 0 {
		return nil
	}
	opt := &dns.OPT{
		Hdr: dns.RR_Header{
			Name:   ".",
			Rrtype: dns.TypeOPT,
			Class:  dns.ClassINET,
		},
	}
	opt.SetUDPSize(1350)
	opt.SetExtendedRcode(0xfe00)
	opt.SetDo(true)
	opt.Option = append(opt.Option, genEDNS0Subnet(clientIP))
	return opt
}

func buildReqMsgs(domain string, option dns_feature.IPOption, reqIDGen func() uint16, reqOpts dns.RR) []*dnsRequest {
	qA := dns.Question{
		Name:   dns.Fqdn(domain),
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	}

	qAAAA := dns.Question{
		Name:   dns.Fqdn(domain),
		Qtype:  dns.TypeAAAA,
		Qclass: dns.ClassINET,
	}

	var reqs []*dnsRequest
	now := time.Now()

	if option.IPv4Enable {
		msg := new(dns.Msg)
		msg.Id = reqIDGen()
		msg.RecursionDesired = true
		msg.Question = []dns.Question{qA}
		if reqOpts != nil {
			msg.Extra = append(msg.Extra, reqOpts)
		}
		reqs = append(reqs, &dnsRequest{
			reqType: dns.TypeA,
			domain:  domain,
			start:   now,
			msg:     msg,
		})
	}

	if option.IPv6Enable {
		msg := new(dns.Msg)
		msg.Id = reqIDGen()
		msg.RecursionDesired = true
		msg.Question = []dns.Question{qAAAA}
		if reqOpts != nil {
			msg.Extra = append(msg.Extra, reqOpts)
		}
		reqs = append(reqs, &dnsRequest{
			reqType: dns.TypeAAAA,
			domain:  domain,
			start:   now,
			msg:     msg,
		})
	}

	return reqs
}

// parseResponse parses DNS answers from the returned payload
func parseResponse(payload []byte) (*IPRecord, error) {
	message := new(dns.Msg)
	err := message.Unpack(payload)
	if err != nil {
		return nil, newError("failed to parse DNS response").Base(err).AtWarning()
	}

	now := time.Now()
	ipRecord := &IPRecord{
		ReqID:  message.Id,
		RCode:  message.Rcode,
		Expire: now,
		TTL:    0,
	}

	if message.Truncated {
		return ipRecord, errTruncated
	}

	for _, answer := range message.Answer {
		ipRecord.TTL = answer.Header().Ttl
		ipRecord.Expire = now.Add(time.Duration(ipRecord.TTL) * time.Second)

		switch answer.Header().Rrtype {
		case dns.TypeA:
			ipRecord.IP = append(ipRecord.IP, net.IPAddress(answer.(*dns.A).A))
		case dns.TypeAAAA:
			ipRecord.IP = append(ipRecord.IP, net.IPAddress(answer.(*dns.AAAA).AAAA))
		default:
			continue
		}
	}

	if len(ipRecord.IP) == 0 && message.Rcode == dns.RcodeSuccess || message.Rcode == dns.RcodeNameError {
		for _, ns := range message.Ns {
			switch ns.Header().Rrtype {
			case dns.TypeSOA:
				ipRecord.TTL = min(ns.Header().Ttl, ns.(*dns.SOA).Minttl)
				ipRecord.Expire = now.Add(time.Duration(ipRecord.TTL) * time.Second)
			default:
				continue
			}
		}
	}

	return ipRecord, nil
}

func filterIP(ips []net.Address, option dns_feature.IPOption) []net.Address {
	filtered := make([]net.Address, 0, len(ips))
	for _, ip := range ips {
		if (ip.Family().IsIPv4() && option.IPv4Enable) || (ip.Family().IsIPv6() && option.IPv6Enable) {
			filtered = append(filtered, ip)
		}
	}
	return filtered
}

func formatRR(str string) string {
	if strings.HasPrefix(str, "\n;; OPT PSEUDOSECTION:\n; ") {
		str, _ = strings.CutPrefix(str, "\n;; OPT PSEUDOSECTION:\n; ")
	} else if strings.HasPrefix(str, ";") {
		str, _ = strings.CutPrefix(str, ";")
	}
	return strings.ReplaceAll(str, "\t", " ")
}
