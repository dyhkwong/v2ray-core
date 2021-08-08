package conf

import (
	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v4/app/observatory"
	"github.com/v2fly/v2ray-core/v4/infra/conf/cfgcommon/duration"
)

type ObservatoryConfig struct {
	SubjectSelector []string          `json:"subjectSelector"`
	ProbeURL        string            `json:"probeURL"`
	ProbeInterval   duration.Duration `json:"probeInterval"`
	ProbeInterval2  duration.Duration `json:"ProbeInterval"` // The key was misspelled. For backward compatibility, we have to keep track of the old key.
}

func (o *ObservatoryConfig) Build() (proto.Message, error) {
	probeInterval := int64(o.ProbeInterval)
	probeInterval2 := int64(o.ProbeInterval2)
	if probeInterval == 0 && probeInterval2 != 0 {
		probeInterval = probeInterval2
	}
	return &observatory.Config{SubjectSelector: o.SubjectSelector, ProbeUrl: o.ProbeURL, ProbeInterval: probeInterval}, nil
}
