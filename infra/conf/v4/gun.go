package v4

import (
	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/transport/internet/grpc"
)

type GunConfig struct {
	ServiceName       string `json:"serviceName"`
	MultiMode         bool   `json:"multiMode"`
	ServiceNameCompat bool   `json:"serviceNameCompat"`
}

func (g GunConfig) Build() (proto.Message, error) {
	return &grpc.Config{
		ServiceName:       g.ServiceName,
		MultiMode:         g.MultiMode,
		ServiceNameCompat: g.ServiceNameCompat,
	}, nil
}
