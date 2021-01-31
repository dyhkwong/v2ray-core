package main

import (
	"flag"
	"os"

	"github.com/v2fly/v2ray-core/v5/main/commands"
	"github.com/v2fly/v2ray-core/v5/main/commands/base"
	_ "github.com/v2fly/v2ray-core/v5/main/distro/all"
)

func main() {
	os.Args = getArgsV4Compatible()

	base.RootCommand.Long = "A unified platform for anti-censorship."
	base.RegisterCommand(commands.CmdRun)
	base.RegisterCommand(commands.CmdVersion)
	base.RegisterCommand(commands.CmdTest)
	base.SortLessFunc = runIsTheFirst
	base.SortCommands()
	base.Execute()
}

func runIsTheFirst(i, j *base.Command) bool {
	left := i.Name()
	right := j.Name()
	if left == "run" {
		return true
	}
	if right == "run" {
		return false
	}
	return left < right
}

func getArgsV4Compatible() []string {
	if len(os.Args) == 1 {
		return []string{os.Args[0], "run"}
	}
	if os.Args[1][0] != '-' {
		return os.Args
	}
	version := false
	test := false
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.BoolVar(&version, "version", false, "")
	fs.BoolVar(&test, "test", false, "")
	// parse silently, no usage, no error output
	fs.Usage = func() {}
	fs.SetOutput(&null{})
	err := fs.Parse(os.Args[1:])
	if err == flag.ErrHelp {
		return []string{os.Args[0], "help"}
	}
	if version {
		return []string{os.Args[0], "version"}
	}
	if test {
		return append([]string{os.Args[0], "test"}, os.Args[2:]...)
	}
	return append([]string{os.Args[0], "run"}, os.Args[1:]...)
}

type null struct{}

func (n *null) Write(p []byte) (int, error) {
	return len(p), nil
}
