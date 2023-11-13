package all

import (
	"github.com/v2fly/v2ray-core/v5/main/commands/all/api"
	"github.com/v2fly/v2ray-core/v5/main/commands/all/tls"
	"github.com/v2fly/v2ray-core/v5/main/commands/base"
)

func init() {
	base.RootCommand.Commands = append(
		base.RootCommand.Commands,
		api.CmdAPI,
		cmdLove,
		tls.CmdTLS,
		cmdUUID,
		cmdX25519,
		cmdWG,

		// documents
		docFormat,
		docMerge,
	)
}
