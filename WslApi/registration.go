package WslApi

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

// Register is a wrapper around Win32's WslRegisterDistribution
func (distro Distro) Register(rootFsPath string) error {

	r, err := distro.IsRegistered()
	if err != nil {
		return fmt.Errorf("failed to detect if '%s' is installed already", distro.Name)
	}
	if r {
		return fmt.Errorf("distro '%s' is already registered", distro.Name)
	}

	distroNameUTF16, err := syscall.UTF16PtrFromString(distro.Name)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to UTF16", distro.Name)
	}

	rootFsPathUTF16, err := syscall.UTF16PtrFromString(rootFsPath)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to UTF16", distro.Name)
	}

	r1, _, _ := wslRegisterDistribution.Call(
		uintptr(unsafe.Pointer(distroNameUTF16)),
		uintptr(unsafe.Pointer(rootFsPathUTF16)))

	if r1 != 0 {
		return fmt.Errorf("failed syscall to wslRegisterDistribution")
	}

	registered, err := distro.IsRegistered()
	if err != nil {
		return err
	}
	if !registered {
		return fmt.Errorf("distro %s was not succesfully registered", distro.Name)
	}

	return nil
}

// IsRegistered returns whether a distro is registered in WSL or not.
func (distro Distro) IsRegistered() (bool, error) {
	outp, err := exec.Command("powershell.exe", "-command", "$env:WSL_UTF8=1 ; wsl.exe --list --quiet").CombinedOutput()
	if err != nil {
		return false, err
	}

	for _, line := range strings.Fields(string(outp)) {
		if line != distro.Name {
			continue
		}
		return true, nil
	}
	return false, nil
}

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
