package WslApi

import (
	"fmt"
	"syscall"
	"unsafe"
)

/*
HRESULT WslConfigureDistribution(
  PCWSTR                 distributionName,
  ULONG                  defaultUID,
  WSL_DISTRIBUTION_FLAGS wslDistributionFlags
);

https://learn.microsoft.com/en-us/windows/win32/api/wslapi/nf-wslapi-wslconfiguredistribution
*/
var wslConfigureDistribution = wslApiDll.NewProc("WslConfigureDistribution")

// Configure is a wrapper around Win32's WslConfigureDistribution.
// Note that only the following config is mutable:
//  - DefaultUID
//  - InteropEnabled
//  - PathAppended
//  - DriveMountingEnabled
func (distro *Distro) Configure(config Configuration) error {

	distroNameUTF16, err := syscall.UTF16PtrFromString(distro.Name)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to UTF16", distro.Name)
	}

	flags, err := config.packFlags()
	if err != nil {
		return err
	}

	r1, _, _ := wslConfigureDistribution.Call(
		uintptr(unsafe.Pointer(distroNameUTF16)),
		uintptr(config.DefaultUID),
		uintptr(flags),
	)

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WslConfigureDistribution")
	}

	return nil
}
