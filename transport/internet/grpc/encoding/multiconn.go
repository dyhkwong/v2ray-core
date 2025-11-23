package encoding

import (
	"context"
	"io"
	"net"

	"google.golang.org/grpc/peer"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net/cnc"
	"github.com/v2fly/v2ray-core/v5/common/signal/done"
)

type GunMultiService interface {
	Context() context.Context
	Send(*MultiHunk) error
	Recv() (*MultiHunk, error)
}

type StreamCloser interface {
	CloseSend() error
}

type HunkReaderWriter struct {
	service GunMultiService
	over    context.CancelFunc
	done    *done.Instance
	buf     [][]byte
}

func NewHunkReadWriter(service GunMultiService, over context.CancelFunc) *HunkReaderWriter {
	return &HunkReaderWriter{
		service: service,
		over:    over,
		done:    done.New(),
		buf:     nil,
	}
}

func NewGunMultiConn(service GunMultiService, over context.CancelFunc) net.Conn {
	var rAddr net.Addr
	pr, ok := peer.FromContext(service.Context())
	if ok {
		rAddr = pr.Addr
	} else {
		rAddr = &net.TCPAddr{
			IP:   []byte{0, 0, 0, 0},
			Port: 0,
		}
	}
	wrc := NewHunkReadWriter(service, over)
	return cnc.NewConnection(
		cnc.ConnectionInputMulti(wrc),
		cnc.ConnectionOutputMulti(wrc),
		cnc.ConnectionOnClose(wrc),
		cnc.ConnectionRemoteAddr(rAddr),
	)
}

func (h *HunkReaderWriter) forceFetch() error {
	hunk, err := h.service.Recv()
	if err != nil {
		if err == io.EOF {
			return err
		}

		return newError("failed to fetch hunk from gRPC tunnel").Base(err)
	}
	h.buf = hunk.Data
	return nil
}

func (h *HunkReaderWriter) ReadMultiBuffer() (buf.MultiBuffer, error) {
	if h.done.Done() {
		return nil, io.EOF
	}
	if err := h.forceFetch(); err != nil {
		return nil, err
	}
	mb := make(buf.MultiBuffer, 0, len(h.buf))
	for _, b := range h.buf {
		if len(b) == 0 {
			continue
		}
		nb := buf.NewWithSize(int32(cap(b)))
		nb.Extend(int32(len(b)))
		copy(nb.Bytes(), b)
		mb = append(mb, nb)
	}
	return mb, nil
}

func (h *HunkReaderWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	defer buf.ReleaseMulti(mb)
	if h.done.Done() {
		return io.ErrClosedPipe
	}
	data := make([][]byte, 0, len(mb))
	for _, b := range mb {
		if b.Len() > 0 {
			data = append(data, b.Bytes())
		}
	}
	err := h.service.Send(&MultiHunk{Data: data})
	if err != nil {
		return err
	}
	return nil
}

func (h *HunkReaderWriter) Close() error {
	if h.over != nil {
		h.over()
	}
	if closer, ok := h.service.(StreamCloser); ok {
		return closer.CloseSend()
	}
	return h.done.Close()
}
