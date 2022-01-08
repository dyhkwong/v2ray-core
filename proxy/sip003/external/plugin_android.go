// workaround https://github.com/golang/go/issues/70508

package external

import (
	"io"
	"os"

	"github.com/v2fly/v2ray-core/v5/common/exec"
)

var _ Cmd = (*CmdWrapper)(nil)

type CmdWrapper struct {
	*exec.Cmd
}

func (c *CmdWrapper) Start() error {
	return c.Cmd.Start()
}

func (c *CmdWrapper) Path() string {
	return c.Cmd.Path
}

func (c *CmdWrapper) Args() []string {
	return c.Cmd.Args
}

func (c *CmdWrapper) Env() []string {
	return c.Cmd.Env
}

func (c *CmdWrapper) Stdout() io.Writer {
	return c.Cmd.Stdout
}

func (c *CmdWrapper) Stderr() io.Writer {
	return c.Cmd.Stderr
}

func (c *CmdWrapper) Dir() string {
	return c.Cmd.Dir
}

func (c *CmdWrapper) Process() *os.Process {
	return c.Cmd.Process
}

func (c *CmdWrapper) Clone() Cmd {
	cmd := &exec.Cmd{
		Path:   c.Cmd.Path,
		Args:   c.Cmd.Args,
		Stdout: c.Cmd.Stdout,
		Stderr: c.Cmd.Stderr,
		Env:    c.Cmd.Env,
		Dir:    c.Cmd.Dir,
	}
	return &CmdWrapper{cmd}
}

func NewCmd(path string, args []string, env []string, stdout, stderr io.Writer, dir string) Cmd {
	cmd := &exec.Cmd{
		Path:   path,
		Args:   args,
		Stdout: stdout,
		Stderr: stderr,
		Env:    env,
		Dir:    dir,
	}
	return &CmdWrapper{cmd}
}
