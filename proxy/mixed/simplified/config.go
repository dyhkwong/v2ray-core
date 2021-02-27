package simplified

import (
	"context"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/proxy/mixed"
)

func init() {
	common.Must(common.RegisterConfig((*ServerConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		simplifiedServer := config.(*ServerConfig)
		_ = simplifiedServer
		fullServer := &mixed.ServerConfig{}
		return common.CreateObject(ctx, fullServer)
	}))
}
