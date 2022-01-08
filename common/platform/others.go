//go:build !windows
// +build !windows

package platform

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/adrg/xdg"
)

func LineSeparator() string {
	return "\n"
}

// GetAssetLocation search for `file` in certain locations
func GetAssetLocation(file string) string {
	const name = "v2ray.location.asset"
	assetPath := NewEnvFlag(name).GetValue(getExecutableDir)
	defPath := filepath.Join(assetPath, file)
	relPath := filepath.Join("v2ray", file)
	fullPath, err := xdg.SearchDataFile(relPath)
	if err != nil {
		return defPath
	}
	return fullPath
}

func CheckChildProcess(proc *os.Process) error {
	return proc.Signal(syscall.Signal(0))
}
