package seed

import (
	"crypto/cipher"
	"errors"
	"strconv"
)

type seed128Cipher struct {
	pdwRoundKey []uint32
}

// KeySizeError is Invalid Key Size Error.
type KeySizeError int

func (k KeySizeError) Error() string {
	return "github.com/geeksbaek/seed128: Invalid key size " + strconv.Itoa(int(k))
}

// NewCipher creates and returns a new cipher.Block.
// The key argument should be 16 bytes.
func NewCipher(key []byte) (cipher.Block, error) {
	k := len(key)
	switch k {
	case 16:
		break
	case 32:
		return nil, errors.New("github.com/geeksbaek/seed128: Unsupported key size 32")
	default:
		return nil, KeySizeError(k)
	}
	return newCipherGeneric(key)
}

func newCipherGeneric(key []byte) (cipher.Block, error) {
	n := len(key) + 28
	c := seed128Cipher{make([]uint32, n)}
	c.pdwRoundKey = seedRoundKey(key)
	return &c, nil
}

func (c *seed128Cipher) BlockSize() int { return BlockSize }

func (c *seed128Cipher) Encrypt(dst, src []byte) {
	if len(src) < BlockSize {
		panic("github.com/geeksbaek/seed128: input not full block")
	}
	if len(dst) < BlockSize {
		panic("github.com/geeksbaek/seed128: output not full block")
	}
	seedEncrypt(c.pdwRoundKey, dst, src)
}

func (c *seed128Cipher) Decrypt(dst, src []byte) {
	if len(src) < BlockSize {
		panic("github.com/geeksbaek/seed128: input not full block")
	}
	if len(dst) < BlockSize {
		panic("github.com/geeksbaek/seed128: output not full block")
	}
	seedDecrypt(c.pdwRoundKey, dst, src)
}
