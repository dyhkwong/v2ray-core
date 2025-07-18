package v4

import (
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/v2fly/v2ray-core/v5/common/serial"
	"github.com/v2fly/v2ray-core/v5/infra/conf/cfgcommon/tlscfg"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tlsmirror/mirrorenrollment"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tlsmirror/server"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tlsmirror/tlstrafficgen"
)

type TLSMirrorConfig struct {
	ForwardAddress                string                          `json:"forwardAddress"`
	ForwardPort                   uint16                          `json:"forwardPort"`
	ForwardTag                    string                          `json:"forwardTag"`
	CarrierConnectionTag          string                          `json:"carrierConnectionTag"`
	EmbeddedTrafficGenerator      *EmbeddedTrafficGeneratorConfig `json:"embeddedTrafficGenerator"`
	PrimaryKey                    []byte                          `json:"primaryKey"`
	ExplicitNonceCiphersuites     []uint32                        `json:"explicitNonceCiphersuites"`
	DeferInstanceDerivedWriteTime *TLSMirrorTimeSpecConfig        `json:"deferInstanceDerivedWriteTime"`
	TransportLayerPadding         *TransportLayerPaddingConfig    `json:"transportLayerPadding"`
	ConnectionEnrollment          *TLSMirrorEnrollmentConfig      `json:"connectionEnrollment"`
	SequenceWatermarkingEnabled   bool                            `json:"sequenceWatermarkingEnabled"`
}

type TLSMirrorEnrollmentConfig struct {
	PrimaryIngressOutbound string `json:"primaryIngressOutbound"`
	PrimaryEgressOutbound  string `json:"primaryEgressOutbound"`
}

type TLSMirrorTimeSpecConfig struct {
	BaseNanoseconds                    uint64 `json:"baseNanoseconds"`
	UniformRandomMultiplierNanoseconds uint64 `json:"uniformRandomMultiplierNanoseconds"`
}

type TransportLayerPaddingConfig struct {
	Enabled bool `json:"enabled"`
}

type EmbeddedTrafficGeneratorConfig struct {
	Steps        []*StepConfig      `json:"steps"`
	TLSSettings  *tlscfg.TLSConfig  `json:"tlsSettings"`
	UTLSSettings *tlscfg.UTLSConfig `json:"utlsSettings"`
}

type StepConfig struct {
	Name                         string                          `json:"name"`
	Host                         string                          `json:"host"`
	Path                         string                          `json:"path"`
	Method                       string                          `json:"method"`
	NextStep                     []*TransferCandidateConfig      `json:"nextStep"`
	ConnectionReady              bool                            `json:"connectionReady"`
	Headers                      []*HeaderConfig                 `json:"headers"`
	ConnectionRecallExit         bool                            `json:"connectionRecallExit"`
	WaitTime                     *TrafficGeneratorTimeSpecConfig `json:"waitTime"`
	H2DoNotWaitForDownloadFinish bool                            `json:"h2DoNotWaitForDownloadFinish"`
}

type TrafficGeneratorTimeSpecConfig struct {
	BaseNanoseconds                    uint64 `json:"baseNanoseconds"`
	UniformRandomMultiplierNanoseconds uint64 `json:"uniformRandomMultiplierNanoseconds"`
}

type TransferCandidateConfig struct {
	Weight       int32 `json:"weight"`
	GotoLocation int64 `json:"gotoLocation"`
}

type HeaderConfig struct {
	Name   string   `json:"name"`
	Value  string   `json:"value"`
	Values []string `json:"values"`
}

// Build implements Buildable.
func (c *TLSMirrorConfig) Build() (proto.Message, error) {
	config := &server.Config{
		ForwardAddress:              c.ForwardAddress,
		ForwardPort:                 uint32(c.ForwardPort),
		ForwardTag:                  c.ForwardTag,
		CarrierConnectionTag:        c.CarrierConnectionTag,
		PrimaryKey:                  c.PrimaryKey,
		ExplicitNonceCiphersuites:   c.ExplicitNonceCiphersuites,
		SequenceWatermarkingEnabled: c.SequenceWatermarkingEnabled,
	}
	if c.EmbeddedTrafficGenerator != nil {
		config.EmbeddedTrafficGenerator = new(tlstrafficgen.Config)
		if c.EmbeddedTrafficGenerator.Steps != nil {
			for _, s := range c.EmbeddedTrafficGenerator.Steps {
				step := &tlstrafficgen.Step{
					Name:                         s.Name,
					Host:                         s.Host,
					Path:                         s.Path,
					Method:                       s.Method,
					ConnectionReady:              s.ConnectionReady,
					ConnectionRecallExit:         s.ConnectionRecallExit,
					H2DoNotWaitForDownloadFinish: s.H2DoNotWaitForDownloadFinish,
				}
				if s.NextStep != nil {
					for _, ns := range s.NextStep {
						step.NextStep = append(step.NextStep, &tlstrafficgen.TransferCandidate{
							Weight:       ns.Weight,
							GotoLocation: ns.GotoLocation,
						})
					}
				}
				if s.Headers != nil {
					for _, header := range s.Headers {
						step.Headers = append(step.Headers, &tlstrafficgen.Header{
							Name:   header.Name,
							Value:  header.Value,
							Values: header.Values,
						})
					}
				}
				if s.WaitTime != nil {
					step.WaitTime = &tlstrafficgen.TimeSpec{
						BaseNanoseconds:                    s.WaitTime.BaseNanoseconds,
						UniformRandomMultiplierNanoseconds: s.WaitTime.UniformRandomMultiplierNanoseconds,
					}
				}
				config.EmbeddedTrafficGenerator.Steps = append(config.EmbeddedTrafficGenerator.Steps, step)
			}
		}
		if c.EmbeddedTrafficGenerator.TLSSettings != nil {
			if c.EmbeddedTrafficGenerator.TLSSettings.Fingerprint != "" {
				imitate := strings.ToLower(c.EmbeddedTrafficGenerator.TLSSettings.Fingerprint)
				imitate = strings.TrimPrefix(imitate, "hello")
				switch imitate {
				case "chrome", "firefox", "safari", "ios", "edge", "360", "qq":
					imitate += "_auto"
				}
				utlsSettings := &tlscfg.UTLSConfig{
					TLSConfig: c.EmbeddedTrafficGenerator.TLSSettings,
					Imitate:   imitate,
				}
				us, err := utlsSettings.Build()
				if err != nil {
					return nil, newError("Failed to build UTLS config.").Base(err)
				}
				config.EmbeddedTrafficGenerator.SecuritySettings = serial.ToTypedMessage(us)
			} else {
				ts, err := c.EmbeddedTrafficGenerator.TLSSettings.Build()
				if err != nil {
					return nil, newError("Failed to build TLS config.").Base(err)
				}
				config.EmbeddedTrafficGenerator.SecuritySettings = serial.ToTypedMessage(ts)
			}
		} else if c.EmbeddedTrafficGenerator.UTLSSettings != nil {
			us, err := c.EmbeddedTrafficGenerator.UTLSSettings.Build()
			if err != nil {
				return nil, newError("Failed to build UTLS config.").Base(err)
			}
			config.EmbeddedTrafficGenerator.SecuritySettings = serial.ToTypedMessage(us)
		}
	}
	if c.DeferInstanceDerivedWriteTime != nil {
		config.DeferInstanceDerivedWriteTime = &server.TimeSpec{
			BaseNanoseconds:                    c.DeferInstanceDerivedWriteTime.BaseNanoseconds,
			UniformRandomMultiplierNanoseconds: c.DeferInstanceDerivedWriteTime.UniformRandomMultiplierNanoseconds,
		}
	}
	if c.TransportLayerPadding != nil {
		config.TransportLayerPadding = &server.TransportLayerPadding{
			Enabled: c.TransportLayerPadding.Enabled,
		}
	}
	if c.ConnectionEnrollment != nil {
		config.ConnectionEnrollment = &mirrorenrollment.Config{
			PrimaryIngressOutbound: c.ConnectionEnrollment.PrimaryIngressOutbound,
			PrimaryEgressOutbound:  c.ConnectionEnrollment.PrimaryEgressOutbound,
		}
	}
	return config, nil
}
