//go:build android && cgo

package localdns

// #include <stdlib.h>
// #include <android/multinetwork.h>
import "C"

import (
	"context"

	"golang.org/x/net/dns/dnsmessage"
	"golang.org/x/sys/unix"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/bytespool"
)

var rawQueryFunc = func(request []byte) ([]byte, error) {
	requestMsg := new(dnsmessage.Message)
	err := requestMsg.Unpack(request)
	if err != nil {
		return nil, newError("failed to parse dns request").Base(err)
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

	if n := C.android_res_nresult(fd, &rcode, (*C.uint8_t)(&response[0]), C.size_t(len(response))); n < 0 {
		return nil, unix.Errno(-n)
	}
	return response[:n], nil
}
