//go:build !windows && !wasm
// +build !windows,!wasm

package domainsocket

import (
	"os"

	"golang.org/x/sys/unix"
)

func (fl *fileLocker) Acquire() error {
	f, err := os.Create(fl.path)
	if err != nil {
		return err
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		f.Close()
		return newError("failed to lock file: ", fl.path).Base(err)
	}
	fl.file = f
	return nil
}

func (fl *fileLocker) Release() {
	if err := unix.Flock(int(fl.file.Fd()), unix.LOCK_UN); err != nil {
		newError("failed to unlock file: ", fl.path).Base(err).WriteToLog()
	}
	if err := fl.file.Close(); err != nil {
		newError("failed to close file: ", fl.path).Base(err).WriteToLog()
	}
	if err := os.Remove(fl.path); err != nil {
		newError("failed to remove file: ", fl.path).Base(err).WriteToLog()
	}
}
