//go:build gowslmock

// This file contains the implementation of testutils geared towards the mock back-end.

package gowsl_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	wsl "github.com/ubuntu/gowsl"
	wslmock "github.com/ubuntu/gowsl/mock"
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

// wslExeGuard guard is a dummy function so that the code compiles with mocks enabled.
// It should never be triggered because wsl.exe is mocked and cannot freeze, so the
// panic is there to indicate supposedly unreachable code.
func wslExeGuard(timeout time.Duration) (cancel func()) {
	tk := time.AfterFunc(timeout, func() {
		panic("wslExec guard triggered in mocks")
	})
	return func() { tk.Stop() }
}

// setupBackend is a convenience function that allows tests to build both with the production
// and mock back-ends, and take appropriate measures to make it work at runtime. Thus, its
// behaviour is different depending on the back-end.
//
// # Production back-end
//
// Any test that manipulates the mock needs the mock back-end to be accessible. setupBackend therefore does nothing,
// except return the same context that was passed, plus the modifyMock function. Attempting to call this function
// means that we need the mock back-end, so tests that call this function are skipped.
//
// # Mock back-end
//
// This module's only statefulness comes from the state of the registry. We're initializing a new back-end,
// therefore the state is not shared with any other tests. Hence, the current test can be marked parallel.
// The returned context contains the mock, and the returned function passes the mock to the supplied closure.
//
//nolint:revive // I'll put t before ctx, thank you.
func setupBackend(t *testing.T, ctx context.Context) (outCtx context.Context, modifyMock func(t *testing.T, f func(m *wslmock.Backend))) {
	t.Helper()
	t.Parallel()
	m := wslmock.New()

	outCtx = wsl.WithMock(ctx, m)
	modifyMock = func(t *testing.T, f func(*wslmock.Backend)) {
		t.Helper()
		f(m)
	}

	return outCtx, modifyMock
}
