//go:build !restfulapi

package v4

import (
	"github.com/golang/protobuf/proto"
)

type RestfulAPIConfig struct{}

func (r *RestfulAPIConfig) Build() (proto.Message, error) { // nolint:staticcheck
	return nil, newError("Restful API unsupported")
}
