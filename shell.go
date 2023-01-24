package gowsl

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

type shellOptions struct {
	command string
	useCWD  bool
}

// UseCWD is an optional parameter for (*Distro).Shell that makes it so the
// shell is started on the current working directory. Otherwise, it starts
// at the distro's $HOME.
func UseCWD() func(*shellOptions) {
	return func(o *shellOptions) {
		o.useCWD = true
	}
}

// WithCommand is an optional parameter for (*Distro).Shell that allows you
// to shell into WSL with the specified command. Particularly useful to choose
// what shell to use. Otherwise, it use's the distro's default shell.
func WithCommand(cmd string) func(*shellOptions) {
	return func(o *shellOptions) {
		o.command = cmd
	}
}

// Shell is a wrapper around Win32's WslLaunchInteractive, which starts a shell
// on WSL with the specified command. If no command is specified, an interactive
// session is started. This is a synchronous, blocking call.
//
// Can be used with optional helper parameters UseCWD and WithCommand.
func (d *Distro) Shell(opts ...func(*shellOptions)) (err error) {
	defer func() {
		if err == nil {
			return
		}
		if errors.Is(err, ExitError{}) {
			return
		}
		err = fmt.Errorf("error in Shell with distro %q: %v", d.Name(), err)
	}()

	r, err := d.IsRegistered()
	if err != nil {
		return err
	}
	if !r {
		return errors.New("distro is not registered")
	}

	options := shellOptions{
		command: "",
		useCWD:  false,
	}
	for _, o := range opts {
		o(&options)
	}

	distroUTF16, err := syscall.UTF16PtrFromString(d.Name())
	if err != nil {
		return errors.New("failed to convert distro name to UTF16")
	}

	commandUTF16, err := syscall.UTF16PtrFromString(options.command)
	if err != nil {
		return fmt.Errorf("failed to convert command %q to UTF16", options.command)
	}

	var useCwd wBOOL
	if options.useCWD {
		useCwd = 1
	}

	var exitCode uint32

	r1, _, _ := wslLaunchInteractive.Call(
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(unsafe.Pointer(commandUTF16)),
		uintptr(useCwd),
		uintptr(unsafe.Pointer(&exitCode)))

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WslLaunchInteractive")
	}

	if exitCode == WindowsError {
		return fmt.Errorf("error on windows' side on WslLaunchInteractive")
	}

	if exitCode != 0 {
		return &ExitError{Code: exitCode}
	}

	return nil
}
