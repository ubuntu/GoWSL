package wsl_test

import (
	"bytes"
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

// TestExitErrorAsString ensures that ExitError's message contains the actual code.
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
	err := realDistro.Command(context.Background(), "exit 0").Run()
	require.NoError(t, err)

	// Enum with various times in the execution
	type when uint
	const (
		CancelNever when = iota
		CancelBeforeRun
		CancelDuringRun
		CancelAfterRun
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
		"success with no cancel":  {cmd: "exit 0", cancelOn: CancelAfterRun},
		"linux error no cancel":   {cmd: "exit 42", cancelOn: CancelAfterRun, wantErr: true, wantExitError: &wsl.ExitError{Code: 42}},
		"windows error no cancel": {cmd: "exit 42", cancelOn: CancelAfterRun, fakeDistro: true, wantErr: true},
		"cancel before Run":       {cmd: "exit 0", cancelOn: CancelBeforeRun, wantErr: true},
		"cancel during Run":       {cmd: "sleep 5 && exit 0", cancelOn: CancelDuringRun, wantErr: true},
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
			if tc.cancelOn != CancelNever {
				ctx, cancel = context.WithCancel(ctx)
				defer cancel()
			}

			cmd := d.Command(ctx, tc.cmd)

			switch tc.cancelOn {
			case CancelBeforeRun:
				cancel()
			case CancelDuringRun:
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

func TestCommandStartWait(t *testing.T) {
	realDistro := newTestDistro(t, jammyRootFs)
	fakeDistro := wsl.Distro{Name: UniqueDistroName(t)}
	wrongDistro := wsl.Distro{Name: UniqueDistroName(t) + "--IHaveA\x00NullChar!"}

	// Enum with various times in the execution
	type when uint
	const (
		Never when = iota
		BeforeStart
		AfterStart
		AfterWait
	)

	whenToString := func(w when) string {
		switch w {
		case Never:
			return "NEVER"
		case BeforeStart:
			return "BEFORE_START"
		case AfterStart:
			return "AFTER_START"
		case AfterWait:
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
		"failure fake distro":         {distro: &fakeDistro, cmd: "exit 0", wantErrOn: AfterWait},
		"failure null char in distro": {distro: &wrongDistro, cmd: "exit 0", wantErrOn: AfterStart},
		"failure exit code":           {distro: &realDistro, cmd: "exit 42", wantErrOn: AfterWait, wantExitError: &wsl.ExitError{42}},

		// Timeout context
		"timeout success":          {distro: &realDistro, cmd: "exit 0", timeout: 2 * time.Second},
		"timeout exit code":        {distro: &realDistro, cmd: "exit 42", timeout: 2 * time.Second, wantErrOn: AfterWait, wantExitError: &wsl.ExitError{42}},
		"timeout before execution": {distro: &realDistro, cmd: "exit 0", timeout: time.Nanosecond, wantErrOn: AfterStart},
		"timeout during execution": {distro: &realDistro, cmd: "sleep 3", timeout: 2 * time.Second, wantErrOn: AfterWait},

		// Cancel context
		"cancel success":          {distro: &realDistro, cmd: "exit 0", cancelOn: AfterWait},
		"cancel exit code":        {distro: &realDistro, cmd: "exit 42", cancelOn: AfterWait, wantErrOn: AfterWait, wantExitError: &wsl.ExitError{42}},
		"cancel before execution": {distro: &realDistro, cmd: "exit 0", cancelOn: BeforeStart, wantErrOn: AfterStart},
		"cancel during execution": {distro: &realDistro, cmd: "sleep 3", cancelOn: AfterStart, wantErrOn: AfterWait},
	}

	// requireErrors checks that an error is emitted when expected, and checks that it is the proper type.
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
			require.Equal(t, err.(*wsl.ExitError).Code, tc.wantExitError.Code, "Unexpected value for ExitError.Code at time %s", whenToString(now)) // nolint: forcetypeassert, errorlint
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
			if tc.cancelOn != Never {
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
			if tc.cancelOn == BeforeStart {
				cancel()
			}

			cmd.Stdin = 0
			err := cmd.Start()

			// AFTER_START block
			if tc.cancelOn == AfterStart {
				cancel()
			}
			if requireErrors(t, tc, AfterStart, err) {
				return
			}

			err = cmd.Start()
			require.Error(t, err, "Unexpectedly succeeded at starting a command that had already been started")

			err = cmd.Wait()

			// AFTER_WAIT block
			requireErrors(t, tc, AfterWait, err)
		})
	}
}

func TestOutPipes(t *testing.T) {
	d := newTestDistro(t, jammyRootFs)

	testCases := map[string]struct {
		stdout     bool
		stderr     bool
		cmd        string
		expectRead string
	}{
		"all discarded":           {},
		"piped stdout":            {stdout: true, expectRead: "Hello stdout\n"},
		"piped stderr":            {stderr: true, expectRead: "Hello stderr\n"},
		"piped stdout and stderr": {stdout: true, stderr: true, expectRead: "Hello stdout\nHello stderr\n"},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			cmd := d.Command(context.Background(), "echo 'Hello stdout' >& 1 && echo 'Hello stderr' >& 2")

			var buff bytes.Buffer
			if tc.stdout {
				cmd.Stdout = &buff
			}
			if tc.stderr {
				cmd.Stderr = &buff
			}

			err := cmd.Run()
			require.NoError(t, err, "Did not expect an error when launching command")

			require.NoError(t, err, "Did not expect read from pipe to return an error")
			require.Equal(t, tc.expectRead, buff.String())
		})
	}

}
