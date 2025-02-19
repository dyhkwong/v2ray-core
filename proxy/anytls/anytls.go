package anytls

import (
	"github.com/v2fly/v2ray-core/v5/common/protocol"
)

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

// MemoryAccount is aPasswordn account type converted from Account.
type MemoryAccount struct {
	Password string
	Email    string
	Level    int32
}

// AsAccount implements protocol.AsAccount.
func (u *User) AsAccount() (protocol.Account, error) {
	return &MemoryAccount{
		Password: u.GetPassword(),
		Email:    u.GetEmail(),
		Level:    u.GetLevel(),
	}, nil
}

// Equals implements protocol.Account.Equals().
func (a *MemoryAccount) Equals(another protocol.Account) bool {
	if account, ok := another.(*MemoryAccount); ok {
		return a.Password == account.Password
	}
	return false
}
