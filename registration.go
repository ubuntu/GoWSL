package wsl

// This file contains utilities to create, destroy, stop WSL distros,
// as well as utilities to query this status.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// Register is a wrapper around Win32's WslRegisterDistribution
func (d *Distro) Register(rootFsPath string) error {
	// TODO: decorate the function

	rootFsPath, err := fixPath(rootFsPath)
	if err != nil {
		return err
	}

	r, err := d.IsRegistered()
	if err != nil {
		return fmt.Errorf("failed to detect if '%q' is installed already", d.Name)
	}
	if r {
		return fmt.Errorf("'%q' is already registered", d.Name)
	}

	distroUTF16, err := syscall.UTF16PtrFromString(d.Name)
	if err != nil {
		return fmt.Errorf("failed to convert '%q' to UTF16", d.Name)
	}

	rootFsPathUTF16, err := syscall.UTF16PtrFromString(rootFsPath)
	if err != nil {
		return fmt.Errorf("failed to convert '%q' to UTF16", rootFsPath)
	}

	r1, _, _ := wslRegisterDistribution.Call(
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(unsafe.Pointer(rootFsPathUTF16)))

	if r1 != 0 {
		return fmt.Errorf("failed syscall to wslRegisterDistribution")
	}

	return nil
}

// RegisteredDistros returns a slice of the registered distros
func RegisteredDistros() ([]Distro, error) {
	return registeredInstances()
}

// IsRegistered returns whether an distro is registered in WSL or not.
func (d Distro) IsRegistered() (bool, error) {
	distros, err := RegisteredDistros()
	if err != nil {
		return false, err
	}

	for _, i := range distros {
		if i.Name != d.Name {
			continue
		}
		return true, nil
	}
	return false, nil
}

// Unregister is a wrapper around Win32's WslUnregisterDistribution.
func (d *Distro) Unregister() error {
	r, err := d.IsRegistered()
	if err != nil {
		return fmt.Errorf("failed to detect if '%q' is installed already", d.Name)
	}
	if !r {
		return fmt.Errorf("WSL distro '%q' is not registered", d.Name)
	}

	distroUTF16, err := syscall.UTF16PtrFromString(d.Name)
	if err != nil {
		return fmt.Errorf("failed to convert '%q' to UTF16", d.Name)
	}

	r1, _, _ := wslUnregisterDistribution.Call(uintptr(unsafe.Pointer(distroUTF16)))

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WslLaunchInteractive")
	}
	return nil
}

// WslRegisterDistribuion is a bit picky with the format.
func fixPath(relative string) (string, error) {
	abs, err := filepath.Abs(filepath.FromSlash(relative))
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(abs); errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("file %q does not exist", abs)
	}
	return abs, nil
}
