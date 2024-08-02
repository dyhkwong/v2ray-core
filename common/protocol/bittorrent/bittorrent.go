package bittorrent

import (
	"encoding/binary"
	"errors"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
)

type SniffHeader struct{}

func (h *SniffHeader) Protocol() string {
	return "bittorrent"
}

func (h *SniffHeader) Domain() string {
	return ""
}

var errNotBittorrent = errors.New("not bittorrent header")

func SniffBittorrent(b []byte) (*SniffHeader, error) {
	if len(b) < 20 {
		return nil, common.ErrNoClue
	}

	if b[0] == 19 && string(b[1:20]) == "BitTorrent protocol" {
		return &SniffHeader{}, nil
	}

	return nil, errNotBittorrent
}

func SniffUTP(b []byte) (*SniffHeader, error) {
	if len(b) < 20 {
		return nil, common.ErrNoClue
	}

	buffer := buf.FromBytes(b)

	var typeAndVersion uint8

	if binary.Read(buffer, binary.BigEndian, &typeAndVersion) != nil {
		return nil, common.ErrNoClue
	} else if b[0]>>4&0xF > 4 || b[0]&0xF != 1 {
		return nil, errNotBittorrent
	}

	extension := b[1]

	buffer = buf.FromBytes(b[20:])

	for extension != 0 {
		if binary.Read(buffer, binary.BigEndian, &extension) != nil {
			return nil, common.ErrNoClue
		}
		var length uint8
		if err := binary.Read(buffer, binary.BigEndian, &length); err != nil {
			return nil, common.ErrNoClue
		}
		if common.Error2(buffer.ReadBytes(int32(length))) != nil {
			return nil, common.ErrNoClue
		}
	}

	return &SniffHeader{}, nil
}

func SniffUDPTracker(b []byte) (*SniffHeader, error) {
	if len(b) < 16 {
		return nil, common.ErrNoClue
	}
	if binary.BigEndian.Uint64(b[:8]) != 0x41727101980 {
		return nil, common.ErrNoClue
	}
	if binary.BigEndian.Uint32(b[8:12]) != 0 {
		return nil, common.ErrNoClue
	}
	return &SniffHeader{}, nil
}
