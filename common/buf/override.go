package buf

import (
	"github.com/v2fly/v2ray-core/v5/common/net"
)

type EndpointOverrideReader struct {
	Reader
	Dest         net.Address
	OriginalDest net.Address
}

func (r *EndpointOverrideReader) ReadMultiBuffer() (MultiBuffer, error) {
	mb, err := r.Reader.ReadMultiBuffer()
	if err == nil {
		for _, b := range mb {
			if b.Endpoint != nil && b.Endpoint.Address == r.OriginalDest {
				b.Endpoint.Address = r.Dest
			}
		}
	}
	return mb, err
}

type EndpointOverrideWriter struct {
	Writer
	Dest         net.Address
	OriginalDest net.Address
}

func (w *EndpointOverrideWriter) WriteMultiBuffer(mb MultiBuffer) error {
	for _, b := range mb {
		if b.Endpoint != nil && b.Endpoint.Address == w.Dest {
			b.Endpoint.Address = w.OriginalDest
		}
	}
	return w.Writer.WriteMultiBuffer(mb)
}
