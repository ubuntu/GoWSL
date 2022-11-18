package wsl_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
	"wsl"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRun(t *testing.T) {
	realDistro := newTestDistro(t, jammyRootFs)
	fakeDistro := wsl.Distro{Name: sanitizeDistroName(fmt.Sprintf("%s_%s_%s", namePrefix, t.Name(), uniqueId()))}

	// Poking distro to wake it up
	cmd := realDistro.Command(context.Background(), "exit 0")
	cmd.Stdout = 0
	cmd.Stderr = 0
	err := cmd.Run()
	require.NoError(t, err)

	// Enum with various times in the execution
	type when uint
	const (
		CANCEL_NEVER when = iota
		CANCEL_BEFORE_RUN
		CANCEL_DURING_RUN
		CANCEL_AFTER_RUN
	)

	testCases := map[string]struct {
		cmd     string
		timeout time.Duration

		fakeDistro bool
		cancelOn   when

		wantErr       bool
		wantExitError *wsl.ExitError
	}{
		"success":       {cmd: "exit 0"},
		"windows error": {cmd: "exit 0", fakeDistro: true, wantErr: true},
		"linux error":   {cmd: "exit 42", wantErr: true, wantExitError: &wsl.ExitError{Code: 42}},

		// timeout cases
		"success with timeout long enough":       {cmd: "exit 0", timeout: 6 * time.Second},
		"linux error with timeout long enough":   {cmd: "exit 42", timeout: 6 * time.Second, wantErr: true, wantExitError: &wsl.ExitError{Code: 42}},
		"windows error with timeout long enough": {cmd: "exit 0", fakeDistro: true, wantErr: true},
		"timeout before Run":                     {cmd: "exit 0", timeout: 1 * time.Nanosecond, wantErr: true},
		"timeout during Run":                     {cmd: "sleep 3 && exit 0", timeout: 2 * time.Second, wantErr: true},

		// cancel cases
		"success with no cancel":  {cmd: "exit 0", cancelOn: CANCEL_AFTER_RUN},
		"linux error no cancel":   {cmd: "exit 42", cancelOn: CANCEL_AFTER_RUN, wantErr: true, wantExitError: &wsl.ExitError{Code: 42}},
		"windows error no cancel": {cmd: "exit 42", cancelOn: CANCEL_AFTER_RUN, fakeDistro: true, wantErr: true},
		"cancel before Run":       {cmd: "exit 0", cancelOn: CANCEL_BEFORE_RUN, wantErr: true},
		"cancel during Run":       {cmd: "sleep 5 && exit 0", cancelOn: CANCEL_DURING_RUN, wantErr: true},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			d := realDistro
			if tc.fakeDistro {
				d = fakeDistro
			}

			ctx := context.Background()
			var cancel context.CancelFunc
			if tc.timeout != 0 {
				ctx, cancel = context.WithTimeout(ctx, tc.timeout)
				defer cancel()
				time.Sleep(time.Second) // Gives time for an early failure
			}
			if tc.cancelOn != CANCEL_NEVER {
				ctx, cancel = context.WithCancel(ctx)
				defer cancel()
			}

			cmd := d.Command(ctx, tc.cmd)
			cmd.Stdout = 0
			cmd.Stderr = 0

			switch tc.cancelOn {
			case CANCEL_BEFORE_RUN:
				cancel()
			case CANCEL_DURING_RUN:
				go func() {
					time.Sleep(1 * time.Second)
					cancel()
				}()
			}

			err := cmd.Run()

			if !tc.wantErr {
				require.NoError(t, err, "did not expect Run() to return an error")
				return
			}

			require.Error(t, err, "expected Run() to return an error")

			if tc.wantExitError != nil {
				var target wsl.ExitError
				if errors.As(err, &target) {
					require.Equal(t, target.Code, tc.wantExitError.Code, "returned error ExitError has unexpected Code status")
				}
				return
			}

			// Ensure that we don't get an ExitError
			require.NotErrorIs(t, err, wsl.ExitError{}, "Run() should not have returned an ExitError")
		})
	}
}

func TestExitErrorIs(t *testing.T) {
	reference := wsl.ExitError{Code: 35}
	exit := wsl.ExitError{Code: 5}
	err := errors.New("")

	assert.ErrorIs(t, exit, reference, "An ExitError should have been detected as being an ExitError")
	assert.NotErrorIs(t, err, reference, "A string error should not have been detected as being an ExitError")
	assert.NotErrorIs(t, reference, err, "An ExitError error should not have been detected as being a string error")
}

//TODO: STAR/WAIT()
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
		wantError     when
		errorContains string
	}{
		"nominal":                 {cancel: AFTER_WAIT, wantError: NEVER},
		"cancel before execution": {cancel: BEFORE_START, wantError: AFTER_START, errorContains: "context canceled"},
		"cancel during execution": {cancel: AFTER_START, wantError: AFTER_WAIT, errorContains: "process was closed before finishing"},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// Informing both the user and the linter
			require.Less(t, tc.cancel, NEVER, "Ill-formed test: cancel must be between %d and %d (inclusive)", BEFORE_START, AFTER_WAIT)
			require.LessOrEqual(t, tc.wantError, NEVER, "Ill-formed test: wantsError must be between %d and %d (inclusive)", AFTER_START, NEVER)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// BEFORE_START block
			if tc.cancel == BEFORE_START {
				cancel()
			}

			cmd = d.Command(ctx, "sleep 3 && exit 0")
			cmd.Stdin = 0
			cmd.Stderr = 0
			cmd.Stdout = 0
			err = cmd.Start()

			// AFTER_START block
			if tc.wantError == AFTER_START {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorContains)
				return
			}
			require.NoError(t, err)

			if tc.cancel == AFTER_START {
				cancel()
			}
			err = cmd.Wait()

			// AFTER_WAIT block
			if tc.wantError == AFTER_WAIT {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorContains)
				return
			}
			require.NoError(t, err)
		})
	}
}
