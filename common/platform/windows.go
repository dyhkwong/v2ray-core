//go:build windows
// +build windows

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

func LineSeparator() string {
	return "\r\n"
}

// GetAssetLocation search for `file` in the excutable dir
func GetAssetLocation(file string) string {
	const name = "v2ray.location.asset"
	assetPath := NewEnvFlag(name).GetValue(getExecutableDir)
	return filepath.Join(assetPath, file)
}

func CheckChildProcess(proc *os.Process) error {
	handle, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(proc.Pid))
	if err != nil {
		return os.NewSyscallError("OpenProcess", err)
	}
	defer syscall.CloseHandle(handle)
	event, err := syscall.WaitForSingleObject(handle, 0)
	if err != nil {
		return os.NewSyscallError("WaitForSingleObject", err)
	}
	if event == syscall.WAIT_TIMEOUT {
		return nil
	}
	switch event {
	case syscall.WAIT_ABANDONED:
		return errors.New("WAIT_ABANDONED")
	case syscall.WAIT_FAILED:
		return errors.New("WAIT_FAILED")
	}
	return os.ErrProcessDone
}
