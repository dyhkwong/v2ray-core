package v4

import (
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/app/persistentstorage/filesystemstorage"
)

type FileSystemStorageConfig struct {
	StateStorageRoot string `json:"stateStorageRoot"`
	InstanceName     string `json:"instanceName"`
	ProtoJSON        bool   `json:"protoJSON"`
}

func (c *FileSystemStorageConfig) Build() (proto.Message, error) {
	config := &filesystemstorage.Config{
		InstanceName: c.InstanceName,
		Protojson:    c.ProtoJSON,
	}
	switch strings.ToLower(c.StateStorageRoot) {
	case "workdir":
		config.StateStorageRoot = filesystemstorage.StateStorageRoot_WorkDir
	default:
		config.StateStorageRoot = filesystemstorage.StateStorageRoot_WorkDir
	}
	return config, nil
}
