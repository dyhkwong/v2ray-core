//go:build !confonly
// +build !confonly

package dns

import (
	"encoding/base64"
	"encoding/binary"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/v2fly/v2ray-core/v4/common"
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

// IPRecord is a cacheable item for a resolved domain
type IPRecord struct {
	ReqID  uint16
	IP     []net.Address
	Expire time.Time
	RCode  dnsmessage.RCode
	TTL    uint32
}

func (r *IPRecord) getIPs() ([]net.Address, time.Time, error) {
	if r == nil || r.TTL > 0 && r.Expire.Before(time.Now()) {
		return nil, time.Time{}, errRecordNotFound
	}
	if r.RCode != dnsmessage.RCodeSuccess {
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
	reqType dnsmessage.Type
	domain  string
	start   time.Time
	expire  time.Time
	msg     *dnsmessage.Message
}

func genEDNS0Subnet(clientIP net.IP) *dnsmessage.Option {
	if len(clientIP) == 0 {
		return nil
	}

	var netmask int
	var family uint16

	if len(clientIP) == 4 {
		family = 1
		netmask = 24 // 24 for IPV4, 96 for IPv6
	} else {
		family = 2
		netmask = 96
	}

	b := make([]byte, 4)
	binary.BigEndian.PutUint16(b[0:], family)
	b[2] = byte(netmask)
	b[3] = 0
	switch family {
	case 1:
		ip := clientIP.To4().Mask(net.CIDRMask(netmask, net.IPv4len*8))
		needLength := (netmask + 8 - 1) / 8 // division rounding up
		b = append(b, ip[:needLength]...)
	case 2:
		ip := clientIP.Mask(net.CIDRMask(netmask, net.IPv6len*8))
		needLength := (netmask + 8 - 1) / 8 // division rounding up
		b = append(b, ip[:needLength]...)
	}

	const EDNS0SUBNET = 0x08

	return &dnsmessage.Option{
		Code: EDNS0SUBNET,
		Data: b,
	}
}

func genEDNS0Options(clientIP net.IP) *dnsmessage.Resource {
	if len(clientIP) == 0 {
		return nil
	}

	opt := new(dnsmessage.Resource)
	common.Must(opt.Header.SetEDNS0(1350, 0xfe00, true))

	opt.Body = &dnsmessage.OPTResource{
		Options: []dnsmessage.Option{*(genEDNS0Subnet(clientIP))},
	}

	return opt
}

func buildReqMsgs(domain string, option dns_feature.IPOption, reqIDGen func() uint16, reqOpts *dnsmessage.Resource) []*dnsRequest {
	qA := dnsmessage.Question{
		Name:  dnsmessage.MustNewName(domain),
		Type:  dnsmessage.TypeA,
		Class: dnsmessage.ClassINET,
	}

	qAAAA := dnsmessage.Question{
		Name:  dnsmessage.MustNewName(domain),
		Type:  dnsmessage.TypeAAAA,
		Class: dnsmessage.ClassINET,
	}

	var reqs []*dnsRequest
	now := time.Now()

	if option.IPv4Enable {
		msg := new(dnsmessage.Message)
		msg.Header.ID = reqIDGen()
		msg.Header.RecursionDesired = true
		msg.Questions = []dnsmessage.Question{qA}
		if reqOpts != nil {
			msg.Additionals = append(msg.Additionals, *reqOpts)
		}
		reqs = append(reqs, &dnsRequest{
			reqType: dnsmessage.TypeA,
			domain:  domain,
			start:   now,
			msg:     msg,
		})
	}

	if option.IPv6Enable {
		msg := new(dnsmessage.Message)
		msg.Header.ID = reqIDGen()
		msg.Header.RecursionDesired = true
		msg.Questions = []dnsmessage.Question{qAAAA}
		if reqOpts != nil {
			msg.Additionals = append(msg.Additionals, *reqOpts)
		}
		reqs = append(reqs, &dnsRequest{
			reqType: dnsmessage.TypeAAAA,
			domain:  domain,
			start:   now,
			msg:     msg,
		})
	}

	return reqs
}

// parseResponse parses DNS answers from the returned payload
func parseResponse(payload []byte) (*IPRecord, error) {
	var parser dnsmessage.Parser
	h, err := parser.Start(payload)
	if err != nil {
		return nil, newError("failed to parse DNS response").Base(err).AtWarning()
	}
	if err := parser.SkipAllQuestions(); err != nil {
		return nil, newError("failed to skip questions in DNS response").Base(err).AtWarning()
	}

	now := time.Now()
	ipRecord := &IPRecord{
		ReqID:  h.ID,
		RCode:  h.RCode,
		Expire: now,
		TTL:    0,
	}

	if h.Truncated {
		return ipRecord, errTruncated
	}

L:
	for {
		ah, err := parser.AnswerHeader()
		if err != nil {
			if err != dnsmessage.ErrSectionDone {
				newError("failed to parse answer section for domain: ", ah.Name.String()).Base(err).WriteToLog()
			}
			break
		}

		ipRecord.TTL = ah.TTL
		ipRecord.Expire = now.Add(time.Duration(ipRecord.TTL) * time.Second)

		switch ah.Type {
		case dnsmessage.TypeA:
			ans, err := parser.AResource()
			if err != nil {
				newError("failed to parse A record for domain: ", ah.Name).Base(err).WriteToLog()
				break L
			}
			ipRecord.IP = append(ipRecord.IP, net.IPAddress(ans.A[:]))
		case dnsmessage.TypeAAAA:
			ans, err := parser.AAAAResource()
			if err != nil {
				newError("failed to parse AAAA record for domain: ", ah.Name).Base(err).WriteToLog()
				break L
			}
			ipRecord.IP = append(ipRecord.IP, net.IPAddress(ans.AAAA[:]))
		default:
			if err := parser.SkipAnswer(); err != nil {
				newError("failed to skip answer").Base(err).WriteToLog()
				break L
			}
			continue
		}
	}

	if len(ipRecord.IP) == 0 && h.RCode == dnsmessage.RCodeSuccess || h.RCode == dnsmessage.RCodeNameError {
	L1:
		for {
			ah, err := parser.AuthorityHeader()
			if err != nil {
				if err != dnsmessage.ErrSectionDone {
					newError("failed to parse authority section for domain: ", ah.Name.String()).Base(err).WriteToLog()
				}
				break
			}
			switch ah.Type {
			case dnsmessage.TypeSOA:
				ans, err := parser.SOAResource()
				if err != nil {
					newError("failed to parse SOA record for domain: ", ah.Name).Base(err).WriteToLog()
					break L1
				}
				ipRecord.TTL = min(ah.TTL, ans.MinTTL)
				ipRecord.Expire = now.Add(time.Duration(ipRecord.TTL) * time.Second)
			default:
				if err := parser.SkipAuthority(); err != nil {
					newError("failed to skip authority").Base(err).WriteToLog()
					break L1
				}
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

func resourceBodyToString(resource dnsmessage.Resource) string {
	switch body := resource.Body.(type) {
	case *dnsmessage.AResource:
		return net.IPAddress(body.A[:]).IP().String()
	case *dnsmessage.AAAAResource:
		return net.IPAddress(body.AAAA[:]).IP().String()
	case *dnsmessage.CNAMEResource:
		return body.CNAME.String()
	case *dnsmessage.MXResource:
		return body.MX.String()
	case *dnsmessage.NSResource:
		return body.NS.String()
	case *dnsmessage.PTRResource:
		return body.PTR.String()
	case *dnsmessage.SOAResource:
		return body.NS.String() + " " + body.MBox.String()
	case *dnsmessage.TXTResource:
		return strings.Join(body.TXT, " ")
	case *dnsmessage.SRVResource:
		return net.JoinHostPort(body.Target.String(), strconv.Itoa(int(body.Port)))
	case *dnsmessage.SVCBResource, *dnsmessage.HTTPSResource:
		var target string
		var params []dnsmessage.SVCParam
		switch body := resource.Body.(type) {
		case *dnsmessage.SVCBResource:
			target = body.Target.String()
			params = body.Params

		case *dnsmessage.HTTPSResource:
			target = body.Target.String()
			params = body.Params
		}
		var paramString []string
		for _, param := range params {
			switch param.Key {
			case dnsmessage.SVCParamPort:
				paramString = append(paramString, "port="+strconv.Itoa(int(binary.BigEndian.Uint16(param.Value))))

			case dnsmessage.SVCParamECH:
				paramString = append(paramString, "ech="+base64.StdEncoding.EncodeToString(param.Value))
			case dnsmessage.SVCParamNoDefaultALPN:
				paramString = append(paramString, "no-default-alpn")
			case dnsmessage.SVCParamALPN:
				var alpnString []string
				pos := 0
				for pos+1 <= len(param.Value) {
					length := param.Value[pos]
					pos++
					if pos+int(length) > len(param.Value) {
						break
					}
					alpnString = append(alpnString, string(param.Value[pos:pos+int(length)]))
					pos += int(length)
				}
				paramString = append(paramString, "alpn="+strings.Join(alpnString, ","))
			case dnsmessage.SVCParamIPv4Hint:
				ipv4String := make([]string, len(param.Value)/4)
				for i := range len(param.Value) / 4 {
					ipv4String[i] = net.IPAddress(param.Value[i*4 : i*4+4]).IP().String()
				}
				paramString = append(paramString, "ipv4hint="+strings.Join(ipv4String, ","))
			case dnsmessage.SVCParamIPv6Hint:
				ipv6String := make([]string, len(param.Value)/16)
				for i := range len(param.Value) / 16 {
					ipv6String[i] = net.IPAddress(param.Value[i*16 : i*16+16]).IP().String()
				}
				paramString = append(paramString, "ipv6hint="+strings.Join(ipv6String, ","))
			}
		}
		return target + " " + strings.Join(paramString, " ")
	default:
		return ""
	}
}
