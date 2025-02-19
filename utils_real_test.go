//go:build !gowslmock

// This file contains the implementation of testutils geared towards the real back-end.

package gowsl_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/decorate"
	wsl "github.com/ubuntu/gowsl"
	wslmock "github.com/ubuntu/gowsl/mock"
)

// installDistro installs using powershell to decouple the tests from Distro.Register
// CommandContext often fails to stop it, so a more aggressive approach is taken by rebooting WSL.
//
//nolint:revive // No, I wont' put the context before the *testing.T.//nolint:revive
func installDistro(t *testing.T, ctx context.Context, distroName, location, rootfs string) {
	t.Helper()

	defer wslExeGuard(2 * time.Minute)()

	cmd := fmt.Sprintf("$env:WSL_UTF8=1 ;  wsl --import %q %q %q", distroName, location, rootfs)

	//nolint:gosec // Code injection is not a concern in tests.
	out, err := exec.Command("powershell.exe", "-Command", cmd).CombinedOutput()
	require.NoErrorf(t, err, "Setup: failed to register %q: %s", distroName, out)
}

// uninstallDistro checks if a distro exists and if it does, it unregisters it.
func uninstallDistro(distro wsl.Distro, allowShutdown bool) (err error) {
	defer decorate.OnError(&err, "could not uninstall %q", distro.Name())

	if r, err := distro.IsRegistered(); err == nil && !r {
		return nil
	}

	unregisterCmd := fmt.Sprintf("$env:WSL_UTF8=1 ; wsl.exe --unregister %q", distro.Name())
	defer wslExeGuard(2 * time.Minute)()

	// 1. Attempt unregistering

	//nolint:gosec // Code injection is not a concern in tests.
	e := exec.Command("powershell.exe", "-Command", unregisterCmd).Run()
	if e == nil {
		return nil
	}
	// Failed unregistration
	err = errors.Join(err, fmt.Errorf("could not unregister: %v", e))

	// 2. Attempt terminate, then unregister

	cmd := fmt.Sprintf("$env:WSL_UTF8=1 ; wsl.exe --terminate %q", distro.Name())
	if out, e := exec.Command("powershell.exe", "-Command", cmd).CombinedOutput(); e != nil { //nolint:gosec // Code injection is not a concern in tests.
		// Failed to terminate
		err = errors.Join(err, fmt.Errorf("could not terminate after failing to unregister: %v. Output: %s", e, string(out)))
	} else {
		// Terminated, retry unregistration
		out, e := exec.Command("powershell.exe", "-Command", unregisterCmd).CombinedOutput() //nolint:gosec // Code injection is not a concern in tests.
		if e != nil {
			return nil
		}

		// Failed unregistration
		err = errors.Join(err, fmt.Errorf("could not unregister after terminating: %v. Output: %s", e, string(out)))
	}

	if !allowShutdown {
		return err
	}

	// 3. Attempt shutdown, then unregister

	fmt.Fprintf(os.Stderr, "Could not unregister %q, shutting down WSL and retrying.", distro.Name())

	if out, e := exec.Command("powershell.exe", "-Command", "$env:WSL_UTF8=1 ; wsl.exe --shutdown").CombinedOutput(); e != nil {
		// Failed to shut down WSL
		return errors.Join(err, fmt.Errorf("could not shut down WSL after failing to unregister: %v. Output: %s", e, string(out)))
	}

	// WSL has been shut down, retry unregistration
	out, e := exec.Command("powershell.exe", "-Command", unregisterCmd).Output() //nolint:gosec // Code injection is not a concern in tests.
	if e != nil {
		// Failed unregistration
		return errors.Join(err, fmt.Errorf("could not unregister after shutdown: %v\nOutput: %v", e, string(out)))
	}

	// Success
	return nil
}

// testDistros finds all distros with a mangled name.
func registeredDistros(ctx context.Context) (distros []wsl.Distro, err error) {
	defer wslExeGuard(5 * time.Second)()

	outp, err := exec.Command("powershell.exe", "-Command", "$env:WSL_UTF8=1 ; wsl.exe --list --quiet --all").Output()
	if err != nil {
		return distros, err
	}

	for _, line := range strings.Fields(string(outp)) {
		distros = append(distros, wsl.NewDistro(ctx, line))
	}

	return distros, err
}

// defaultDistro gets the default distro's name via wsl.exe --status to bypass wsl.Default in order to better decouple tests.
func defaultDistro(ctx context.Context) (string, bool, error) {
	defer wslExeGuard(5 * time.Second)()

	out, err := exec.Command("powershell.exe", "-NoProfile", "-Command", "$env:WSL_UTF8=1; wsl.exe --status").CombinedOutput()
	if err != nil {
		return "", false, fmt.Errorf("failed to find current default distro: %v", err)
	}

	// wsl --status output looks like
	// When there is a default distro:
	// ```
	// Default Distribution: Ubuntu-20.04
	// Default Version: 2
	// ```
	// When there is no default distro:
	// ```
	// Default Version: 2
	// ```
	//
	// The first field (considering ":" as separator) is localized, so we cannot rely on it for comparisons.
	// The second field, though, is either the WSL version (1 or 2) or the default distro name.
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		fields := strings.Split(s.Text(), ":")
		if len(fields) < 2 { //ill-formed line?
			continue
		}

		if _, err := strconv.Atoi(strings.TrimSpace(fields[1])); err == nil {
			// it's a number, so must be the version line
			continue
		}

		return strings.TrimSpace(fields[1]), true, nil
	}

	return "", false, nil
}

// setDefaultDistro sets a distro as default using Powershell.
func setDefaultDistro(ctx context.Context, distroName string) error {
	defer wslExeGuard(5 * time.Second)()

	// No threat of code injection, wsl.exe will only interpret this text as a distro name
	// and throw ErrNotExist if it does not exist.
	out, err := exec.Command("wsl.exe", "--set-default", distroName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set distro %q back as default: %v. Output: %s", distroName, err, out)
	}
	return nil
}

// wslExeGuard guards against common problems with wsl.exe, and should be called every time
// wsl.exe is used. It solves:
//   - Trying to use it from WSL can have unexpected results, so it panics when not on Windows.
//   - wsl.exe occasionally freezing: sometimes, for no apparent reason, wsl.exe stops responding,
//     and cancelling the context of the command is not enough to unfreeze it. The only known
//     workaround is to call `wsl --shutdown` from elsewhere.
//
// This function does just that when the timeout is exceeded.
func wslExeGuard(timeout time.Duration) (cancel func()) {
	if runtime.GOOS != "windows" {
		panic("You must use the mock back-end when not running on Windows")
	}

	gentleTimeout := time.AfterFunc(timeout, func() {
		fmt.Fprintf(os.Stderr, "wslExec guard triggered, shutting WSL down")
		_ = exec.Command("powershell.exe", "-Command", "$env:WSL_UTF8=1 ; wsl.exe --shutdown").Run()
	})

	panicTimeout := time.AfterFunc(timeout+30*time.Second, func() {
		panic("WSL froze and couldn't be stopped. Tests aborted.")
	})

	return func() {
		gentleTimeout.Stop()
		panicTimeout.Stop()
	}
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

	return ctx, func(t *testing.T, f func(*wslmock.Backend)) {
		t.Helper()
		t.Skip("This test is only available with the mock enabled")
	}
}
