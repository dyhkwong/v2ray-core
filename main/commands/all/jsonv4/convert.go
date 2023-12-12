package jsonv4

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"

	"google.golang.org/protobuf/proto"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/infra/conf/merge"
	"github.com/v2fly/v2ray-core/v5/main/commands/base"
	"github.com/v2fly/v2ray-core/v5/main/commands/helpers"
)

var cmdConvert = &base.Command{
	CustomFlags: true,
	UsageLine:   "{{.Exec}} convert [c1.json] [<url>.json] [dir1] ...",
	Short:       "convert config files",
	Long: `
Convert config files between different formats. Files are merged 
before convert.

Arguments:

	-i, -input <format>
		The input format.
		Available values: "auto", "json"
		Default: "auto"

	-o, -output <format>
		The output format
		Available values: "json", "protobuf" / "pb"
		Default: "json"

	-r
		Load folders recursively.

Examples:

	{{.Exec}} {{.LongName}} -output=protobuf "path/to/dir"   (1)
	{{.Exec}} {{.LongName}} -o=pb config.json                (2)
	{{.Exec}} {{.LongName}} c1.json c2.json                  (3)
	{{.Exec}} {{.LongName}} -output=json c1.json <url>.json  (4)

(1) Merge all supported files in dir and convert to protobuf
(2) Convert json to pb
(3) Merge json files
(4) Merge json files

Use "{{.Exec}} help config-merge" for more information about merge.
`,
}

func init() {
	cmdConvert.Run = executeConvert // break init loop
}

var (
	inputFormat        string
	outputFormat       string
	confDirRecursively bool
)

func setConfArgs(cmd *base.Command) {
	cmd.Flag.StringVar(&inputFormat, "input", core.FormatAuto, "")
	cmd.Flag.StringVar(&inputFormat, "i", core.FormatAuto, "")
	cmd.Flag.StringVar(&outputFormat, "output", "json", "")
	cmd.Flag.StringVar(&outputFormat, "o", "json", "")
	cmd.Flag.BoolVar(&confDirRecursively, "r", false, "")
}

func executeConvert(cmd *base.Command, args []string) {
	setConfArgs(cmd)
	cmd.Flag.Parse(args)
	inputFormat = strings.ToLower(inputFormat)
	outputFormat = strings.ToLower(outputFormat)

	inputFormatMerge := inputFormat
	if inputFormat == "jsonv5" {
		inputFormatMerge = "json"
	}
	m, err := helpers.LoadConfigToMap(cmd.Flag.Args(), inputFormatMerge, confDirRecursively)
	if err != nil {
		base.Fatalf("failed to merge: %s", err)
	}
	err = merge.ApplyRules(m)
	if err != nil {
		base.Fatalf("failed to apply merge rules: %s", err)
	}

	var out []byte
	switch outputFormat {
	case core.FormatJSON:
		out, err = json.Marshal(m)
		if err != nil {
			base.Fatalf("failed to convert to json: %s", err)
		}
	case core.FormatProtobuf, core.FormatProtobufShort:
		data, err := json.Marshal(m)
		if err != nil {
			base.Fatalf("failed to marshal json: %s", err)
		}
		r := bytes.NewReader(data)
		pbConfig, err := core.LoadConfig(inputFormat, r)
		if err != nil {
			base.Fatalf(err.Error())
		}
		out, err = proto.Marshal(pbConfig)
		if err != nil {
			base.Fatalf("failed to convert to protobuf: %s", err)
		}
	default:
		base.Errorf("invalid output format: %s", outputFormat)
		base.Errorf("Run '%s help %s' for details.", base.CommandEnv.Exec, cmd.LongName())
		base.Exit()
	}

	if _, err := os.Stdout.Write(out); err != nil {
		base.Fatalf("failed to write stdout: %s", err)
	}
}
