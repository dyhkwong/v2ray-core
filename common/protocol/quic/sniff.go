package quic

import (
	"crypto"
	"crypto/aes"
	"encoding/binary"
	"io"

	"golang.org/x/crypto/hkdf"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/errors"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
	ptls "github.com/v2fly/v2ray-core/v5/common/protocol/tls"
	"github.com/v2fly/v2ray-core/v5/common/rangelist"
)

type SniffHeader struct {
	domain string
}

func (s SniffHeader) Protocol() string {
	return "quic"
}

func (s SniffHeader) Domain() string {
	return s.domain
}

const (
	versionDraft29 uint32 = 0xff00001d
	version1       uint32 = 0x1
	version2       uint32 = 0x6b3343cf
)

var (
	quicSaltOld       = []byte{0xaf, 0xbf, 0xec, 0x28, 0x99, 0x93, 0xd2, 0x4c, 0x9e, 0x97, 0x86, 0xf1, 0x9c, 0x61, 0x11, 0xe0, 0x43, 0x90, 0xa8, 0x99}
	quicSaltV1        = []byte{0x38, 0x76, 0x2c, 0xf7, 0xf5, 0x59, 0x34, 0xb3, 0x4d, 0x17, 0x9a, 0xe6, 0xa4, 0xc8, 0x0c, 0xad, 0xcc, 0xbb, 0x7f, 0x0a}
	quicSaltV2        = []byte{0x0d, 0xed, 0xe3, 0xde, 0xf7, 0x00, 0xa6, 0xdb, 0x81, 0x93, 0x81, 0xbe, 0x6e, 0x26, 0x9d, 0xcb, 0xf9, 0xbd, 0x2e, 0xd9}
	errNotQuic        = errors.New("not quic")
	errNotQuicInitial = errors.New("not initial packet")
)

const (
	hkdfLabelKeyV1              = "quic key"
	hkdfLabelKeyV2              = "quicv2 key"
	hkdfLabelIVV1               = "quic iv"
	hkdfLabelIVV2               = "quicv2 iv"
	hkdfLabelHeaderProtectionV1 = "quic hp"
	hkdfLabelHeaderProtectionV2 = "quicv2 hp"
)

func SniffQUIC(b []byte) (*SniffHeader, error) {
	if len(b) == 0 {
		return nil, common.ErrNoClue
	}

	// Crypto data separated across packets
	cryptoLen := 0
	cryptoDataBuf := buf.NewWithSize(32767)
	defer cryptoDataBuf.Release()

	cache := buf.New()
	defer cache.Release()

	validRange := rangelist.NewRangeList()
	// Parse QUIC packets
	for len(b) > 0 {
		buffer := buf.FromBytes(b)
		typeByte, err := buffer.ReadByte()
		if err != nil {
			return nil, errNotQuic
		}

		isLongHeader := typeByte&0x80 > 0
		if !isLongHeader || typeByte&0x40 == 0 {
			return nil, errNotQuicInitial
		}

		vb, err := buffer.ReadBytes(4)
		if err != nil {
			return nil, errNotQuic
		}

		versionNumber := binary.BigEndian.Uint32(vb)
		if versionNumber != 0 && typeByte&0x40 == 0 {
			return nil, errNotQuic
		} else if versionNumber != versionDraft29 && versionNumber != version1 && versionNumber != version2 {
			return nil, errNotQuic
		}

		packetType := (typeByte & 0x30) >> 4
		isQuicInitial := packetType == 0x0

		var destConnID []byte
		if l, err := buffer.ReadByte(); err != nil {
			return nil, errNotQuic
		} else if destConnID, err = buffer.ReadBytes(int32(l)); err != nil {
			return nil, errNotQuic
		}

		if l, err := buffer.ReadByte(); err != nil {
			return nil, errNotQuic
		} else if common.Error2(buffer.ReadBytes(int32(l))) != nil {
			return nil, errNotQuic
		}

		if isQuicInitial { // Only initial packets have token, see https://datatracker.ietf.org/doc/html/rfc9000#section-17.2.2
			tokenLen, err := readUvarint(buffer)
			if err != nil || tokenLen > uint64(len(b)) {
				return nil, errNotQuic
			}

			if _, err = buffer.ReadBytes(int32(tokenLen)); err != nil {
				return nil, errNotQuic
			}
		}

		packetLen, err := readUvarint(buffer)
		if err != nil {
			return nil, errNotQuic
		}
		// packet is impossible to shorter than this
		if packetLen < 4 /* length of origPNBytes */ {
			return nil, errNotQuic
		}

		hdrLen := len(b) - int(buffer.Len())
		if len(b) < hdrLen+int(packetLen) {
			return nil, common.ErrNoClue // Not enough data to read as a QUIC packet. QUIC is UDP-based, so this is unlikely to happen.
		}

		restPayload := b[hdrLen+int(packetLen):]
		if !isQuicInitial { // Skip this packet if it's not initial packet
			b = restPayload
			continue
		}

		origPNBytes := make([]byte, 4)
		copy(origPNBytes, b[hdrLen:hdrLen+4])

		var salt []byte
		switch versionNumber {
		case version1:
			salt = quicSaltV1
		case version2:
			salt = quicSaltV2
		default:
			salt = quicSaltOld
		}

		var hkdfHeaderProtectionLabel string
		switch versionNumber {
		case version2:
			hkdfHeaderProtectionLabel = hkdfLabelHeaderProtectionV2
		default:
			hkdfHeaderProtectionLabel = hkdfLabelHeaderProtectionV1
		}

		initialSecret := hkdf.Extract(crypto.SHA256.New, destConnID, salt)
		secret := hkdfExpandLabel(crypto.SHA256, initialSecret, []byte{}, "client in", crypto.SHA256.Size())
		hpKey := hkdfExpandLabel(crypto.SHA256, secret, []byte{}, hkdfHeaderProtectionLabel, 16)
		block, err := aes.NewCipher(hpKey)
		if err != nil {
			return nil, err
		}

		cache.Clear()
		mask := cache.Extend(int32(block.BlockSize()))
		if int(packetLen) < 4+block.BlockSize() {
			return nil, errNotQuic
		}
		block.Encrypt(mask, b[hdrLen+4:hdrLen+4+block.BlockSize()])
		b[0] ^= mask[0] & 0xf
		for i := range b[hdrLen : hdrLen+4] {
			b[hdrLen+i] ^= mask[i+1]
		}
		packetNumberLength := b[0]&0x3 + 1
		if int(packetLen) < int(packetNumberLength) {
			return nil, errNotQuic
		}
		var packetNumber uint32
		switch packetNumberLength {
		case 1:
			packetNumber = uint32(b[hdrLen])
		case 2:
			packetNumber = uint32(binary.BigEndian.Uint16(b[hdrLen:]))
		case 3:
			packetNumber = uint32(b[hdrLen+2]) | uint32(b[hdrLen+1])<<8 | uint32(b[hdrLen])<<16
		case 4:
			packetNumber = binary.BigEndian.Uint32(b[hdrLen:])
		default:
			return nil, errNotQuicInitial
		}

		extHdrLen := hdrLen + int(packetNumberLength)
		copy(b[extHdrLen:hdrLen+4], origPNBytes[packetNumberLength:])
		data := b[extHdrLen : int(packetLen)+hdrLen]

		var keyLabel string
		var ivLabel string
		switch versionNumber {
		case version2:
			keyLabel = hkdfLabelKeyV2
			ivLabel = hkdfLabelIVV2
		default:
			keyLabel = hkdfLabelKeyV1
			ivLabel = hkdfLabelIVV1
		}

		key := hkdfExpandLabel(crypto.SHA256, secret, []byte{}, keyLabel, 16)
		iv := hkdfExpandLabel(crypto.SHA256, secret, []byte{}, ivLabel, 12)
		cipher := aeadAESGCMTLS13(key, iv)
		nonce := cache.Extend(int32(cipher.NonceSize()))
		binary.BigEndian.PutUint64(nonce[len(nonce)-8:], uint64(packetNumber))
		decrypted, err := cipher.Open(b[extHdrLen:extHdrLen], nonce, data, b[:extHdrLen])
		if err != nil {
			return nil, err
		}
		buffer = buf.FromBytes(decrypted)
		for i := 0; !buffer.IsEmpty(); i++ {
			frameType := byte(0x0) // Default to PADDING frame
			for frameType == 0x0 && !buffer.IsEmpty() {
				frameType, err = buffer.ReadByte()
				if err != nil {
					return nil, io.ErrUnexpectedEOF
				}
			}
			switch frameType {
			case 0x00: // PADDING frame
			case 0x01: // PING frame
			case 0x02, 0x03: // ACK frame
				if _, err = readUvarint(buffer); err != nil { // Field: Largest Acknowledged
					return nil, io.ErrUnexpectedEOF
				}
				if _, err = readUvarint(buffer); err != nil { // Field: ACK Delay
					return nil, io.ErrUnexpectedEOF
				}
				ackRangeCount, err := readUvarint(buffer) // Field: ACK Range Count
				if err != nil {
					return nil, io.ErrUnexpectedEOF
				}
				if _, err = readUvarint(buffer); err != nil { // Field: First ACK Range
					return nil, io.ErrUnexpectedEOF
				}
				for i := 0; i < int(ackRangeCount); i++ { // Field: ACK Range
					if _, err = readUvarint(buffer); err != nil { // Field: ACK Range -> Gap
						return nil, io.ErrUnexpectedEOF
					}
					if _, err = readUvarint(buffer); err != nil { // Field: ACK Range -> ACK Range Length
						return nil, io.ErrUnexpectedEOF
					}
				}
				if frameType == 0x03 {
					if _, err = readUvarint(buffer); err != nil { // Field: ECN Counts -> ECT0 Count
						return nil, io.ErrUnexpectedEOF
					}
					if _, err = readUvarint(buffer); err != nil { // Field: ECN Counts -> ECT1 Count
						return nil, io.ErrUnexpectedEOF
					}
					if _, err = readUvarint(buffer); err != nil { //nolint:misspell // Field: ECN Counts -> ECT-CE Count
						return nil, io.ErrUnexpectedEOF
					}
				}
			case 0x06: // CRYPTO frame, we will use this frame
				offset, err := readUvarint(buffer) // Field: Offset
				if err != nil {
					return nil, io.ErrUnexpectedEOF
				}
				length, err := readUvarint(buffer) // Field: Length
				if err != nil || length > uint64(buffer.Len()) {
					return nil, io.ErrUnexpectedEOF
				}
				if cryptoLen < int(offset+length) {
					cryptoLen = int(offset + length)
					if cryptoDataBuf.Cap() < int32(cryptoLen) {
						return nil, io.ErrShortBuffer
					}
					if cryptoDataBuf.Len() != int32(cryptoLen) {
						cryptoDataBuf.Extend(int32(cryptoLen) - cryptoDataBuf.Len())
					}
				}
				if _, err := buffer.Read(cryptoDataBuf.BytesRange(int32(offset), int32(offset+length))); err != nil { // Field: Crypto Data
					return nil, io.ErrUnexpectedEOF
				}
				validRange.Add(int(offset), int(offset+length))
			case 0x1c: // CONNECTION_CLOSE frame, only 0x1c is permitted in initial packet
				if _, err = readUvarint(buffer); err != nil { // Field: Error Code
					return nil, io.ErrUnexpectedEOF
				}
				if _, err = readUvarint(buffer); err != nil { // Field: Frame Type
					return nil, io.ErrUnexpectedEOF
				}
				length, err := readUvarint(buffer) // Field: Reason Phrase Length
				if err != nil {
					return nil, io.ErrUnexpectedEOF
				}
				if _, err := buffer.ReadBytes(int32(length)); err != nil { // Field: Reason Phrase
					return nil, io.ErrUnexpectedEOF
				}
			default:
				// Only above frame types are permitted in initial packet.
				// See https://www.rfc-editor.org/rfc/rfc9000.html#section-17.2.2-8
				return nil, errNotQuicInitial
			}
		}

		tlsHdr := &ptls.SniffHeader{}
		err = ptls.ReadClientHello(cryptoDataBuf.BytesRange(0, int32(cryptoLen)), tlsHdr, validRange)
		if err != nil {
			// The crypto data may have not been fully recovered in current packets,
			// So we continue to sniff rest packets.
			b = restPayload
			continue
		}
		return &SniffHeader{domain: tlsHdr.Domain()}, nil
	}

	// All payload is parsed as valid QUIC packets, but we need more packets for crypto data to read client hello.
	return nil, protocol.ErrProtoNeedMoreData
}

func hkdfExpandLabel(hash crypto.Hash, secret, context []byte, label string, length int) []byte { //nolint:unparam
	b := make([]byte, 3, 3+6+len(label)+1+len(context))
	binary.BigEndian.PutUint16(b, uint16(length))
	b[2] = uint8(6 + len(label))
	b = append(b, []byte("tls13 ")...)
	b = append(b, []byte(label)...)
	b = b[:3+6+len(label)+1]
	b[3+6+len(label)] = uint8(len(context))
	b = append(b, context...)

	out := make([]byte, length)
	n, err := hkdf.Expand(hash.New, secret, b).Read(out)
	if err != nil || n != length {
		panic("quic: HKDF-Expand-Label invocation failed unexpectedly")
	}
	return out
}

func readUvarint(r io.ByteReader) (uint64, error) {
	firstByte, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	// the first two bits of the first byte encode the length
	l := 1 << ((firstByte & 0xc0) >> 6)
	b1 := firstByte & (0xff - 0xc0)
	if l == 1 {
		return uint64(b1), nil
	}
	b2, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	if l == 2 {
		return uint64(b2) + uint64(b1)<<8, nil
	}
	b3, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	b4, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	if l == 4 {
		return uint64(b4) + uint64(b3)<<8 + uint64(b2)<<16 + uint64(b1)<<24, nil
	}
	b5, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	b6, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	b7, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	b8, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	return uint64(b8) + uint64(b7)<<8 + uint64(b6)<<16 + uint64(b5)<<24 + uint64(b4)<<32 + uint64(b3)<<40 + uint64(b2)<<48 + uint64(b1)<<56, nil
}
