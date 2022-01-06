package buf

type EndpointErasureReader struct {
	Reader
}

func (r *EndpointErasureReader) ReadMultiBuffer() (MultiBuffer, error) {
	mb, err := r.Reader.ReadMultiBuffer()
	if err == nil {
		for _, buffer := range mb {
			buffer.Endpoint = nil
		}
	}
	return mb, err
}

type EndpointErasureWriter struct {
	Writer
}

func (w *EndpointErasureWriter) WriteMultiBuffer(mb MultiBuffer) error {
	for _, buffer := range mb {
		buffer.Endpoint = nil
	}
	return w.Writer.WriteMultiBuffer(mb)
}
