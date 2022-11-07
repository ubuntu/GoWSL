package WslApi

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Register is a wrapper around Win32's WslUnregisterDistribution.
func (distro *Distro) Unregister() error {
	r, err := distro.IsRegistered()
	if err != nil {
		return fmt.Errorf("failed to detect if '%s' is installed already", distro.Name)
	}
	if !r {
		return fmt.Errorf("distro '%s' is not registered", distro.Name)
	}

	distroNameUTF16, err := syscall.UTF16PtrFromString(distro.Name)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to UTF16", distro.Name)
	}

	r1, _, _ := wslUnregisterDistribution.Call(uintptr(unsafe.Pointer(distroNameUTF16)))

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WslLaunchInteractive")
	}
	return nil
}
