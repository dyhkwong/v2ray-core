package external

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/v2fly/v2ray-core/v5/common/platform"
	"github.com/v2fly/v2ray-core/v5/common/signal/done"
	"github.com/v2fly/v2ray-core/v5/proxy/sip003"
)

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

var _ sip003.Plugin = (*Plugin)(nil)

func init() {
	sip003.SetPluginLoader(func(plugin string) sip003.Plugin {
		return &Plugin{Plugin: plugin}
	})
}

type Cmd interface {
	Start() error
	Path() string
	Args() []string
	Env() []string
	Stdout() io.Writer
	Stderr() io.Writer
	Dir() string
	Process() *os.Process
	Clone() Cmd
}

type Plugin struct {
	Plugin        string
	pluginProcess Cmd
	done          *done.Instance
}

func (p *Plugin) Init(localHost string, localPort string, remoteHost string, remotePort string, pluginOpts string, pluginArgs []string, workingDir string) error {
	p.done = done.New()
	path, err := exec.LookPath(p.Plugin)
	if err != nil && !errors.Is(err, exec.ErrDot) {
		return newError("plugin ", p.Plugin, " not found").Base(err)
	}
	_, name := filepath.Split(path)
	env := []string{
		"SS_REMOTE_HOST=" + remoteHost,
		"SS_REMOTE_PORT=" + remotePort,
		"SS_LOCAL_HOST=" + localHost,
		"SS_LOCAL_PORT=" + localPort,
	}
	if pluginOpts != "" {
		env = append(env, "SS_PLUGIN_OPTIONS="+pluginOpts)
	}
	env = append(env, os.Environ()...)
	proc := NewCmd(
		path,
		append([]string{path}, pluginArgs...),
		env,
		&pluginOutWriter{
			name: name,
		},
		&pluginErrWriter{
			name: name,
		},
		workingDir,
	)
	if err := p.startPlugin(proc); err != nil {
		return err
	}
	return nil
}

func (p *Plugin) startPlugin(oldProc Cmd) error {
	if p.done.Done() {
		return newError("closed")
	}

	proc := oldProc.Clone()

	newError("start process ", strings.Join(proc.Args(), " ")).AtInfo().WriteToLog()

	err := proc.Start()
	if err != nil {
		return newError("failed to start sip003 plugin ", proc.Path).Base(err)
	}

	go func() {
		time.Sleep(time.Second)
		err = platform.CheckChildProcess(proc.Process())
		if err != nil {
			newError("sip003 plugin ", proc.Path, " exits too fast").Base(err).WriteToLog()
			return
		}
		go p.waitPlugin()
	}()

	p.pluginProcess = proc

	return nil
}

func (p *Plugin) waitPlugin() {
	status, err := p.pluginProcess.Process().Wait()

	if p.done.Done() {
		return
	}

	if err != nil {
		newError("failed to get sip003 plugin status").Base(err).WriteToLog()
	} else {
		newError("sip003 plugin exited with code ", status.ExitCode(), ", try to restart").WriteToLog()
	}

	time.Sleep(time.Second)

	if err := p.startPlugin(p.pluginProcess); err != nil {
		newError(err).WriteToLog()
	} else {
		go p.waitPlugin()
	}
}

func (p *Plugin) Close() error {
	p.done.Close()
	proc := p.pluginProcess
	if proc != nil && proc.Process() != nil {
		proc.Process().Kill()
	}
	return nil
}
