package gowsl

import (
	"fmt"

	"github.com/ubuntu/decorate"
)

// ShellError returns error information when shell commands do not succeed.
type ShellError struct {
	exitCode uint32
}

// Error makes it so ShellError implements the error interface. In displays
// the exit code and some auxiliary info.
//
// We know that exit codes above 255 come from Windows, but error codes under
// 256 can come from both sides.
func (err *ShellError) Error() string {
	if int(err.exitCode) > 0xff {
		// Windows errors are commonly displayed in HEX, so we stick to the standard
		return fmt.Sprintf("shell failed Windows-side: exit code 0x%x", err.exitCode)
	}
	// Linux exit codes are always displayed in decimal
	return fmt.Sprintf("shell launched but returned exit code %d", err.exitCode)
}

// ExitCode is a getter for the exit code of the shell when it produces.
// Experimentally we've seen that linux produces exit codes under 255, and
// Windows produces them above or equal to 256.
func (err *ShellError) ExitCode() uint32 {
	return err.exitCode
}

// ShellOption is an optional parameter for (*Distro).Shell. Use any of the
// provided functions such as UseCWD().
type ShellOption func(*shellOptions)

type shellOptions struct {
	command string
	useCWD  bool
}

// UseCWD is an optional parameter for (*Distro).Shell that makes it so the
// shell is started on the current working directory. Otherwise, it starts
// at the distro's $HOME.
func UseCWD() ShellOption {
	return func(o *shellOptions) {
		o.useCWD = true
	}
}

// WithCommand is an optional parameter for (*Distro).Shell that allows you
// to shell into WSL with the specified command. Particularly useful to choose
// what shell to use. Otherwise, it uses the distro's default shell.
func WithCommand(cmd string) ShellOption {
	return func(o *shellOptions) {
		o.command = cmd
	}
}

// Shell is a wrapper around Win32's WslLaunchInteractive, which starts a shell
// on WSL with the specified command. If no command is specified, the default
// shell for that distro is launched.
//
// If the command is interactive (e.g. python, sh, bash, fish, etc.) an interactive
// session is started. This is a synchronous, blocking call.
//
// Stdout and Stderr are sent to the console, even if os.Stdout and os.Stderr are
// redirected:
//
//	PS> go run .\examples\demo.go > demo.log # This will not redirect the Shell
//
// Stdin will read from os.Stdin but if you try to pass it via powershell
// strange things happen, same as if you did:
//
//	PS> "exit 5" | wsl.exe
//
// Can be used with optional helper parameters UseCWD and WithCommand.
func (d *Distro) Shell(args ...ShellOption) (err error) {
	defer decorate.OnError(&err, "unsuccessful shell into distro %s", d.name)

	if err := d.mustBeRegistered(); err != nil {
		return err
	}

	options := shellOptions{
		command: "",
		useCWD:  false,
	}
	for _, f := range args {
		f(&options)
	}

	exitCode, err := d.backend.WslLaunchInteractive(d.Name(), options.command, options.useCWD)
	if err != nil {
		return err
	}

	if exitCode != 0 {
		return &ShellError{exitCode}
	}

	return nil
}
