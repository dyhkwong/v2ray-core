package reality

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	gotls "crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"time"
	"unsafe"

	utls "github.com/refraction-networking/utls"
	"github.com/xtls/reality"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/net/http2"

	"github.com/v2fly/v2ray-core/v5/common/dice"
	"github.com/v2fly/v2ray-core/v5/common/net"
)

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

type Conn struct {
	*reality.Conn
}

func (c *Conn) HandshakeAddress() net.Address {
	if err := c.Handshake(); err != nil {
		return nil
	}
	state := c.ConnectionState()
	if state.ServerName == "" {
		return nil
	}
	return net.ParseAddress(state.ServerName)
}

func Server(conn net.Conn, config *reality.Config) (net.Conn, error) {
	realityConn, err := reality.Server(context.Background(), conn, config)
	if err != nil {
		return nil, err
	}
	return &Conn{Conn: realityConn}, nil
}

type UConn struct {
	*utls.UConn
	ServerName string
	AuthKey    []byte
	Verified   bool
}

func (c *UConn) HandshakeAddress() net.Address {
	if err := c.Handshake(); err != nil {
		return nil
	}
	state := c.ConnectionState()
	if state.ServerName == "" {
		return nil
	}
	return net.ParseAddress(state.ServerName)
}

func (c *UConn) VerifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	p, _ := reflect.TypeOf(c.Conn).Elem().FieldByName("peerCertificates")
	certs := *(*([]*x509.Certificate))(unsafe.Pointer(uintptr(unsafe.Pointer(c.Conn)) + p.Offset))
	if pub, ok := certs[0].PublicKey.(ed25519.PublicKey); ok {
		h := hmac.New(sha512.New, c.AuthKey)
		h.Write(pub)
		if bytes.Equal(h.Sum(nil), certs[0].Signature) {
			c.Verified = true
			return nil
		}
	}
	opts := x509.VerifyOptions{
		DNSName:       c.ServerName,
		Intermediates: x509.NewCertPool(),
	}
	for _, cert := range certs[1:] {
		opts.Intermediates.AddCert(cert)
	}
	if _, err := certs[0].Verify(opts); err != nil {
		return err
	}
	return nil
}

func UClient(ctx context.Context, conn net.Conn, dest net.Destination, config *Config) (net.Conn, error) {
	uConn := &UConn{}
	utlsConfig := &utls.Config{
		VerifyPeerCertificate:  uConn.VerifyPeerCertificate,
		ServerName:             config.ServerName,
		InsecureSkipVerify:     true,
		SessionTicketsDisabled: true,
	}
	if utlsConfig.ServerName == "" {
		utlsConfig.ServerName = dest.Address.String()
	}
	uConn.ServerName = utlsConfig.ServerName
	fingerprint := GetFingerprint(config.Fingerprint)
	if fingerprint == nil {
		return nil, newError("REALITY: failed to get fingerprint").AtError()
	}
	uConn.UConn = utls.UClient(conn, utlsConfig, *fingerprint)
	if err := uConn.BuildHandshakeState(); err != nil {
		return nil, newError("REALITY: unable to build client hello").Base(err)
	}
	if config.DisableX25519Mlkem768 {
		for _, extension := range uConn.Extensions {
			if ext, ok := extension.(*utls.SupportedCurvesExtension); ok {
				ext.Curves = slices.DeleteFunc(ext.Curves, func(curveID utls.CurveID) bool {
					return curveID == utls.X25519MLKEM768
				})
			}
			if ext, ok := extension.(*utls.KeyShareExtension); ok {
				ext.KeyShares = slices.DeleteFunc(ext.KeyShares, func(share utls.KeyShare) bool {
					return share.Group == utls.X25519MLKEM768
				})
			}
		}
		if err := uConn.BuildHandshakeState(); err != nil {
			return nil, newError("REALITY: unable to build client hello")
		}
	}
	hello := uConn.HandshakeState.Hello
	raw := hello.Raw
	if len(raw) == 0 {
		// utls.HelloGolang
		var err error
		raw, err = hello.Marshal()
		if err != nil {
			return nil, err
		}
	}
	hello.SessionId = make([]byte, 32)
	copy(raw[39:], hello.SessionId) // the fixed location of `Session ID`
	hello.SessionId[0] = 25         // Version_x
	hello.SessionId[1] = 5          // Version_y
	hello.SessionId[2] = 16         // Version_z
	hello.SessionId[3] = 0          // reserved
	binary.BigEndian.PutUint32(hello.SessionId[4:], uint32(time.Now().Unix()))
	copy(hello.SessionId[8:], config.ShortId)
	publicKey, err := ecdh.X25519().NewPublicKey(config.PublicKey)
	if err != nil {
		return nil, newError("REALITY: publicKey == nil").Base(err)
	}
	var ecdhe *ecdh.PrivateKey
	if keyShareKeys := uConn.HandshakeState.State13.KeyShareKeys; keyShareKeys != nil {
		if keyShareKeys.Ecdhe != nil {
			ecdhe = uConn.HandshakeState.State13.KeyShareKeys.Ecdhe
		}
		if !config.DisableX25519Mlkem768 && ecdhe == nil && keyShareKeys.MlkemEcdhe != nil {
			ecdhe = uConn.HandshakeState.State13.KeyShareKeys.MlkemEcdhe
		}
	}
	if ecdhe == nil {
		return nil, newError("Current fingerprint ", uConn.ClientHelloID.Client, uConn.ClientHelloID.Version, " does not support TLS 1.3, REALITY handshake cannot establish.")
	}
	uConn.AuthKey, _ = ecdhe.ECDH(publicKey)
	if uConn.AuthKey == nil {
		return nil, newError("REALITY: SharedKey == nil")
	}
	if _, err := hkdf.New(sha256.New, uConn.AuthKey, hello.Random[:20], []byte("REALITY")).Read(uConn.AuthKey); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(uConn.AuthKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	aead.Seal(hello.SessionId[:0], hello.Random[20:], hello.SessionId[:16], raw)
	copy(raw[39:], hello.SessionId)
	if err := uConn.HandshakeContext(ctx); err != nil {
		return nil, err
	}
	if !uConn.Verified {
		go func() {
			client := &http.Client{
				Transport: &http2.Transport{
					DialTLSContext: func(ctx context.Context, network, addr string, cfg *gotls.Config) (net.Conn, error) {
						return uConn, nil
					},
				},
			}
			req, err := http.NewRequest("GET", "https://"+uConn.ServerName, nil)
			if err != nil {
				return
			}
			req.Header.Set("User-Agent", fingerprint.Client)
			req.AddCookie(&http.Cookie{
				Name:  "padding",
				Value: strings.Repeat("0", dice.Roll(32)+30),
			})
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			_, _ = io.Copy(io.Discard, resp.Body)
		}()
		return nil, newError("REALITY: processed invalid connection")
	}
	return uConn, nil
}
