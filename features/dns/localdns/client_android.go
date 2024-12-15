//go:build android && cgo

package localdns

// #cgo LDFLAGS: -landroid
// #include <stdlib.h>
// #include <android/multinetwork.h>
import "C"

import (
	"context"

	"golang.org/x/net/dns/dnsmessage"
	"golang.org/x/sys/unix"

	"github.com/v2fly/v2ray-core/v4/common/buf"
)

var rawQueryFunc = func(request []byte) ([]byte, error) {
	message := new(dnsmessage.Message)
	if err := message.Unpack(request); err != nil {
		return nil, newError("failed to parse dns request").Base(err)
	}
	if message.Response || len(message.Answers) > 0 {
		return nil, newError("failed to parse dns request: not query")
	}

	fd := C.android_res_nsend(0, (*C.uint8_t)(&request[0]), C.size_t(len(request)), 0)
	if fd < 0 {
		return nil, unix.Errno(-fd)
	}
	nReady, err := unix.Poll([]unix.PollFd{{Fd: int32(fd), Events: unix.EPOLLIN | unix.EPOLLERR}}, 5000)
	if err != nil {
		return nil, err
	}
	if nReady == 0 {
		return nil, context.DeadlineExceeded
	}
	response := make([]byte, buf.Size)
	var rcode C.int

	n := C.android_res_nresult(fd, &rcode, (*C.uint8_t)(&response[0]), C.size_t(len(response)))
	if n < 0 {
		return nil, unix.Errno(-n)
	}

	if err := message.Unpack(response[:n]); err != nil {
		return nil, newError("failed to parse dns response").Base(err)
	}

	return response[:n], nil
}
