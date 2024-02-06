package internet

import (
	"net"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"

	v2net "github.com/v2fly/v2ray-core/v5/common/net"
)

const (
	// TCP_FASTOPEN_SERVER is the value to enable TCP fast open on darwin for server connections.
	TCP_FASTOPEN_SERVER = 0x01 // nolint: revive,stylecheck
	// TCP_FASTOPEN_CLIENT is the value to enable TCP fast open on darwin for client connections.
	TCP_FASTOPEN_CLIENT = 0x02 // nolint: revive,stylecheck
)

const (
	PfOut       = 2
	IOCOut      = 0x40000000
	IOCIn       = 0x80000000
	IOCInOut    = IOCIn | IOCOut
	IOCPARMMask = 0x1FFF
	LEN         = 4*16 + 4*4 + 4*1
	// #define	_IOC(inout,group,num,len) (inout | ((len & IOCPARMMask) << 16) | ((group) << 8) | (num))
	// #define	_IOWR(g,n,t)	_IOC(IOCInOut,	(g), (n), sizeof(t))
	// #define DIOCNATLOOK		_IOWR('D', 23, struct pfioc_natlook)
	DIOCNATLOOK = IOCInOut | ((LEN & IOCPARMMask) << 16) | ('D' << 8) | 23
)

// OriginalDst uses ioctl to read original destination from /dev/pf
func OriginalDst(la, ra v2net.Addr) (v2net.IP, int, error) {
	f, err := os.Open("/dev/pf")
	if err != nil {
		return v2net.IP{}, -1, newError("failed to open device /dev/pf").Base(err)
	}
	defer f.Close()
	fd := f.Fd()
	nl := struct { // struct pfioc_natlook
		saddr, daddr, rsaddr, rdaddr       [16]byte
		sxport, dxport, rsxport, rdxport   [4]byte
		af, proto, protoVariant, direction uint8
	}{
		af:        syscall.AF_INET,
		proto:     syscall.IPPROTO_TCP,
		direction: PfOut,
	}
	raIP, laIP := ra.(*v2net.TCPAddr).IP, la.(*v2net.TCPAddr).IP
	raPort, laPort := ra.(*v2net.TCPAddr).Port, la.(*v2net.TCPAddr).Port
	if raIP.To4() != nil {
		if laIP.IsUnspecified() {
			laIP = v2net.ParseIP("127.0.0.1")
		}
		copy(nl.saddr[:v2net.IPv4len], raIP.To4())
		copy(nl.daddr[:v2net.IPv4len], laIP.To4())
	}
	if raIP.To16() != nil && raIP.To4() == nil {
		if laIP.IsUnspecified() {
			laIP = v2net.ParseIP("::1")
		}
		copy(nl.saddr[:], raIP)
		copy(nl.daddr[:], laIP)
	}
	nl.sxport[0], nl.sxport[1] = byte(raPort>>8), byte(raPort)
	nl.dxport[0], nl.dxport[1] = byte(laPort>>8), byte(laPort)
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, DIOCNATLOOK, uintptr(unsafe.Pointer(&nl))); errno != 0 {
		return v2net.IP{}, -1, os.NewSyscallError("ioctl", errno)
	}

	odPort := nl.rdxport
	var odIP v2net.IP
	switch nl.af {
	case syscall.AF_INET:
		odIP = make(v2net.IP, v2net.IPv4len)
		copy(odIP, nl.rdaddr[:v2net.IPv4len])
	case syscall.AF_INET6:
		odIP = make(v2net.IP, v2net.IPv6len)
		copy(odIP, nl.rdaddr[:])
	}
	return odIP, int(v2net.PortFromBytes(odPort[:2])), nil
}

func applyOutboundSocketOptions(network string, address string, fd uintptr, config *SocketConfig) error {
	if isTCPSocket(network) {
		switch config.Tfo {
		case SocketConfig_Enable:
			if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_FASTOPEN, TCP_FASTOPEN_CLIENT); err != nil {
				return err
			}
		case SocketConfig_Disable:
			if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_FASTOPEN, 0); err != nil {
				return err
			}
		}

		if config.TcpKeepAliveInterval > 0 {
			if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_KEEPINTVL, int(config.TcpKeepAliveInterval)); err != nil {
				return newError("failed to set TCP_KEEPINTVL").Base(err)
			}
		}
		if config.TcpKeepAliveIdle > 0 {
			if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_KEEPALIVE, int(config.TcpKeepAliveIdle)); err != nil {
				return newError("failed to set TCP_KEEPALIVE (TCP keepalive idle time on Darwin)").Base(err)
			}
		}

		if config.TcpKeepAliveInterval > 0 || config.TcpKeepAliveIdle > 0 {
			if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_KEEPALIVE, 1); err != nil {
				return newError("failed to set SO_KEEPALIVE").Base(err)
			}
		}
	}

	if config.BindToDevice != "" {
		iface, err := net.InterfaceByName(config.BindToDevice)
		if err != nil {
			return newError("failed to get interface ", config.BindToDevice).Base(err)
		}
		dest, err := v2net.ParseDestination(address)
		if err != nil {
			return err
		}
		switch {
		case dest.Address == v2net.AnyIP, dest.Address == v2net.AnyIPv6:
			_ = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_BOUND_IF, iface.Index)
			_ = unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_BOUND_IF, iface.Index)
		case dest.Address.Family().IsIPv4():
			if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_BOUND_IF, iface.Index); err != nil {
				return newError("failed to set IP_BOUND_IF", err)
			}
		case dest.Address.Family().IsIPv6():
			if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_BOUND_IF, iface.Index); err != nil {
				return newError("failed to set IPV6_BOUND_IF", err)
			}
		}
	}

	if config.TxBufSize != 0 {
		if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_SNDBUF, int(config.TxBufSize)); err != nil {
			return newError("failed to set SO_SNDBUF").Base(err)
		}
	}

	if config.RxBufSize != 0 {
		if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_RCVBUF, int(config.RxBufSize)); err != nil {
			return newError("failed to set SO_RCVBUF").Base(err)
		}
	}

	return nil
}

func applyInboundSocketOptions(network string, address string, fd uintptr, config *SocketConfig) error {
	if isTCPSocket(network) {
		switch config.Tfo {
		case SocketConfig_Enable:
			if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_FASTOPEN, TCP_FASTOPEN_SERVER); err != nil {
				return err
			}
		case SocketConfig_Disable:
			if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_FASTOPEN, 0); err != nil {
				return err
			}
		}
		if config.TcpKeepAliveInterval > 0 {
			if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_KEEPINTVL, int(config.TcpKeepAliveInterval)); err != nil {
				return newError("failed to set TCP_KEEPINTVL").Base(err)
			}
		}
		if config.TcpKeepAliveIdle > 0 {
			if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_KEEPALIVE, int(config.TcpKeepAliveIdle)); err != nil {
				return newError("failed to set TCP_KEEPALIVE (TCP keepalive idle time on Darwin)").Base(err)
			}
		}
		if config.TcpKeepAliveInterval > 0 || config.TcpKeepAliveIdle > 0 {
			if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_KEEPALIVE, 1); err != nil {
				return newError("failed to set SO_KEEPALIVE").Base(err)
			}
		}
	}

	if config.BindToDevice != "" {
		iface, err := net.InterfaceByName(config.BindToDevice)
		if err != nil {
			return newError("failed to get interface ", config.BindToDevice).Base(err)
		}
		dest, err := v2net.ParseDestination(address)
		if err != nil {
			return err
		}
		switch {
		case dest.Address == v2net.AnyIP, dest.Address == v2net.AnyIPv6:
			_ = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_BOUND_IF, iface.Index)
			_ = unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_BOUND_IF, iface.Index)
		case dest.Address.Family().IsIPv4():
			if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_BOUND_IF, iface.Index); err != nil {
				return newError("failed to set IP_BOUND_IF", err)
			}
		case dest.Address.Family().IsIPv6():
			if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_BOUND_IF, iface.Index); err != nil {
				return newError("failed to set IPV6_BOUND_IF", err)
			}
		}
	}

	if config.TxBufSize != 0 {
		if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_SNDBUF, int(config.TxBufSize)); err != nil {
			return newError("failed to set SO_SNDBUF/SO_SNDBUFFORCE").Base(err)
		}
	}

	if config.RxBufSize != 0 {
		if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_RCVBUF, int(config.RxBufSize)); err != nil {
			return newError("failed to set SO_RCVBUF/SO_RCVBUFFORCE").Base(err)
		}
	}
	return nil
}

func bindAddr(_ uintptr, _ []byte, _ uint32) error {
	return nil
}

func setReuseAddr(_ uintptr) error {
	return nil
}

func setReusePort(_ uintptr) error {
	return nil
}
