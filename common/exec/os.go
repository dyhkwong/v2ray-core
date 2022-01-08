// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android

package exec

import (
	"os"
	"runtime"
	"syscall"
)

func startProcess(name string, argv []string, attr *os.ProcAttr) (*os.Process, error) {
	if attr != nil && attr.Sys == nil && attr.Dir != "" {
		if _, err := os.Stat(attr.Dir); err != nil {
			pe := err.(*os.PathError)
			pe.Op = "chdir"
			return nil, pe
		}
	}

	sysattr := &syscall.ProcAttr{
		Dir: attr.Dir,
		Env: attr.Env,
		Sys: attr.Sys,
	}
	if sysattr.Env == nil {
		sysattr.Env = syscall.Environ()
	}
	sysattr.Files = make([]uintptr, 0, len(attr.Files))
	for _, f := range attr.Files {
		sysattr.Files = append(sysattr.Files, f.Fd())
	}

	pid, _, e := syscall.StartProcess(name, argv, sysattr)

	runtime.KeepAlive(attr)

	if e != nil {
		return nil, &os.PathError{Op: "fork/exec", Path: name, Err: e}
	}

	return &os.Process{
		Pid: pid,
	}, nil
}
