// This file contains windows-specific typedefs and constants
package WslApi

import (
	"os/exec"
	"syscall"
)

var (
	wslApiDll                       = syscall.NewLazyDLL("wslapi.dll")
	wslConfigureDistribution        = wslApiDll.NewProc("WslConfigureDistribution")
	wslGetDistributionConfiguration = wslApiDll.NewProc("WslGetDistributionConfiguration")
	wslLaunch                       = wslApiDll.NewProc("WslLaunch")
	wslLaunchInteractive            = wslApiDll.NewProc("WslLaunchInteractive")
	wslRegisterDistribution         = wslApiDll.NewProc("WslRegisterDistribution")
	wslUnregisterDistribution       = wslApiDll.NewProc("WslUnregisterDistribution")
)

func Shutdown() error {
	return exec.Command("wsl.exe", "--shutdown").Run()
}

func (distro Distro) Terminate() error {
	return exec.Command("wsl.exe", "--terminate", distro.Name).Run()
}
