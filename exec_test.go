package wsl_test

import (
	"context"
	"testing"
	"time"
	"wsl"

	"github.com/stretchr/testify/require"
)

func TestCommandExitStatusSuccess(tst *testing.T) {
	t := NewTester(tst)
	d := t.CachedDistro()

	// Test that exit values are returned correctly
	cmd := d.Command("exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Run()
	require.NoError(t, err)
}

func TestCommandNoExistExecutable(tst *testing.T) {
	t := NewTester(tst)
	d := t.CachedDistro()

	// Can't run a non-existent executable
	cmd := d.Command("/no-exist-executable")
	cmd.Stderr = 0
	cmd.Stdout = 0
	err := cmd.Run()
	require.Error(t, err)

	var errAsExitError *wsl.ExitError
	require.ErrorAs(t, err, &errAsExitError)
	require.Equal(t, errAsExitError.Code, wsl.ExitCode(127)) // 127: command not found
}

func TestCommandExitStatusFailed(tst *testing.T) {
	t := NewTester(tst)
	d := t.CachedDistro()

	// Test that exit values are returned correctly
	cmd := d.Command("exit 42")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Run()
	require.Error(t, err)

	var errAsExitError *wsl.ExitError
	require.ErrorAs(t, err, &errAsExitError)
	require.Equal(t, errAsExitError.Code, wsl.ExitCode(42))
}

func TestCommandFailureNoDistro(tst *testing.T) {
	t := NewTester(tst)
	d := t.NewWslDistro("ubuntu")
	// We do not register it

	// Test that exit values are returned correctly
	cmd := d.Command("exit 0")
	cmd.Stderr = 0
	cmd.Stdout = 0

	err := cmd.Run()
	require.Error(t, err)
	require.Contains(t, err.Error(), "something went wrong Windows-side")

	var errAsExitError *wsl.ExitError
	require.NotErrorIs(t, err, errAsExitError)
}

// TestCommandTimeoutSuccess tests no error is returned if the command finishes on time
func TestCommandTimeoutSuccess(tst *testing.T) {
	t := NewTester(tst)
	d := t.CachedDistro()

	// Poking distro to wake it up
	cmd := d.Command("exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Run()
	require.NoError(t, err)

	// Actual test
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd = d.CommandContext(ctx, "exit 0")
	cmd.Stderr = 0
	cmd.Stdout = 0
	err = cmd.Run()
	require.NoError(t, err)
}

// TestCommandTimeoutEarlyFailure tests behaviour when timing out before command is launched
func TestCommandTimeoutEarlyFailure(tst *testing.T) {
	t := NewTester(tst)
	d := t.CachedDistro()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond)

	cmd := d.CommandContext(ctx, "exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Run()
	require.Error(t, err)
	require.Contains(t, err.Error(), "context deadline exceeded")
}

// TestCommandTimeoutLateFailure tests behaviour when timing out after command is launched
func TestCommandTimeoutLateFailure(tst *testing.T) {
	t := NewTester(tst)
	d := t.CachedDistro()

	// Poking distro to wake it up
	cmd := d.Command("exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Run()
	require.NoError(t, err)

	// Actual test
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	cmd = d.CommandContext(ctx, "sleep 5 && exit 0")
	cmd.Stderr = 0
	cmd.Stdout = 0
	err = cmd.Run()
	require.Error(t, err)
	require.Contains(t, err.Error(), "process was closed before finshing")
}

// TestCommandCancelSuccess tests no error is returned if the command finishes on time
func TestCommandCancelSuccess(tst *testing.T) {
	t := NewTester(tst)
	d := t.CachedDistro()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := d.CommandContext(ctx, "exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Run()
	require.NoError(t, err)
}

// TestCommandCancelEarlyFailure tests behaviour when timing out before command is launched
func TestCommandCancelEarlyFailure(tst *testing.T) {
	t := NewTester(tst)
	d := t.CachedDistro()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cmd := d.CommandContext(ctx, "exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Run()
	require.Error(t, err)
	require.Contains(t, err.Error(), "context canceled")
}

// TestCommandCancelLateFailure tests behaviour when timing out after command is launched
func TestCommandCancelLateFailure(tst *testing.T) {
	t := NewTester(tst)
	d := t.CachedDistro()

	ctx, cancel := context.WithCancel(context.Background())

	cmd := d.CommandContext(ctx, "exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Start()
	require.NoError(t, err)

	cancel()

	err = cmd.Wait()
	require.Error(t, err)
	require.Contains(t, err.Error(), "process was closed before finshing")
}
