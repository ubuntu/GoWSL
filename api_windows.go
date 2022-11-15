package wsl

// This file contains windows-only API definitions and imports

import (
	"regexp"
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

// Windows' typedefs
type wBOOL = int       // Windows' BOOL
type wULONG = uint32   // Windows' ULONG
type ExitCode = uint32 // Windows' DWORD
type char = byte       // Windows' CHAR (which is the same as C's char)

// Replaces path.IsAbs, which assumes you're on linux
func pathIsAbs(p string) bool {
	pattern := regexp.MustCompile(`^[A-Z]:*.*`)
	return pattern.Match([]byte(p))
}
