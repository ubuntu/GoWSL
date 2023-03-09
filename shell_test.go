package gowsl_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	wsl "github.com/ubuntu/gowsl"
)

func TestShell(t *testing.T) {
	ctx := testContext(context.Background())

	realDistro := newTestDistro(t, ctx, rootFs)
	fakeDistro := wsl.NewDistro(ctx, uniqueDistroName(t))
	wrongDistro := wsl.NewDistro(ctx, "I have a \x00 null char in my name")

	cmdExit0 := "exit 0"
	cmdExit42 := "exit 42"

	cmdCheckNotCWD := "[ `pwd` = /root ]"
	cmdCheckCWD := "[ `pwd` != /root ]"

	wrongCommand := "echo 'Oh no!, There is a \x00 in my command!'"

	testCases := map[string]struct {
		withCwd      bool
		withCommand  *string
		distro       *wsl.Distro
		wantError    bool
		wantExitCode uint32
	}{
		// Test with no arguments
		"happy path":   {distro: &realDistro},
		"fake distro":  {distro: &fakeDistro, wantError: true},
		"wrong distro": {distro: &wrongDistro, wantError: true},

		// Test UseCWD
		"success with CWD": {distro: &realDistro, withCwd: true},
		"failure with CWD": {distro: &fakeDistro, withCwd: true, wantError: true},

		// Test withCommand
		"success with command":              {distro: &realDistro, withCommand: &cmdExit0},
		"failure command with exit error":   {distro: &realDistro, withCommand: &cmdExit42, wantError: true, wantExitCode: 42},
		"failure with null char in command": {distro: &realDistro, withCommand: &wrongCommand, wantError: true},

		// Test that UseCWD actually changes the working directory
		"ensure default is not CWD": {distro: &realDistro, withCommand: &cmdCheckNotCWD},
		"ensure UseCWD uses CWD":    {distro: &realDistro, withCwd: true, withCommand: &cmdCheckCWD},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			d := *tc.distro

			// Keeping distro awake so there are no unexpected timeouts
			if d == realDistro {
				defer keepAwake(t, context.Background(), &realDistro)()
			}

			var opts []wsl.ShellOption
			if tc.withCommand != nil {
				opts = append(opts, wsl.WithCommand(*tc.withCommand))
			}
			if tc.withCwd {
				opts = append(opts, wsl.UseCWD())
			}

			// Because Shell is an interactive command, it needs to be quit from
			// outside. This goroutine sets a fuse before shutting down the distro.
			tk := time.AfterFunc(3*time.Second, func() {
				t.Logf("Command timed out")
				err := d.Terminate()
				if err != nil {
					t.Log(err)
				}
			})

			err := d.Shell(opts...)
			tk.Stop()

			if !tc.wantError {
				require.NoError(t, err, "Unexpected error after Distro.Shell")
				return
			}

			require.Error(t, err, "Unexpected success after Distro.Shell")

			var target *wsl.ShellError
			if tc.wantExitCode == 0 {
				notErrorAsf(t, err, &target, "unexpected ShellError, expected any other type")
				return
			}

			require.ErrorAs(t, err, &target, "unexpected error type, expected a ShellError")
			require.Equal(t, tc.wantExitCode, target.ExitCode(), "Unexpected value for ExitCode returned from Distro.Shell")
		})
	}
}
