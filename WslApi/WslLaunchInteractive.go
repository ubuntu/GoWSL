package WslApi

import (
	"fmt"
	"syscall"
	"unsafe"
)

// LaunchInteractive is a wrapper around Win32's WslLaunchInteractive.
func (distro *Distro) LaunchInteractive(command string, useCWD bool) (ExitCode, error) {
	distroNameUTF16, err := syscall.UTF16PtrFromString(distro.Name)
	if err != nil {
		return 0, fmt.Errorf("failed to convert '%s' to UTF16", distro.Name)
	}

	commandUTF16, err := syscall.UTF16PtrFromString(command)
	if err != nil {
		return 0, fmt.Errorf("failed to convert '%s' to UTF16", command)
	}

	var useCwd wBOOL = 0
	if useCWD {
		useCwd = 1
	}

	var exitCode ExitCode

	r1, _, _ := wslLaunchInteractive.Call(
		uintptr(unsafe.Pointer(distroNameUTF16)),
		uintptr(unsafe.Pointer(commandUTF16)),
		uintptr(useCwd),
		uintptr(unsafe.Pointer(&exitCode)))

	if r1 != 0 {
		return exitCode, fmt.Errorf("failed syscall to WslLaunchInteractive")
	}

	if exitCode == WindowsError {
		return exitCode, fmt.Errorf("nonzero return value from WslLaunchInteractive (error code %d)", exitCode)
	}

	return exitCode, nil
}
