//go:build gowslmock

// This file contains the implementation of testutils geared towards the mock back-end.

package gowsl_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	wsl "github.com/ubuntu/gowsl"
	wslmock "github.com/ubuntu/gowsl/mock"
)

// TestContext creates a context that will instruct GoWSL to use the right back-end
// based on whether it was build with mocking enabled.
func testContext(ctx context.Context) context.Context {
	return wsl.WithMock(ctx, wslmock.Backend{})
}

// installDistro installs using powershell to decouple the tests from Distro.Register
// CommandContext sometimes fails to stop it, so a more aggressive approach is taken by rebooting WSL.
//
// TODO: Implement mock.
//
//nolint:revive // No, I wont' put the context before the *testing.T.
func installDistro(t *testing.T, ctx context.Context, distroName string, rootfs string) {
	t.Helper()

	require.Fail(t, "Mock not implemented")
}

// uninstallDistro checks if a distro exists and if it does, it unregisters it.
//
// TODO: Implement mock.
func uninstallDistro(distro wsl.Distro) error {
	return errors.New("uninstallDistro not implemented for mock back-end")
}

// testDistros finds all distros with a mangled name.
//
// TODO: Implement mock.
func registeredDistros(ctx context.Context) (distros []wsl.Distro, err error) {
	return nil, errors.New("registeredDistros not implemented for mock back-end")
}

// defaultDistro gets the default distro's name via wsl.exe to bypass wsl.DefaultDistro in order to
// better decouple tests.
//
// TODO: Implement mock.
func defaultDistro(ctx context.Context) (string, error) {
	return "", errors.New("defaultDistro not implemented for mock back-end")
}

// setDefaultDistro sets a distro as default using Powershell.
//
// TODO: Implement mock.
func setDefaultDistro(ctx context.Context, distroName string) error {
	return errors.New("setDefaultDistro not implemented for mock back-end")
}
