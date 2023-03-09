//go:build !gowslmock

// This file contains the implementation of testutils geared towards the real back-end.

package gowsl_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	wsl "github.com/ubuntu/gowsl"
)

// TestContext creates a context that will instruct GoWSL to use the right back-end
// based on whether it was build with mocking enabled.
func testContext(ctx context.Context) context.Context {
	return ctx
}

// installDistro installs using powershell to decouple the tests from Distro.Register
// CommandContext sometimes fails to stop it, so a more aggressive approach is taken by rebooting WSL.
// TODO: Consider if we want to retry.
//
//nolint:revive // No, I wont' put the context before the *testing.T.//nolint:revive
func installDistro(t *testing.T, ctx context.Context, distroName string, rootfs string) {
	t.Helper()

	// Timeout to attempt a graceful failure
	const gracefulTimeout = 60 * time.Second

	// Timeout to shutdown WSL
	const aggressiveTimeout = gracefulTimeout + 10*time.Second

	// Cannot use context.WithTimeout because I want to quit by doing wsl --shutdown
	expired := time.After(aggressiveTimeout)

	type combinedOutput struct {
		output string
		err    error
	}
	cmdOut := make(chan combinedOutput)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), gracefulTimeout)
		defer cancel()
		cmd := fmt.Sprintf("$env:WSL_UTF8=1 ; wsl.exe --import %s %s %s", distroName, t.TempDir(), rootfs)
		o, e := exec.CommandContext(ctx, "powershell.exe", "-Command", cmd).CombinedOutput() //nolint:gosec

		cmdOut <- combinedOutput{output: string(o), err: e}
		close(cmdOut)
	}()

	var output combinedOutput
	select {
	case <-expired:
		t.Logf("Setup: installation of WSL distro %s got stuck. Rebooting WSL.", distroName)
		e := exec.Command("wsl.exe", "--shutdown").Run()
		require.NoError(t, e, "Setup: failed to shutdown WSL after distro installation got stuck")

		// Almost guaranteed to error out here
		output = <-cmdOut
		require.NoError(t, output.err, output.output)

		require.Fail(t, "Setup: unknown state: successfully registered while WSL was shut down, stdout+stderr:", output.output)
	case output = <-cmdOut:
	}
	require.NoErrorf(t, output.err, "Setup: failed to register %q: %s", distroName, output.output)
}

// uninstallDistro checks if a distro exists and if it does, it unregisters it.
func uninstallDistro(distro wsl.Distro) error {
	if r, err := distro.IsRegistered(); err == nil && !r {
		return nil
	}
	cmd := fmt.Sprintf("$env:WSL_UTF8=1 ; wsl.exe --unregister %s", distro.Name())
	_, err := exec.Command("powershell.exe", "-command", cmd).CombinedOutput() //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to clean up test WSL distro %q: %v", distro.Name(), err)
	}
	return nil
}

// testDistros finds all distros with a mangled name.
func registeredDistros(ctx context.Context) (distros []wsl.Distro, err error) {
	outp, err := exec.Command("powershell.exe", "-command", "$env:WSL_UTF8=1 ; wsl.exe --list --quiet").Output()
	if err != nil {
		return distros, err
	}

	for _, line := range strings.Fields(string(outp)) {
		distros = append(distros, wsl.NewDistro(ctx, line))
	}

	return distros, err
}

// defaultDistro gets the default distro's name via wsl.exe to bypass wsl.DefaultDistro in order to
// better decouple tests.
func defaultDistro(ctx context.Context) (string, error) {
	out, err := exec.Command("powershell.exe", "-Command", "$env:WSL_UTF8=1; wsl.exe --list --verbose").CombinedOutput()
	if err != nil {
		if target := (&exec.ExitError{}); !errors.As(err, &target) {
			return "", fmt.Errorf("failed to find current default distro: %v", err)
		}
		// cannot read from target.StdErr because message is printed to Stdout
		if !strings.Contains(string(out), "Wsl/WSL_E_DEFAULT_DISTRO_NOT_FOUND") {
			return "", fmt.Errorf("failed to find current default distro: %v. Output: %s", err, out)
		}
		return "", nil // No distros installed: no default
	}

	s := bufio.NewScanner(bytes.NewReader(out))
	s.Scan() // Ignore first line (table header)
	for s.Scan() {
		line := s.Text()
		if !strings.HasPrefix(line, "*") {
			continue
		}
		data := strings.Fields(line)
		if len(data) < 2 {
			return "", fmt.Errorf("failed to parse 'wsl.exe --list --verbose' output, line %q", line)
		}
		return data[1], nil
	}

	if err := s.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("failed to find default distro in 'wsl.exe --list --verbose' output:\n%s", string(out))
}

// setDefaultDistro sets a distro as default using Powershell.
func setDefaultDistro(ctx context.Context, distroName string) error {
	// No threat of code injection, wsl.exe will only interpret this text as a distro name
	// and throw Wsl/Service/WSL_E_DISTRO_NOT_FOUND if it does not exist.
	out, err := exec.Command("wsl.exe", "--set-default", distroName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set distro %q back as default: %v. Output: %s", distroName, err, out)
	}
	return nil
}
