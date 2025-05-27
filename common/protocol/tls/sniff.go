package tls

import (
	"encoding/binary"
	"errors"
	"io"
	"strings"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
	"github.com/v2fly/v2ray-core/v5/common/rangelist"
)

type SniffHeader struct {
	domain string
}

func (h *SniffHeader) Protocol() string {
	return "tls"
}

func (h *SniffHeader) Domain() string {
	return h.domain
}

var (
	errNotTLS         = errors.New("not TLS header")
	errNotClientHello = errors.New("not client hello")
)

func IsValidTLSVersion(major, minor byte) bool {
	return major == 3
}

// ReadClientHello returns server name (if any) from TLS client hello message.
// https://github.com/golang/go/blob/master/src/crypto/tls/handshake_messages.go#L300
func ReadClientHello(data []byte, h *SniffHeader, validRange *rangelist.RangeList) error {
	if len(data) < 42 {
		return protocol.ErrProtoNeedMoreData
	}
	sessionIDLen := int(data[38])
	if sessionIDLen > 32 || len(data) < 39+sessionIDLen {
		return protocol.ErrProtoNeedMoreData
	}
	data = data[39+sessionIDLen:]
	if len(data) < 2 {
		return protocol.ErrProtoNeedMoreData
	}
	// cipherSuiteLen is the number of bytes of cipher suite numbers. Since
	// they are uint16s, the number must be even.
	cipherSuiteLen := int(data[0])<<8 | int(data[1])
	if cipherSuiteLen%2 == 1 {
		return errNotClientHello
	}
	if len(data) < 2+cipherSuiteLen {
		return protocol.ErrProtoNeedMoreData
	}
	data = data[2+cipherSuiteLen:]
	if len(data) < 1 {
		return protocol.ErrProtoNeedMoreData
	}
	compressionMethodsLen := int(data[0])
	if len(data) < 1+compressionMethodsLen {
		return protocol.ErrProtoNeedMoreData
	}
	data = data[1+compressionMethodsLen:]

	if len(data) < 2 {
		return protocol.ErrProtoNeedMoreData
	}

	extensionsLength := int(data[0])<<8 | int(data[1])
	data = data[2:]
	dataLen := len(data)

	offset := 44 + sessionIDLen + cipherSuiteLen + compressionMethodsLen
	for len(data) != 0 {
		if len(data) < 4 {
			if dataLen == extensionsLength {
				return errNotClientHello
			}
			return protocol.ErrProtoNeedMoreData
		}
		extension := uint16(data[0])<<8 | uint16(data[1])
		length := int(data[2])<<8 | int(data[3])
		data = data[4:]
		offset += 4
		if len(data) < length {
			if dataLen == extensionsLength {
				return errNotClientHello
			}
			return protocol.ErrProtoNeedMoreData
		}

		if extension == 0x00 { /* extensionServerName */
			d := data[:length]
			if len(d) < 2 {
				if dataLen == extensionsLength {
					return errNotClientHello
				}
				return protocol.ErrProtoNeedMoreData
			}
			namesLen := int(d[0])<<8 | int(d[1])
			d = d[2:]
			if len(d) != namesLen {
				if dataLen == extensionsLength {
					return errNotClientHello
				}
				return protocol.ErrProtoNeedMoreData
			}
			for len(d) > 0 {
				if len(d) < 3 {
					if dataLen == extensionsLength {
						return errNotClientHello
					}
					return protocol.ErrProtoNeedMoreData
				}
				nameType := d[0]
				nameLen := int(d[1])<<8 | int(d[2])
				d = d[3:]
				if len(d) < nameLen {
					if dataLen == extensionsLength {
						return errNotClientHello
					}
					return protocol.ErrProtoNeedMoreData
				}
				if nameType == 0 {
					if validRange != nil && !validRange.In(offset+5, offset+5+nameLen) {
						return protocol.ErrProtoNeedMoreData
					}
					serverName := string(d[:nameLen])
					// An SNI value may not include a
					// trailing dot. See
					// https://tools.ietf.org/html/rfc6066#section-3.
					if strings.HasSuffix(serverName, ".") {
						return errNotClientHello
					}
					h.domain = serverName
					return nil
				}
				d = d[nameLen:]
			}
		}
		data = data[length:]
		offset += length
	}

	if dataLen == extensionsLength {
		return nil
	}
	return protocol.ErrProtoNeedMoreData
}

func SniffTLS(b []byte) (*SniffHeader, error) {
	if len(b) < 5 {
		return nil, common.ErrNoClue
	}

	if b[0] != 0x16 /* TLS Handshake */ {
		return nil, errNotTLS
	}
	if !IsValidTLSVersion(b[1], b[2]) {
		return nil, errNotTLS
	}
	headerLen := int(binary.BigEndian.Uint16(b[3:5]))
	if headerLen == 0 || headerLen >= 16384 {
		// Implementations MUST NOT send zero-length fragments
		// The length MUST NOT exceed 2^14
		return nil, errNotTLS
	}

	h := &SniffHeader{}
	data := buf.NewWithSize(32767)
	defer data.Release()
	for len(b) > 0 {
		if data.Cap() < int32(len(b)) {
			return nil, io.ErrShortBuffer
		}
		data.Write(b[5:min(5+headerLen, len(b))])
		err := ReadClientHello(data.Bytes(), h, nil)
		if err == nil {
			return h, nil
		}
		if err == errNotTLS || err == errNotClientHello {
			return nil, err
		}
		b = b[min(5+headerLen, len(b)):]

		if len(b) < 5 {
			return nil, protocol.ErrProtoNeedMoreData
		}

		if b[0] != 0x16 /* TLS Handshake */ {
			return nil, errNotTLS
		}
		if !IsValidTLSVersion(b[1], b[2]) {
			return nil, errNotTLS
		}
		headerLen := int(binary.BigEndian.Uint16(b[3:5]))
		if headerLen == 0 || headerLen >= 16384 {
			// Implementations MUST NOT send zero-length fragments
			// The length MUST NOT exceed 2^14
			return nil, errNotTLS
		}
	}
	return nil, protocol.ErrProtoNeedMoreData
}
