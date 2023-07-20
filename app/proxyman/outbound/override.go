package outbound

import (
	"sync"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/features/dns"
)

type EndpointOverrideReader struct {
	buf.Reader
	Dest         net.Address
	OriginalDest net.Address
	ipToDomain   *sync.Map
	fakedns      dns.FakeDNSEngine
	usedFakeIPs  *sync.Map
}

func (r *EndpointOverrideReader) ReadMultiBuffer() (buf.MultiBuffer, error) {
	mb, err := r.Reader.ReadMultiBuffer()
	if err != nil {
		return nil, err
	}
	for _, b := range mb {
		if b.Endpoint == nil {
			continue
		}
		if b.Endpoint.Address == r.OriginalDest {
			b.Endpoint.Address = r.Dest
			continue
		}
		if r.ipToDomain != nil && b.Endpoint.Address.Family().IsIP() {
			if domain, ok := r.ipToDomain.Load(b.Endpoint.Address); ok {
				b.Endpoint.Address = domain.(net.Address)
			}
		}
		if r.fakedns != nil && r.usedFakeIPs != nil && b.Endpoint.Address.Family().IsDomain() {
			if ips := r.fakedns.GetFakeIPForDomain(b.Endpoint.Address.Domain()); len(ips) > 0 {
				for _, ip := range ips {
					if _, ok := r.usedFakeIPs.Load(ip); ok {
						b.Endpoint.Address = ip
						break
					}
				}
			}
		}
	}
	return mb, nil
}

type EndpointOverrideWriter struct {
	buf.Writer
	Dest         net.Address
	OriginalDest net.Address
	fakedns      dns.FakeDNSEngine
	usedFakeIPs  *sync.Map
	resolveIP    func(domain string) net.Address
	ipToDomain   *sync.Map
}

func (w *EndpointOverrideWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	for _, b := range mb {
		if b.Endpoint == nil {
			continue
		}
		if b.Endpoint.Address == w.Dest && w.OriginalDest != nil {
			b.Endpoint.Address = w.OriginalDest
			continue
		}
		if w.fakedns != nil && w.usedFakeIPs != nil && b.Endpoint.Address.Family().IsIP() {
			if domain := w.fakedns.GetDomainFromFakeDNS(b.Endpoint.Address); len(domain) > 0 {
				w.usedFakeIPs.LoadOrStore(b.Endpoint.Address, true)
				b.Endpoint.Address = net.DomainAddress(domain)
			}
		}
		if w.resolveIP != nil && w.ipToDomain != nil && b.Endpoint.Address.Family().IsDomain() {
			if ip := w.resolveIP(b.Endpoint.Address.Domain()); ip != nil {
				w.ipToDomain.LoadOrStore(ip, b.Endpoint.Address)
				b.Endpoint.Address = ip
			}
		}
	}
	return w.Writer.WriteMultiBuffer(mb)
}
