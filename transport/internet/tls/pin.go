package tls

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
)

func CalculatePEMCertChainSHA256Hash(certContent []byte) string {
	var certChain [][]byte
	for {
		block, remain := pem.Decode(certContent)
		if block == nil {
			break
		}
		certChain = append(certChain, block.Bytes)
		certContent = remain
	}
	certChainHash := GenerateCertChainHash(certChain)
	certChainHashB64 := base64.StdEncoding.EncodeToString(certChainHash)
	return certChainHashB64
}

func CalculatePEMCertPublicKeySHA256Hash(certContent []byte) (string, error) {
	block, _ := pem.Decode(certContent)
	if block == nil {
		return "", newError("invalid certificate")
	}
	certPublicKeyHash, err := GenerateCertPublicKeyHash(block.Bytes)
	if err != nil {
		return "", newError("invalid certificate")
	}
	return base64.StdEncoding.EncodeToString(certPublicKeyHash), nil
}

func CalculatePEMCertSHA256Hash(certContent []byte) (string, error) {
	block, _ := pem.Decode(certContent)
	if block == nil {
		return "", newError("invalid certificate")
	}
	return hex.EncodeToString(GenerateCertHash(block.Bytes)), nil
}

func GenerateCertChainHash(rawCerts [][]byte) []byte {
	var hashValue []byte
	for _, certValue := range rawCerts {
		out := sha256.Sum256(certValue)
		if hashValue == nil {
			hashValue = out[:]
		} else {
			newHashValue := sha256.Sum256(append(hashValue, out[:]...))
			hashValue = newHashValue[:]
		}
	}
	return hashValue
}

func GenerateCertPublicKeyHash(rawCert []byte) ([]byte, error) {
	cert, err := x509.ParseCertificate(rawCert)
	if err != nil {
		return nil, newError("invalid certificate")
	}
	publicKey, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return nil, newError("invalid public key")
	}
	hashValue := sha256.Sum256(publicKey)
	return hashValue[:], nil
}

func GenerateCertHash(rawCert []byte) []byte {
	hashValue := sha256.Sum256(rawCert)
	return hashValue[:]
}
