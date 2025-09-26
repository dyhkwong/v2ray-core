package tls

import (
	"fmt"
	"os"

	"github.com/v2fly/v2ray-core/v5/main/commands/base"
	v2tls "github.com/v2fly/v2ray-core/v5/transport/internet/tls"
)

var cmdHash = &base.Command{
	UsageLine: "{{.Exec}} tls certHash [--cert <cert.pem>]",
	Short:     "Generate certificate hash for given certificate bundle",
}

func init() {
	cmdHash.Run = executeCertHash // break init loop
}

var certFileForHash = cmdHash.Flag.String("cert", "cert.pem", "")

func executeCertHash(cmd *base.Command, args []string) {
	if len(*certFileForHash) == 0 {
		base.Fatalf("cert file not specified")
	}
	certContent, err := os.ReadFile(*certFileForHash)
	if err != nil {
		base.Fatalf("Failed to read cert file: %s", err)
		return
	}
	certHashHex, err := v2tls.CalculatePEMCertSHA256Hash(certContent)
	if err != nil {
		base.Fatalf("%s", err)
		return
	}
	fmt.Println(certHashHex)
}
