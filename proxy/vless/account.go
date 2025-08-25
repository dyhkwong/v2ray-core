package vless

import (
	"crypto/sha1"

	"github.com/v2fly/v2ray-core/v5/common/protocol"
	"github.com/v2fly/v2ray-core/v5/common/uuid"
)

// AsAccount implements protocol.Account.AsAccount().
func (a *Account) AsAccount() (protocol.Account, error) {
	id, err := uuid.ParseString(a.Id)
	if err != nil {
		hash := sha1.New()
		if _, err = hash.Write(id[:]); err != nil {
			return nil, err
		}
		if _, err = hash.Write([]byte(a.Id)); err != nil {
			return nil, err
		}
		u := hash.Sum(nil)[:16]
		u[6] = (u[6] & 0x0f) | (5 << 4)
		u[8] = (u[8]&(0xff>>2) | (0x02 << 6))
		copy(id[:], u)
	}
	return &MemoryAccount{
		ID:         protocol.NewID(id),
		Flow:       a.Flow,       // needs parser here?
		Encryption: a.Encryption, // needs parser here?
	}, nil
}

// MemoryAccount is an in-memory form of VLess account.
type MemoryAccount struct {
	// ID of the account.
	ID *protocol.ID
	// Flow of the account.
	Flow string
	// Encryption of the account. Used for client connections, and only accepts "none" for now.
	Encryption string
}

// Equals implements protocol.Account.Equals().
func (a *MemoryAccount) Equals(account protocol.Account) bool {
	vlessAccount, ok := account.(*MemoryAccount)
	if !ok {
		return false
	}
	return a.ID.Equals(vlessAccount.ID)
}
