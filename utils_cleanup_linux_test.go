//go:build !gowslmock

package gowsl_test

import (
	"testing"

	wsl "github.com/ubuntu/gowsl"
)

func cleanupRegistry(t *testing.T, d wsl.Distro) error {
	t.Helper()

	panic("not implemented on linux")
}
