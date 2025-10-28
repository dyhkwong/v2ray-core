package all

import (
	"crypto/mlkem"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/v2fly/v2ray-core/v5/main/commands/base"
)

var cmdMLKEM768 = &base.Command{
	UsageLine: `{{.Exec}} mlkem768 [-i "seed (base64.RawURLEncoding)"]`,
	Short:     `generate new VLESS encryption key pair`,
	Long: `
Generate new VLESS encryption key pair.
Random: {{.Exec}} mlkem768
From seed: {{.Exec}} mlkem768 -i "seed (base64.RawURLEncoding)"
`,
}

func init() {
	cmdMLKEM768.Run = executeMLKEM768 // break init loop
}

var input_mlkem768 = cmdMLKEM768.Flag.String("i", "", "")

func executeMLKEM768(cmd *base.Command, args []string) {
	var seed [64]byte
	if len(*input_mlkem768) > 0 {
		s, _ := base64.RawURLEncoding.DecodeString(*input_mlkem768)
		if len(s) != 64 {
			fmt.Println("Invalid length of ML-KEM-768 seed.")
			return
		}
		seed = [64]byte(s)
	} else {
		rand.Read(seed[:])
	}
	key, _ := mlkem.NewDecapsulationKey768(seed[:])
	client := key.EncapsulationKey().Bytes()
	fmt.Printf("Seed: %v\nClient: %v",
		base64.RawURLEncoding.EncodeToString(seed[:]),
		base64.RawURLEncoding.EncodeToString(client))
}
