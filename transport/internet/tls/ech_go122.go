//go:build !go1.23 && !confonly
// +build !go1.23,!confonly

package tls

import (
	"crypto/tls"
)

func ApplyECH(c *Config, config *tls.Config) error {
	return newError("using ECH require go 1.23 or higher")
}
