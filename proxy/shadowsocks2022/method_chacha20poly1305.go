package shadowsocks2022

import (
	"crypto/cipher"

	"golang.org/x/crypto/chacha20poly1305"
)

func newChacha20Poly1305Method() *Chacha20Poly1305Method {
	return &Chacha20Poly1305Method{}
}

type Chacha20Poly1305Method struct{}

func (a Chacha20Poly1305Method) GetSessionSubKeyAndSaltLength() int {
	return 32
}

func (a Chacha20Poly1305Method) GetStreamAEAD(sessionSubKey []byte) (cipher.AEAD, error) {
	aead, err := chacha20poly1305.New(sessionSubKey)
	if err != nil {
		return nil, newError("failed to create Chacha20-Poly1305").Base(err)
	}
	return aead, nil
}

func (a Chacha20Poly1305Method) GenerateEIH(currentIdentitySubKey []byte, nextPskHash []byte, out []byte) error {
	return newError("Chacha20-Poly1305 does not support EIH")
}

func (a Chacha20Poly1305Method) GetUDPClientProcessor(_ [][]byte, psk []byte, derivation KeyDerivation) (UDPClientPacketProcessor, error) {
	aead, err := chacha20poly1305.NewX(psk)
	if err != nil {
		return nil, newError("failed to create XChacha20-Poly1305").Base(err)
	}
	return NewChacha20Poly1305UDPClientPacketProcessor(aead), nil
}
