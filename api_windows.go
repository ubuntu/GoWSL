// This file contains windows-only API definitions and imports
package WslApi

import (
	"syscall"

	"golang.org/x/sys/windows/registry"
)

var (
	// WSL api
	wslApiDll                       = syscall.NewLazyDLL("wslapi.dll")
	wslConfigureDistribution        = wslApiDll.NewProc("WslConfigureDistribution")
	wslGetDistributionConfiguration = wslApiDll.NewProc("WslGetDistributionConfiguration")
	wslLaunch                       = wslApiDll.NewProc("WslLaunch")
	wslLaunchInteractive            = wslApiDll.NewProc("WslLaunchInteractive")
	wslRegisterDistribution         = wslApiDll.NewProc("WslRegisterDistribution")
	wslUnregisterDistribution       = wslApiDll.NewProc("WslUnregisterDistribution")
)

const (
	lxssRegistry = registry.CURRENT_USER
	lxssPath     = `Software\Microsoft\Windows\CurrentVersion\Lxss\`
)
