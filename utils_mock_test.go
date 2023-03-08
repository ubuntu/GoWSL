//go:build gowslmock

// This file contains the implementation of testutils geared towards the mock back-end.

package gowsl_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	wsl "github.com/ubuntu/gowsl"
)

// installDistro installs a distro.
//
//nolint:revive // No, I wont' put the context before the *testing.T.
func installDistro(t *testing.T, ctx context.Context, distroName, location, rootfs string) {
	t.Helper()

	d := wsl.NewDistro(ctx, distroName)
	err := d.Register(rootfs)
	require.NoError(t, err, "Setup: failed to register %q", distroName)
}

// uninstallDistro checks if a distro exists and if it does, it unregisters it.
//
// allowShutdown is unused because it is not necessary in the mock.
func uninstallDistro(distro wsl.Distro, allowShutdown bool) error {
	r, err := distro.IsRegistered()
	if err != nil {
		return err
	}
	if !r {
		return nil
	}

	return distro.Unregister()
}

// testDistros finds all registered distros.
func registeredDistros(ctx context.Context) ([]wsl.Distro, error) {
	return wsl.RegisteredDistros(ctx)
}

// defaultDistro gets the default distro's name.
func defaultDistro(ctx context.Context) (string, error) {
	d, err := wsl.DefaultDistro(ctx)
	if err != nil {
		return "", nil
	}
	return d.Name(), nil
}

// setDefaultDistro sets the default distro.
func setDefaultDistro(ctx context.Context, distroName string) error {
	d := wsl.NewDistro(ctx, distroName)
	return d.SetAsDefault()
}
