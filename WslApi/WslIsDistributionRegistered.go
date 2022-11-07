package WslApi

import (
	"os/exec"
	"strings"
)

/*
Mocks the WslIsDistributionRegistered Win32 API endpoint. It cannot be invoked
directly because there is no way of obtaining the return value.

BOOL WslIsDistributionRegistered(
  PCWSTR distributionName
);

https://learn.microsoft.com/en-us/windows/win32/api/wslapi/nf-wslapi-wslisdistributionregistered
*/

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
