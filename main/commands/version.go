package commands

import (
	"fmt"

	"github.com/v2fly/v2ray-core/v4"
	"github.com/v2fly/v2ray-core/v4/main/commands/base"
)

// CmdVersion prints V2Ray Versions
var CmdVersion = &base.Command{
	UsageLine: "{{.Exec}} version",
	Short:     "Print V2Ray Versions",
	Long: `Prints the build information for V2Ray.
`,
	Run: executeVersion,
}

func executeVersion(cmd *base.Command, args []string) {
	printVersion()
}

func printVersion() {
	version := core.VersionStatement()
	for _, s := range version {
		fmt.Println(s)
	}
}
