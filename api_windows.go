package gowsl

// This file contains windows-only API definitions and imports

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	// WSL api.
	wslAPIDll                       = syscall.NewLazyDLL("wslapi.dll")
	wslConfigureDistribution        = wslAPIDll.NewProc("WslConfigureDistribution")
	wslGetDistributionConfiguration = wslAPIDll.NewProc("WslGetDistributionConfiguration")
	wslLaunch                       = wslAPIDll.NewProc("WslLaunch")
	wslLaunchInteractive            = wslAPIDll.NewProc("WslLaunchInteractive")
	wslRegisterDistribution         = wslAPIDll.NewProc("WslRegisterDistribution")
	wslUnregisterDistribution       = wslAPIDll.NewProc("WslUnregisterDistribution")
)

const (
	lxssRegistry = registry.CURRENT_USER
	lxssPath     = `Software\Microsoft\Windows\CurrentVersion\Lxss\`
)

// Windows' typedefs.
type wBOOL = int     // Windows' BOOL
type wULONG = uint32 // Windows' ULONG
type char = byte     // Windows' CHAR (which is the same as C's char)

func coTaskMemFree(p unsafe.Pointer) {
	windows.CoTaskMemFree(p)
}
