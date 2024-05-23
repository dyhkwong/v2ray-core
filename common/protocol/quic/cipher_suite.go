package quic

import (
	"crypto/cipher"
	_ "crypto/tls"
	_ "unsafe"
)

// github.com/quic-go/quic-go/internal/handshake/cipher_suite.go describes these cipher suite implementations are copied from the standard library crypto/tls package.
// So we can user go:linkname to implement the same feature.

//go:linkname aeadAESGCMTLS13 crypto/tls.aeadAESGCMTLS13
func aeadAESGCMTLS13(key, nonceMask []byte) cipher.AEAD
