package internet

import (
	"encoding/binary"
	"net"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	v2net "github.com/v2fly/v2ray-core/v5/common/net"
)

const (
	TCP_FASTOPEN    = 15 // nolint: revive,stylecheck
	IP_UNICAST_IF   = 31 // nolint: revive,stylecheck
	IPV6_UNICAST_IF = 31 // nolint: revive,stylecheck
)

func setTFO(fd syscall.Handle, settings SocketConfig_TCPFastOpenState) error {
	switch settings {
	case SocketConfig_Enable:
		if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_FASTOPEN, 1); err != nil {
			return err
		}
	case SocketConfig_Disable:
		if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_FASTOPEN, 0); err != nil {
			return err
		}
	}
	return nil
}

func applyOutboundSocketOptions(network string, address string, fd uintptr, config *SocketConfig) error {
	if isTCPSocket(network) {
		if err := setTFO(syscall.Handle(fd), config.Tfo); err != nil {
			return err
		}
		if config.TcpKeepAliveIdle > 0 {
			if err := syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 1); err != nil {
				return newError("failed to set SO_KEEPALIVE", err)
			}
		}
	}

	if config.BindToDevice != "" {
		iface, err := net.InterfaceByName(config.BindToDevice)
		if err != nil {
			return newError("failed to get interface ", config.BindToDevice).Base(err)
		}
		dest, err := v2net.ParseDestination(address)
		multiCast := err == nil && dest.Address.Family().IsIP() && dest.Address.IP().IsMulticast()
		// bind is always used in the default dialer
		switch network {
		case "tcp4", "udp4":
			var bytes [4]byte
			binary.BigEndian.PutUint32(bytes[:], uint32(iface.Index))
			index := *(*uint32)(unsafe.Pointer(&bytes[0]))
			if err := windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IP, IP_UNICAST_IF, int(index)); err != nil {
				return newError("failed to set IP_UNICAST_IF", err)
			}
			if network == "udp4" || multiCast {
				if err := windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IP, windows.IP_MULTICAST_IF, int(index)); err != nil {
					return newError("failed to set IP_MULTICAST_IF", err)
				}
			}
		case "tcp6", "udp6":
			if err := windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IPV6, IPV6_UNICAST_IF, iface.Index); err != nil {
				return newError("failed to set IPV6_UNICAST_IF", err)
			}
			if network == "udp6" || multiCast {
				if err := windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IPV6, windows.IPV6_MULTICAST_IF, iface.Index); err != nil {
					return newError("failed to set IPV6_MULTICAST_IF", err)
				}
			}
		}
	}

	if config.TxBufSize != 0 {
		if err := windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_SNDBUF, int(config.TxBufSize)); err != nil {
			return newError("failed to set SO_SNDBUF").Base(err)
		}
	}

	if config.RxBufSize != 0 {
		if err := windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_RCVBUF, int(config.TxBufSize)); err != nil {
			return newError("failed to set SO_RCVBUF").Base(err)
		}
	}

	return nil
}

func applyInboundSocketOptions(network string, address string, fd uintptr, config *SocketConfig) error {
	if isTCPSocket(network) {
		if err := setTFO(syscall.Handle(fd), config.Tfo); err != nil {
			return err
		}
		if config.TcpKeepAliveIdle > 0 {
			if err := syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 1); err != nil {
				return newError("failed to set SO_KEEPALIVE", err)
			}
		}
	}

	if config.BindToDevice != "" {
		iface, err := net.InterfaceByName(config.BindToDevice)
		if err != nil {
			return newError("failed to get interface ", config.BindToDevice).Base(err)
		}
		dest, err := v2net.ParseDestination(address)
		multiCast := err == nil && dest.Address.Family().IsIP() && dest.Address.IP().IsMulticast()
		// bind is always used in the default dialer
		switch network {
		case "tcp4", "udp4":
			var bytes [4]byte
			binary.BigEndian.PutUint32(bytes[:], uint32(iface.Index))
			index := *(*uint32)(unsafe.Pointer(&bytes[0]))
			if network == "udp4" || multiCast {
				if err := windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IP, windows.IP_MULTICAST_IF, int(index)); err != nil {
					return newError("failed to set IP_MULTICAST_IF", err)
				}
			}
			if network == "udp4" || !multiCast {
				if err := windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IP, IP_UNICAST_IF, int(index)); err != nil {
					return newError("failed to set IP_UNICAST_IF", err)
				}
			}
		case "tcp6", "udp6":
			if network == "udp6" || multiCast {
				if err := windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IPV6, windows.IPV6_MULTICAST_IF, iface.Index); err != nil {
					return newError("failed to set IPV6_MULTICAST_IF", err)
				}
			}
			if network == "udp6" || !multiCast {
				if err := windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IPV6, IPV6_UNICAST_IF, iface.Index); err != nil {
					return newError("failed to set IPV6_UNICAST_IF", err)
				}
			}
		}
	}

	if config.TxBufSize != 0 {
		if err := windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_SNDBUF, int(config.TxBufSize)); err != nil {
			return newError("failed to set SO_SNDBUF").Base(err)
		}
	}

	if config.RxBufSize != 0 {
		if err := windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_RCVBUF, int(config.TxBufSize)); err != nil {
			return newError("failed to set SO_RCVBUF").Base(err)
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
