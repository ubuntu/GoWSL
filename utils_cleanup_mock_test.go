//go:build gowslmock

package gowsl_test

import (
	"testing"

	wsl "github.com/ubuntu/gowsl"
)

func cleanupRegistry(t *testing.T, d wsl.Distro) error {
	t.Helper()
	t.Logf("Cleaning potential registry leftovers for %q are not needed when testing with mocks\n", d.Name())

	return nil
}
