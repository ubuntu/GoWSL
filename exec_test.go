package wsl_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
			return "Never"
		case BeforeStart:
			return "BeforeStart"
		case AfterStart:
			return "AfterStart"
		case AfterWait:
			return "AfterWait"
		}
		return "UNKNOWN_TIME"
	}

	type testCase struct {
		distro     *wsl.Distro
		cmd        string
		cancelOn   when
		timeout    time.Duration
		stdoutPipe bool
		stderrPipe bool

		wantStdout    string
		wantStderr    string
		wantErrOn     when
		wantExitError *wsl.ExitError
	}

	testCases := map[string]testCase{
		// Background context
		"success":                     {distro: &realDistro, cmd: "exit 0"},
		"failure fake distro":         {distro: &fakeDistro, cmd: "exit 0", wantErrOn: AfterWait},
		"failure null char in distro": {distro: &wrongDistro, cmd: "exit 0", wantErrOn: AfterStart},
		"failure exit code":           {distro: &realDistro, cmd: "exit 42", wantErrOn: AfterWait, wantExitError: &wsl.ExitError{Code: 42}},

		// Pipe success
		"success with empty stdout": {distro: &realDistro, cmd: "exit 0", stdoutPipe: true},
		"success with stdout":       {distro: &realDistro, cmd: "echo 'Hello!'", stdoutPipe: true, wantStdout: "Hello!\n"},
		"success with empty stderr": {distro: &realDistro, cmd: "exit 0", stdoutPipe: true},
		"success with stderr":       {distro: &realDistro, cmd: "echo 'Error!' 1>&2", stderrPipe: true, wantStderr: "Error!\n"},
		"success with both pipes":   {distro: &realDistro, cmd: "echo 'Hello!' && echo 'Error!' 1>&2", stdoutPipe: true, wantStdout: "Hello!\n", stderrPipe: true, wantStderr: "Error!\n"},

		// Pipe failure
		"failure exit code with stdout": {distro: &realDistro, cmd: "echo 'Hello!' && echo 'Error!' 1>&2 && exit 42", stdoutPipe: true, wantStdout: "Hello!\n", wantErrOn: AfterWait, wantExitError: &wsl.ExitError{Code: 42}},
		"failure exit code with stderr": {distro: &realDistro, cmd: "echo 'Hello!' && echo 'Error!' 1>&2 && exit 42", stderrPipe: true, wantStderr: "Error!\n", wantErrOn: AfterWait, wantExitError: &wsl.ExitError{Code: 42}},
		"failure exit code both pipes":  {distro: &realDistro, cmd: "echo 'Hello!' && echo 'Error!' 1>&2 && exit 42", stdoutPipe: true, wantStdout: "Hello!\n", stderrPipe: true, wantStderr: "Error!\n", wantErrOn: AfterWait, wantExitError: &wsl.ExitError{Code: 42}},

		// Timeout context
		"timeout success":          {distro: &realDistro, cmd: "exit 0", timeout: 2 * time.Second},
		"timeout exit code":        {distro: &realDistro, cmd: "exit 42", timeout: 2 * time.Second, wantErrOn: AfterWait, wantExitError: &wsl.ExitError{Code: 42}},
		"timeout before execution": {distro: &realDistro, cmd: "exit 0", timeout: time.Nanosecond, wantErrOn: AfterStart},
		"timeout during execution": {distro: &realDistro, cmd: "sleep 3", timeout: 2 * time.Second, wantErrOn: AfterWait},

		// Cancel context
		"cancel success":          {distro: &realDistro, cmd: "exit 0", cancelOn: AfterWait},
		"cancel exit code":        {distro: &realDistro, cmd: "exit 42", cancelOn: AfterWait, wantErrOn: AfterWait, wantExitError: &wsl.ExitError{Code: 42}},
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
			var stdout, stderr *bufio.Reader
			if tc.stdoutPipe {
				pr, err := cmd.StdoutPipe()
				require.NoError(t, err, "Unexpected failure in call to (*Cmd).StdoutPipe")
				stdout = bufio.NewReader(pr)
			}
			if tc.stderrPipe {
				pr, err := cmd.StderrPipe()
				require.NoError(t, err, "Unexpected failure in call to (*Cmd).StderrPipe")
				stderr = bufio.NewReader(pr)
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

			if stdout != nil {
				text, err := stdout.ReadString('\n')
				if err != nil {
					require.ErrorIs(t, err, io.EOF, "Unexpected failure reading from StdoutPipe")
				}
				assert.Equal(t, tc.wantStdout, text, "Mismatch in piped stdout")
			}

			if stderr != nil {
				text, err := stderr.ReadString('\n')
				if err != nil {
					require.ErrorIs(t, err, io.EOF, "Unexpected failure reading from StderrPipe")
				}
				assert.Equal(t, tc.wantStderr, text, "Mismatch in piped stderr")
			}

			err = cmd.Start()
			require.Error(t, err, "Unexpectedly succeeded at starting a command that had already been started")

			err = cmd.Wait()

			// AFTER_WAIT block
			requireErrors(t, tc, AfterWait, err)
		})
	}
}

func TestCommandOutPipes(t *testing.T) {
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

func TestCommandOutput(t *testing.T) {
	realDistro := newTestDistro(t, jammyRootFs)
	fakeDistro := wsl.Distro{Name: UniqueDistroName(t)}
	wrongDistro := wsl.Distro{Name: UniqueDistroName(t) + "--IHaveA\x00NullChar!"}

	testCases := map[string]struct {
		cmd           string
		distro        *wsl.Distro
		wantStdout    string
		wantError     bool
		wantExitError bool
		// Only relevant if wantExitError==true
		wantExitCode uint32
		wantStderr   string
	}{
		"happy path":                                   {cmd: "exit 0", distro: &realDistro},
		"happy path with stdout":                       {cmd: "echo Hello", distro: &realDistro, wantStdout: "Hello\n"},
		"unregistered distro":                          {cmd: "exit 0", distro: &fakeDistro, wantError: true},
		"null char in distro name":                     {cmd: "exit 0", distro: &wrongDistro, wantError: true},
		"non-zero return value":                        {cmd: "exit 42", distro: &realDistro, wantError: true, wantExitError: true, wantExitCode: 42},
		"non-zero return value with stderr":            {cmd: "echo 'Error!' >&2 && exit 42", distro: &realDistro, wantError: true, wantExitError: true, wantExitCode: 42, wantStderr: "Error!\n"},
		"non-zero return value with stdout and stderr": {cmd: "echo Hello && echo 'Error!' >&2 && exit 42", distro: &realDistro, wantError: true, wantExitError: true, wantExitCode: 42, wantStdout: "Hello\n", wantStderr: "Error!\n"},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			stdout, err := tc.distro.Command(ctx, tc.cmd).Output()

			if tc.wantError {
				require.Errorf(t, err, "Unexpected success calling Output(). Stdout:\n%s", stdout)
			} else {
				require.NoErrorf(t, err, "Unexpected failure calling Output(). Stdout:\n%s", stdout)
			}

			if tc.wantStdout != "" {
				require.Equal(t, tc.wantStdout, string(stdout), "Unexpected contents in stdout")
			}

			if !tc.wantExitError {
				return // Success
			}

			require.ErrorIsf(t, err, wsl.ExitError{}, "Unexpected error type. Expected an ExitCode.")
			require.Equal(t, err.(*wsl.ExitError).Code, tc.wantExitCode, "Unexpected value for ExitError.Code.") // nolint: forcetypeassert, errorlint

			require.Equal(t, tc.wantStderr, string(err.(*wsl.ExitError).Stderr), "Unexpected contents in stderr") // nolint: forcetypeassert, errorlint
		})
	}
}

func TestCommandCombinedOutput(t *testing.T) {
	realDistro := newTestDistro(t, jammyRootFs)
	fakeDistro := wsl.Distro{Name: UniqueDistroName(t)}
	wrongDistro := wsl.Distro{Name: UniqueDistroName(t) + "--IHaveA\x00NullChar!"}

	testCases := map[string]struct {
		cmd           string
		distro        *wsl.Distro
		wantOutput    string
		wantError     bool
		wantExitError bool
		// Only relevant if wantExitError==true
		wantExitCode uint32
	}{
		"happy path":                                   {cmd: "exit 0", distro: &realDistro},
		"happy path with stdout":                       {cmd: "echo Hello", distro: &realDistro, wantOutput: "Hello\n"},
		"unregistered distro":                          {cmd: "exit 0", distro: &fakeDistro, wantError: true},
		"null char in distro name":                     {cmd: "exit 0", distro: &wrongDistro, wantError: true},
		"non-zero return value":                        {cmd: "exit 42", distro: &realDistro, wantError: true, wantExitError: true, wantExitCode: 42},
		"non-zero return value with stderr":            {cmd: "echo 'Error!' >&2 && exit 42", distro: &realDistro, wantError: true, wantExitError: true, wantExitCode: 42, wantOutput: "Error!\n"},
		"non-zero return value with stdout and stderr": {cmd: "echo Hello && echo 'Error!' >&2 && exit 42", distro: &realDistro, wantError: true, wantExitError: true, wantExitCode: 42, wantOutput: "Hello\nError!\n"},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			output, err := tc.distro.Command(ctx, tc.cmd).CombinedOutput()

			if tc.wantError {
				require.Errorf(t, err, "Unexpected success calling CombinedOutput(). Stdout:\n%s", output)
			} else {
				require.NoErrorf(t, err, "Unexpected failure calling CombinedOutput(). Stdout:\n%s", output)
			}

			if tc.wantOutput != "" {
				// Cannot check for all outputs, because some are localized (e.g. when the distro does not exist)
				// So we only check when one is specified.
				require.Equal(t, tc.wantOutput, string(output), "Unexpected contents in stdout")
			}

			if !tc.wantExitError {
				return // Success
			}

			require.ErrorIsf(t, err, wsl.ExitError{}, "Unexpected error type. Expected an ExitCode.")
			require.Equal(t, err.(*wsl.ExitError).Code, tc.wantExitCode, "Unexpected value for ExitError.Code.") // nolint: forcetypeassert, errorlint
		})
	}
}
