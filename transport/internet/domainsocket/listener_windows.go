//go:build windows
// +build windows

package domainsocket

func (fl *fileLocker) Acquire() error {
	return nil
}

func (fl *fileLocker) Release() {}
