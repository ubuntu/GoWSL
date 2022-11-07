package WslApi

import (
	"fmt"
	"syscall"
	"unsafe"
)

/*
HRESULT WslRegisterDistribution(
  [in] PCWSTR distributionName,
  [in] PCWSTR tarGzFilename
);

https://learn.microsoft.com/en-us/windows/win32/api/wslapi/nf-wslapi-wslregisterdistribution
*/
var wslRegisterDistribution = wslApiDll.NewProc("WslRegisterDistribution")

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
		return fmt.Errorf("failed syscall to WslLaunchInteractive")
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
