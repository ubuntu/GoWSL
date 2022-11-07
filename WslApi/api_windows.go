// This file contains windows-specific typedefs and constants
package WslApi

import "syscall"

var (
	wslApiDll                       = syscall.NewLazyDLL("wslapi.dll")
	wslConfigureDistribution        = wslApiDll.NewProc("WslConfigureDistribution")
	wslGetDistributionConfiguration = wslApiDll.NewProc("WslGetDistributionConfiguration")
	wslLaunch                       = wslApiDll.NewProc("WslLaunch")
	wslLaunchInteractive            = wslApiDll.NewProc("WslLaunchInteractive")
	wslRegisterDistribution         = wslApiDll.NewProc("WslRegisterDistribution")
	wslUnregisterDistribution       = wslApiDll.NewProc("WslUnregisterDistribution")
)
