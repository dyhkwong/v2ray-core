package mux

import (
	"github.com/v2fly/v2ray-core/v5/common/buf"
)

type endpointWrapperWriter struct {
	buf.Writer
	*Session
}

func (w *endpointWrapperWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	for _, buffer := range mb {
		if buffer.Endpoint != nil {
			if w.sendEndpoint == -1 || *buffer.Endpoint == w.endpoint {
				buffer.Endpoint = nil
			}
		}
	}
	return w.Writer.WriteMultiBuffer(mb)
}
