// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// workaround https://github.com/golang/go/issues/70508

//go:build android

package exec

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type wrappedError struct {
	prefix string
	err    error
}

func (w wrappedError) Error() string {
	return w.prefix + ": " + w.err.Error()
}

func (w wrappedError) Unwrap() error {
	return w.err
}

type Cmd struct {
	Path          string
	Args          []string
	Env           []string
	Dir           string
	Stdout        io.Writer
	Stderr        io.Writer
	Process       *os.Process
	ctx           context.Context
	err           error
	childIOFiles  []io.Closer
	parentIOPipes []io.Closer
	goroutine     []func() error
	goroutineErr  <-chan error
}

func interfaceEqual(a, b any) bool {
	defer func() {
		recover()
	}()
	return a == b
}

func (c *Cmd) argv() []string {
	if len(c.Args) > 0 {
		return c.Args
	}
	return []string{c.Path}
}

func (c *Cmd) childStdin() (*os.File, error) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		return nil, err
	}
	c.childIOFiles = append(c.childIOFiles, f)
	return f, nil
}

func (c *Cmd) childStdout() (*os.File, error) {
	return c.writerDescriptor(c.Stdout)
}

func (c *Cmd) childStderr(childStdout *os.File) (*os.File, error) {
	if c.Stderr != nil && interfaceEqual(c.Stderr, c.Stdout) {
		return childStdout, nil
	}
	return c.writerDescriptor(c.Stderr)
}

func (c *Cmd) writerDescriptor(w io.Writer) (*os.File, error) {
	if w == nil {
		f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			return nil, err
		}
		c.childIOFiles = append(c.childIOFiles, f)
		return f, nil
	}

	if f, ok := w.(*os.File); ok {
		return f, nil
	}

	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	c.childIOFiles = append(c.childIOFiles, pw)
	c.parentIOPipes = append(c.parentIOPipes, pr)
	c.goroutine = append(c.goroutine, func() error {
		_, err := io.Copy(w, pr)
		pr.Close()
		return err
	})
	return pw, nil
}

func closeDescriptors(closers []io.Closer) {
	for _, fd := range closers {
		fd.Close()
	}
}

func (c *Cmd) Start() error {
	if c.Process != nil {
		return errors.New("exec: already started")
	}

	started := false
	defer func() {
		closeDescriptors(c.childIOFiles)
		c.childIOFiles = nil

		if !started {
			closeDescriptors(c.parentIOPipes)
			c.parentIOPipes = nil
		}
	}()

	if c.Path == "" && c.err == nil {
		c.err = errors.New("exec: no command")
	}
	if c.err != nil {
		return c.err
	}
	lp := c.Path
	if c.ctx != nil {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
		}
	}

	childFiles := make([]*os.File, 0, 3)
	stdin, err := c.childStdin()
	if err != nil {
		return err
	}
	childFiles = append(childFiles, stdin)
	stdout, err := c.childStdout()
	if err != nil {
		return err
	}
	childFiles = append(childFiles, stdout)
	stderr, err := c.childStderr(stdout)
	if err != nil {
		return err
	}
	childFiles = append(childFiles, stderr)

	env, err := c.environ()
	if err != nil {
		return err
	}

	c.Process, err = startProcess(lp, c.argv(), &os.ProcAttr{
		Dir:   c.Dir,
		Files: childFiles,
		Env:   env,
	})
	if err != nil {
		return err
	}
	started = true

	if len(c.goroutine) > 0 {
		goroutineErr := make(chan error, 1)

		type goroutineStatus struct {
			running  int
			firstErr error
		}
		statusc := make(chan goroutineStatus, 1)
		statusc <- goroutineStatus{running: len(c.goroutine)}
		for _, fn := range c.goroutine {
			go func(fn func() error) {
				err := fn()

				status := <-statusc
				if status.firstErr == nil {
					status.firstErr = err
				}
				status.running--
				if status.running == 0 {
					goroutineErr <- status.firstErr
				} else {
					statusc <- status
				}
			}(fn)
		}
		c.goroutine = nil
	}

	return nil
}

func (c *Cmd) environ() ([]string, error) {
	var err error

	env := c.Env
	if env == nil {
		env = syscall.Environ()

		if c.Dir != "" {
			if pwd, absErr := filepath.Abs(c.Dir); absErr == nil {
				env = append(env, "PWD="+pwd)
			} else {
				err = absErr
			}
		}
	}

	env, dedupErr := dedupEnv(env)
	if err == nil {
		err = dedupErr
	}
	return env, err
}

func dedupEnv(env []string) ([]string, error) {
	var err error
	out := make([]string, 0, len(env))
	saw := make(map[string]bool, len(env))
	for n := len(env); n > 0; n-- {
		kv := env[n-1]

		if strings.IndexByte(kv, 0) != -1 {
			err = errors.New("exec: environment variable contains NUL")
			continue
		}

		i := strings.Index(kv, "=")
		if i == 0 {
			i = strings.Index(kv[1:], "=") + 1
		}
		if i < 0 {
			if kv != "" {
				out = append(out, kv)
			}
			continue
		}
		k := kv[:i]
		if saw[k] {
			continue
		}

		saw[k] = true
		out = append(out, kv)
	}

	for i := 0; i < len(out)/2; i++ {
		j := len(out) - i - 1
		out[i], out[j] = out[j], out[i]
	}

	return out, err
}
