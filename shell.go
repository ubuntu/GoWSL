package wsl

import (
	"fmt"
	"syscall"
	"unsafe"
)

type shellOptions struct {
	command string
	useCWD  bool
}

func UseCWD() func(*shellOptions) {
	return func(o *shellOptions) {
		o.useCWD = true
	}
}

func WithCommand(cmd string) func(*shellOptions) {
	return func(o *shellOptions) {
		o.command = cmd
	}
}

// Shell is a wrapper around Win32's WslLaunchInteractive.
// This is a syncronous, blocking call.
func (i *Distro) Shell(opts ...func(*shellOptions)) error {

	options := shellOptions{
		command: "",
		useCWD:  false,
	}
	for _, o := range opts {
		o(&options)
	}

	instanceUTF16, err := syscall.UTF16PtrFromString(i.Name)
	if err != nil {
		return fmt.Errorf("failed to convert %q to UTF16", i.Name)
	}

	commandUTF16, err := syscall.UTF16PtrFromString(options.command)
	if err != nil {
		return fmt.Errorf("failed to convert %q to UTF16", options.command)
	}

	var useCwd wBOOL = 0
	if options.useCWD {
		useCwd = 1
	}

	var exitCode ExitCode

	r1, _, _ := wslLaunchInteractive.Call(
		uintptr(unsafe.Pointer(instanceUTF16)),
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
