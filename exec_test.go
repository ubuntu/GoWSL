package wsl_test

import (
	"context"
	"fmt"
	"testing"
	"time"
	"wsl"

	"github.com/stretchr/testify/require"
)

func TestCommandBackgroundContext(t *testing.T) {
	realDistro := newTestDistro(t, jammyRootFs)
	fakeDistro := wsl.Distro{Name: sanitizeDistroName(fmt.Sprintf("%s_%s_%s", namePrefix, t.Name(), uniqueId()))}

	// Poking distro to wake it up
	cmd := realDistro.Command(context.Background(), "exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Run()
	require.NoError(t, err)

	testCases := map[string]struct {
		cmd        string
		fakeDistro bool
		wantsError error
	}{
		"nominal":       {cmd: "exit 0"},
		"windows error": {cmd: "exit 0", fakeDistro: true, wantsError: fmt.Errorf("error during Distro.Wait: failed to Launch Linux command due to Windows-side error")},
		"linux error":   {cmd: "exit 42", wantsError: &wsl.ExitError{Code: 42}},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			d := realDistro
			if tc.fakeDistro {
				d = fakeDistro
			}

			cmd := d.Command(context.Background(), tc.cmd)
			cmd.Stdout = 0
			cmd.Stderr = 0
			err := cmd.Run()

			if tc.wantsError == nil {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			require.Equal(t, tc.wantsError.Error(), err.Error())
		})

	}
}

func TestCommandWithTimeout(t *testing.T) {
	d := newTestDistro(t, jammyRootFs)

	// Poking distro to wake it up
	cmd := d.Command(context.Background(), "exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Run()
	require.NoError(t, err)

	testCases := map[string]struct {
		wantsError    bool
		errorContains string
		timeout       time.Duration
	}{
		"nominal":               {timeout: 10 * time.Second},
		"deadline before Start": {timeout: 1 * time.Nanosecond, wantsError: true, errorContains: "context deadline exceeded"},
		"deadline after Start":  {timeout: 3 * time.Second, wantsError: true, errorContains: "process was closed before finshing"},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			time.Sleep(time.Second) // Gives time for an early failure

			cmd = d.Command(ctx, "sleep 4 && exit 0") // Gives time for a late failure, then exits normally
			cmd.Stderr = 0
			cmd.Stdout = 0
			err = cmd.Run()

			if !tc.wantsError {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errorContains)
		})
	}
}

func TestCommandWithCancel(t *testing.T) {
	d := newTestDistro(t, jammyRootFs)

	// Poking distro to wake it up
	cmd := d.Command(context.Background(), "exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Run()
	require.NoError(t, err)

	// Enum with various times in the execution
	type when uint
	const (
		BEFORE_START when = iota
		AFTER_START
		AFTER_WAIT
		NEVER
	)

	testCases := map[string]struct {
		cancel        when
		wantsError    when
		errorContains string
	}{
		"nominal":                 {cancel: AFTER_WAIT, wantsError: NEVER},
		"cancel before execution": {cancel: BEFORE_START, wantsError: AFTER_START, errorContains: "context canceled"},
		"cancel during execution": {cancel: AFTER_START, wantsError: AFTER_WAIT, errorContains: "process was closed before finshing"},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// Informing both the user and the linter
			require.Less(t, tc.cancel, NEVER, "Ill-formed test: cancel must be between %d and %d (inclusive)", BEFORE_START, AFTER_WAIT)
			require.LessOrEqual(t, tc.wantsError, NEVER, "Ill-formed test: wantsError must be between %d and %d (inclusive)", AFTER_START, NEVER)

			ctx, cancel := context.WithCancel(context.Background())
			var cancelled bool
			defer func() {
				if !cancelled {
					cancel()
				}
			}()

			// BEFORE_START block
			if tc.cancel == BEFORE_START {
				cancel()
				cancelled = true
			}

			cmd = d.Command(ctx, "sleep 3 && exit 0")
			cmd.Stdin = 0
			cmd.Stderr = 0
			cmd.Stdout = 0
			err = cmd.Start()

			// AFTER_START block
			if tc.wantsError == AFTER_START {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorContains)
				return
			}
			require.NoError(t, err)

			if tc.cancel == AFTER_START {
				cancel()
				cancelled = true
			}

			err = cmd.Wait()

			// AFTER_WAIT block
			if tc.wantsError == AFTER_WAIT {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorContains)
				return
			}
			require.NoError(t, err)
		})
	}
}
