package common

import (
	"bytes"
	"io"

	"github.com/v2fly/v2ray-core/v5/common/antireplay"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
)

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

// MemoryAccount is an account type converted from Account.
type MemoryAccount struct {
	Cipher Cipher
	Key    []byte

	ReplayFilter antireplay.GeneralizedReplayFilter

	ReducedIVEntropy bool
}

// Equals implements protocol.Account.Equals().
func (a *MemoryAccount) Equals(another protocol.Account) bool {
	if account, ok := another.(*MemoryAccount); ok {
		return bytes.Equal(a.Key, account.Key)
	}
	return false
}

func (a *MemoryAccount) CheckIV(iv []byte) error {
	if a.ReplayFilter == nil {
		return nil
	}
	if a.ReplayFilter.Check(iv) {
		return nil
	}
	return newError("IV is not unique")
}

// Cipher is an interface for all Shadowsocks ciphers.
type Cipher interface {
	KeySize() int32
	IVSize() int32
	NewEncryptionWriter(key []byte, iv []byte, writer io.Writer) (buf.Writer, error)
	NewDecryptionReader(key []byte, iv []byte, reader io.Reader) (buf.Reader, error)
	IsAEAD() bool
	EncodePacket(key []byte, b *buf.Buffer) error
	DecodePacket(key []byte, b *buf.Buffer) error
}
