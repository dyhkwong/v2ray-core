package tls

import (
	"fmt"
	"os"

	"github.com/v2fly/v2ray-core/v5/main/commands/base"
	v2tls "github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

var cmdPublicKeyHash = &base.Command{
	UsageLine: "{{.Exec}} tls certPublicKeyHash [--cert <cert.pem>]",
	Short:     "Generate certificate public key hash for given certificate bundle",
}

func init() {
	cmdPublicKeyHash.Run = executeCertPublicKeyHash // break init loop
}

var certFileForPubicKeyHash = cmdPublicKeyHash.Flag.String("cert", "cert.pem", "")

func executeCertPublicKeyHash(cmd *base.Command, args []string) {
	if len(*certFileForPubicKeyHash) == 0 {
		base.Fatalf("cert file not specified")
	}
	certContent, err := os.ReadFile(*certFileForPubicKeyHash)
	if err != nil {
		base.Fatalf("Failed to read cert file: %s", err)
		return
	}
	certPublicKeyHashB64, err := v2tls.CalculatePEMCertPublicKeySHA256Hash(certContent)
	if err != nil {
		base.Fatalf("%s", err)
		return
	}
	fmt.Println(certPublicKeyHashB64)
}
