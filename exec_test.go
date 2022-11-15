package wsl_test

import (
	"context"
	"testing"
	"time"
	"wsl"

	"github.com/stretchr/testify/require"
)

func TestExitStatusSuccess(tst *testing.T) {
	t := NewTester(tst)
	d := t.CachedDistro()

	// Test that exit values are returned correctly
	err := d.Command("exit 0").Run()
	require.NoError(t, err)
}

func TestNoExistExecutable(tst *testing.T) {
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

func TestExitStatusFailed(tst *testing.T) {
	t := NewTester(tst)
	d := t.CachedDistro()

	// Test that exit values are returned correctly
	err := d.Command("exit 42").Run()
	require.Error(t, err)

	var errAsExitError *wsl.ExitError
	require.ErrorAs(t, err, &errAsExitError)
	require.Equal(t, errAsExitError.Code, wsl.ExitCode(42))
}

func TestFailureNoDistro(tst *testing.T) {
	t := NewTester(tst)
	d := t.NewWslDistro("ubuntu")
	// We do not register it

	// Test that exit values are returned correctly
	err := d.Command("exit 0").Run()
	require.Error(t, err)
	require.Contains(t, err.Error(), "something went wrong Windows-side")

	var errAsExitError *wsl.ExitError
	require.NotErrorIs(t, err, errAsExitError)
}

// TestTimeoutEarlyFailure tests no error is returned if the command finishes on time
func TestTimeoutSuccess(tst *testing.T) {
	t := NewTester(tst)
	d := t.CachedDistro()

	// Poking distro to wake it up
	err := d.Command("exit 0").Run()
	require.NoError(t, err)

	// Actual test
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = d.CommandContext(ctx, "exit 0").Run()
	require.NoError(t, err)
}

// TestTimeoutEarlyFailure tests behaviour when timing out before command is launched
func TestTimeoutEarlyFailure(tst *testing.T) {
	t := NewTester(tst)
	d := t.CachedDistro()

	// Actual test
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(2 * time.Millisecond)
	t.Logf("Started\n")
	err := d.CommandContext(ctx, "exit 0").Run()
	t.Logf("Finished\n")
	require.Error(t, err)
	require.Contains(t, err.Error(), "context deadline exceeded")
}

// TestTimeoutLateFailure tests behaviour when timing out after command is launched
func TestTimeoutLateFailure(tst *testing.T) {
	t := NewTester(tst)
	d := t.CachedDistro()

	// Poking distro to wake it up
	err := d.Command("exit 0").Run()
	require.NoError(t, err)

	// Actual test
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	t.Logf("Started\n")
	err = d.CommandContext(ctx, "sleep 5 && exit 0").Run()
	t.Logf("Finished\n")
	require.Error(t, err)
	require.Contains(t, err.Error(), "process was closed before finshing")
}
