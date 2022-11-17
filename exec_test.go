package wsl_test

import (
	"context"
	"testing"
	"time"
	"wsl"

	"github.com/stretchr/testify/require"
)

func TestCommandExitStatusSuccess(t *testing.T) {
	d := newDistro(t, jammyRootFs)

	// Test that exit values are returned correctly
	cmd := d.Command(context.Background(), "exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Run()
	require.NoError(t, err)
}

func TestCommandNoExistExecutable(t *testing.T) {
	d := newDistro(t, jammyRootFs)

	// Can't run a non-existent executable
	cmd := d.Command(context.Background(), "/no-exist-executable")
	cmd.Stderr = 0
	cmd.Stdout = 0
	err := cmd.Run()
	require.Error(t, err)

	var errAsExitError *wsl.ExitError
	require.ErrorAs(t, err, &errAsExitError)
	require.Equal(t, errAsExitError.Code, wsl.ExitCode(127)) // 127: command not found
}

func TestCommandExitStatusFailed(t *testing.T) {
	d := newDistro(t, jammyRootFs)

	// Test that exit values are returned correctly
	cmd := d.Command(context.Background(), "exit 42")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Run()
	require.Error(t, err)

	var errAsExitError *wsl.ExitError
	require.ErrorAs(t, err, &errAsExitError)
	require.Equal(t, errAsExitError.Code, wsl.ExitCode(42))
}

func TestCommandFailureNoDistro(t *testing.T) {
	d := wsl.Distro{Name: "my favourite distro"}
	// We do not register it

	// Test that exit values are returned correctly
	cmd := d.Command(context.Background(), "exit 0")
	cmd.Stderr = 0
	cmd.Stdout = 0

	err := cmd.Run()
	require.Error(t, err)
	require.Contains(t, err.Error(), "something went wrong Windows-side")

	var errAsExitError *wsl.ExitError
	require.NotErrorIs(t, err, errAsExitError)
}

// TestCommandTimeoutSuccess tests no error is returned if the command finishes on time
func TestCommandTimeoutSuccess(t *testing.T) {
	d := newDistro(t, jammyRootFs)

	// Poking distro to wake it up
	cmd := d.Command(context.Background(), "exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Run()
	require.NoError(t, err)

	// Actual test
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd = d.Command(ctx, "exit 0")
	cmd.Stderr = 0
	cmd.Stdout = 0
	err = cmd.Run()
	require.NoError(t, err)
}

// TestCommandTimeoutEarlyFailure tests behaviour when timing out before command is launched
func TestCommandTimeoutEarlyFailure(t *testing.T) {
	d := newDistro(t, jammyRootFs)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(100 * time.Millisecond)

	cmd := d.Command(ctx, "exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Run()
	require.Error(t, err)
	require.Contains(t, err.Error(), "context deadline exceeded")
}

// TestCommandTimeoutLateFailure tests behaviour when timing out after command is launched
func TestCommandTimeoutLateFailure(t *testing.T) {
	d := newDistro(t, jammyRootFs)

	// Poking distro to wake it up
	cmd := d.Command(context.Background(), "exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Run()
	require.NoError(t, err)

	// Actual test
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	cmd = d.Command(ctx, "sleep 5 && exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err = cmd.Run()
	require.Error(t, err)
	require.Contains(t, err.Error(), "process was closed before finshing")
}

// TestCommandCancelSuccess tests no error is returned if the command finishes on time
func TestCommandCancelSuccess(t *testing.T) {
	d := newDistro(t, jammyRootFs)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := d.Command(ctx, "exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Run()
	require.NoError(t, err)
}

// TestCommandCancelEarlyFailure tests behaviour when timing out before command is launched
func TestCommandCancelEarlyFailure(t *testing.T) {
	d := newDistro(t, jammyRootFs)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cmd := d.Command(ctx, "exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Run()
	require.Error(t, err)
	require.Contains(t, err.Error(), "context canceled")
}

// TestCommandCancelLateFailure tests behaviour when timing out after command is launched
func TestCommandCancelLateFailure(t *testing.T) {
	d := newDistro(t, jammyRootFs)

	ctx, cancel := context.WithCancel(context.Background())

	cmd := d.Command(ctx, "exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Start()
	require.NoError(t, err)

	cancel()

	err = cmd.Wait()
	require.Error(t, err)
	require.Contains(t, err.Error(), "process was closed before finshing")
}

// func TestDistroString(t *testing.T) {
// 	t := NewTester(tst)
// 	d := t.CachedDistro()
//	got := fmt.Sprintf("%s", d)
// }
