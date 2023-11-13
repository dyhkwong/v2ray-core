package all

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"

	"github.com/v2fly/v2ray-core/v5/main/commands/base"
)

func init() {
	cmdWG.Run = executeWG // break init loop
}

var cmdWG = &base.Command{
	UsageLine: "{{.Exec}} wg [-i \"private key\"]",
	Short:     "generate new WireGuard key pair",
	Long: `Generate new WireGuard key pair.
`,
}

var wgInput = cmdWG.Flag.String("i", "", "")

func executeWG(cmd *base.Command, args []string) {
	cmd.Flag.Parse(args)
	var privateKey [32]byte
	if *wgInput != "" {
		b, err := base64.StdEncoding.DecodeString(*wgInput)
		if err != nil {
			base.Fatalf("%s", err)
			return
		}
		copy(privateKey[:], b)
	} else {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			base.Fatalf("%s", err)
			return
		}
		copy(privateKey[:], b)
		// Modify random bytes using algorithm described at:
		// https://cr.yp.to/ecdh.html.
		privateKey[0] &= 248
		privateKey[31] &= 127
		privateKey[31] |= 64
	}
	fmt.Println("Private key:", base64.StdEncoding.EncodeToString(privateKey[:]))
	var publicKey [32]byte
	// ScalarBaseMult uses the correct base value per https://cr.yp.to/ecdh.html,
	// so no need to specify it.
	curve25519.ScalarBaseMult(&publicKey, &privateKey)
	fmt.Println("Public key:", base64.StdEncoding.EncodeToString(publicKey[:]))
}
