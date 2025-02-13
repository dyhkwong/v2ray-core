package all

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"

	"github.com/v2fly/v2ray-core/v5/main/commands/base"
)

func init() {
	cmdX25519.Run = executeX25519 // break init loop
}

var cmdX25519 = &base.Command{
	UsageLine: "{{.Exec}} x25519 [-i \"private key\"]",
	Short:     "generate new REALITY key pair",
	Long: `Generate new REALITY key pair.
`,
}

var x25519Input = cmdX25519.Flag.String("i", "", "")

func executeX25519(cmd *base.Command, args []string) {
	cmd.Flag.Parse(args)
	var privateKey []byte
	if *x25519Input != "" {
		b, err := base64.RawURLEncoding.DecodeString(*x25519Input)
		if err != nil {
			base.Fatalf("%s", err)
			return
		}
		privateKey = b
	} else {
		privateKey = make([]byte, curve25519.ScalarSize)
		if _, err := rand.Read(privateKey); err != nil {
			base.Fatalf("%s", err)
			return
		}
		// Modify random bytes using algorithm described at:
		// https://cr.yp.to/ecdh.html.
		privateKey[0] &= 248
		privateKey[31] &= 127
		privateKey[31] |= 64
	}
	fmt.Println("Private key:", base64.RawURLEncoding.EncodeToString(privateKey))
	publicKey, err := curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		base.Fatalf("%s", err)
	}
	fmt.Println("Public key:", base64.RawURLEncoding.EncodeToString(publicKey))
}
