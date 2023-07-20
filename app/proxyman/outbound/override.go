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
	Handler      *Handler
	mutex        *sync.Mutex
	ipToDomain   map[net.Address]net.Address
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
		if b.Endpoint.Address == r.OriginalDest && r.Dest != nil {
			b.Endpoint.Address = r.Dest
			continue
		}
		if b.Endpoint.Address.Family().IsIP() && r.ipToDomain != nil {
			r.mutex.Lock()
			domain, ok := r.ipToDomain[b.Endpoint.Address]
			r.mutex.Unlock()
			if ok {
				b.Endpoint.Address = domain
			}
		}
		if b.Endpoint.Address.Family().IsDomain() && r.OriginalDest != nil && r.OriginalDest.Family().IsIP() && r.Handler.fakedns != nil {
			if fakedns, ok := r.Handler.fakedns.(dns.FakeDNSEngineRev0); ok {
				if ips := fakedns.GetFakeIPForDomain3(b.Endpoint.Address.Domain(),
					r.OriginalDest.Family().IsIPv4(),
					r.OriginalDest.Family().IsIPv6(),
				); len(ips) > 0 {
					b.Endpoint.Address = ips[0]
				}
			} else {
				ips := r.Handler.fakedns.GetFakeIPForDomain(b.Endpoint.Address.Domain())
				for _, ip := range ips {
					if ip.Family().IsIPv4() == r.OriginalDest.Family().IsIPv4() {
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
	Handler      *Handler
	mutex        *sync.Mutex
	ipToDomain   map[net.Address]net.Address
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
		if b.Endpoint.Address.Family().IsIP() && w.Handler.fakedns != nil {
			if domain := w.Handler.fakedns.GetDomainFromFakeDNS(b.Endpoint.Address); len(domain) > 0 {
				b.Endpoint.Address = net.DomainAddress(domain)
			}
		}
		if w.ipToDomain != nil {
			switch b.Endpoint.Address.Family() {
			case net.AddressFamilyDomain:
				if ip := w.Handler.resolveIP(w.Handler.ctx, b.Endpoint.Address.Domain(), w.Handler.Address()); ip != nil {
					w.mutex.Lock()
					domain, ok := w.ipToDomain[ip]
					if !ok {
						w.ipToDomain[ip] = b.Endpoint.Address
						w.mutex.Unlock()
						b.Endpoint.Address = ip
					} else {
						w.mutex.Unlock()
						if domain != b.Endpoint.Address {
							newError("ignored mapping conflict for ", ip, ", existing: ", domain, ", new: ", b.Endpoint.Address).AtDebug().WriteToLog()
						}
					}
				}
			case net.AddressFamilyIPv4, net.AddressFamilyIPv6:
				w.mutex.Lock()
				domain, ok := w.ipToDomain[b.Endpoint.Address]
				w.mutex.Unlock()
				if ok {
					newError("ignored mapping conflict for ", b.Endpoint.Address, ", existing: ", domain).AtDebug().WriteToLog()
				}
			}
		}
	}
	return w.Writer.WriteMultiBuffer(mb)
}
