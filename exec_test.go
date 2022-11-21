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

func TestExitErrorIs(t *testing.T) {
	reference := wsl.ExitError{Code: 35}
	exit := wsl.ExitError{Code: 5}
	err := errors.New("")

	assert.ErrorIs(t, exit, reference, "An ExitError should have been detected as being an ExitError")
	assert.NotErrorIs(t, err, reference, "A string error should not have been detected as being an ExitError")
	assert.NotErrorIs(t, reference, err, "An ExitError error should not have been detected as being a string error")
}

// TestExitErrorAsString ensures that ExitError's message contains the actual code
func TestExitErrorAsString(t *testing.T) {
	t.Parallel()
	testCases := []uint32{1, 15, 255, wsl.ActiveProcess, wsl.WindowsError}

	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("%d", tc), func(t *testing.T) {
			t.Parallel()

			err := wsl.ExitError{Code: tc}

			s := fmt.Sprintf("%v", err)
			assert.Contains(t, s, fmt.Sprintf("%d", tc))

			s = err.Error()
			assert.Contains(t, s, fmt.Sprintf("%d", tc))
		})
	}
}

func TestCommandRun(t *testing.T) {
	realDistro := newTestDistro(t, jammyRootFs)
	fakeDistro := wsl.Distro{Name: UniqueDistroName(t)}

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

//TODO: STAR/WAIT()
func TestCommandStartWait(t *testing.T) {
	realDistro := newTestDistro(t, jammyRootFs)
	fakeDistro := wsl.Distro{Name: UniqueDistroName(t)}
	wrongDistro := wsl.Distro{Name: UniqueDistroName(t) + "--IHaveA\x00NullChar!"}

	// Enum with various times in the execution
	type when uint
	const (
		NEVER when = iota
		BEFORE_START
		AFTER_START
		AFTER_WAIT
	)

	whenToString := func(w when) string {
		switch w {
		case NEVER:
			return "NEVER"
		case BEFORE_START:
			return "BEFORE_START"
		case AFTER_START:
			return "AFTER_START"
		case AFTER_WAIT:
			return "AFTER_WAIT"
		}
		return "UNKNOWN_TIME"
	}

	type testCase struct {
		distro   *wsl.Distro
		cmd      string
		cancelOn when
		timeout  time.Duration

		wantErrOn     when
		wantExitError *wsl.ExitError
	}

	testCases := map[string]testCase{
		// Background context
		"success":                     {distro: &realDistro, cmd: "exit 0"},
		"failure fake distro":         {distro: &fakeDistro, cmd: "exit 0", wantErrOn: AFTER_WAIT},
		"failure null char in distro": {distro: &wrongDistro, cmd: "exit 0", wantErrOn: AFTER_START},
		"failure exit code":           {distro: &realDistro, cmd: "exit 42", wantErrOn: AFTER_WAIT, wantExitError: &wsl.ExitError{42}},

		// Timeout context
		"timeout sucess":           {distro: &realDistro, cmd: "exit 0", timeout: 2 * time.Second},
		"timeout exit code":        {distro: &realDistro, cmd: "exit 42", timeout: 2 * time.Second, wantErrOn: AFTER_WAIT, wantExitError: &wsl.ExitError{42}},
		"timeout before execution": {distro: &realDistro, cmd: "exit 0", timeout: time.Nanosecond, wantErrOn: AFTER_START},
		"timeout during execution": {distro: &realDistro, cmd: "sleep 3", timeout: 2 * time.Second, wantErrOn: AFTER_WAIT},

		// Cancel context
		"cancel sucess":           {distro: &realDistro, cmd: "exit 0", cancelOn: AFTER_WAIT},
		"cancel exit code":        {distro: &realDistro, cmd: "exit 42", cancelOn: AFTER_WAIT, wantErrOn: AFTER_WAIT, wantExitError: &wsl.ExitError{42}},
		"cancel before execution": {distro: &realDistro, cmd: "exit 0", cancelOn: BEFORE_START, wantErrOn: AFTER_START},
		"cancel during execution": {distro: &realDistro, cmd: "sleep 3", cancelOn: AFTER_START, wantErrOn: AFTER_WAIT},
	}

	// requireErrors checks that an error is emited when expected, and checks that it is the proper type.
	// Returns true if, as expected, an error was caught.
	// Returns false if, as expected, no error was caught.
	// Fails the test if err does not match expectations.
	requireErrors := func(t *testing.T, tc testCase, now when, err error) bool {
		t.Helper()
		if tc.wantErrOn != now {
			require.NoError(t, err, "did not expect an error at time %s", whenToString(now))
			return false
		}
		require.Error(t, err, "Unexpected success at time %s", whenToString(now))

		if tc.wantExitError != nil {
			require.ErrorIsf(t, err, wsl.ExitError{}, "Unexpected error type at time %s. Expected an ExitCode.", whenToString(now))
			require.Equal(t, err.(*wsl.ExitError).Code, tc.wantExitError.Code, "Unexpected value for ExitError.Code at time %s", whenToString(now))
			return true
		}

		// Ensure that we don't get an ExitError
		require.NotErrorIs(t, err, wsl.ExitError{}, "Unexpected error type at time %s. Expected anything but an ExitCode.", whenToString(now))
		return true
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			var cancel context.CancelFunc
			if tc.cancelOn != NEVER {
				ctx, cancel = context.WithCancel(context.Background())
				defer cancel()
			}
			if tc.timeout != 0 {
				ctx, cancel = context.WithTimeout(ctx, tc.timeout)
				defer cancel()
				time.Sleep(time.Second)
			}

			cmd := tc.distro.Command(ctx, tc.cmd)

			// BEFORE_START block
			if tc.cancelOn == BEFORE_START {
				cancel()
			}

			cmd.Stdin = 0
			cmd.Stderr = 0
			cmd.Stdout = 0
			err := cmd.Start()

			// AFTER_START block
			if tc.cancelOn == AFTER_START {
				cancel()
			}
			if requireErrors(t, tc, AFTER_START, err) {
				return
			}

			err = cmd.Wait()

			// AFTER_WAIT block
			if requireErrors(t, tc, AFTER_WAIT, err) {
				return
			}
		})
	}
}
