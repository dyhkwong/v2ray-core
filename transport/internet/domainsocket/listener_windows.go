//go:build windows && !confonly
// +build windows,!confonly

package domainsocket

func (fl *fileLocker) Acquire() error {
	return nil
}

func (fl *fileLocker) Release() {}
