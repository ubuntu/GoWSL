package gowsl

// This file contains windows-only API definitions and imports

import (
	"errors"
	"fmt"
	"os"
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

// startProcess replaces os.StartProcess with WSL commands.
func (c Cmd) startProcess() (process *os.Process, err error) {
	distroUTF16, err := syscall.UTF16PtrFromString(c.distro.Name())
	if err != nil {
		return nil, errors.New("failed to convert distro name to UTF16")
	}

	commandUTF16, err := syscall.UTF16PtrFromString(c.command)
	if err != nil {
		return nil, fmt.Errorf("failed to convert command %q to UTF16", c.command)
	}

	var useCwd wBOOL
	if c.UseCWD {
		useCwd = 1
	}

	var handle windows.Handle
	r1, _, _ := wslLaunch.Call(
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(unsafe.Pointer(commandUTF16)),
		uintptr(useCwd),
		c.stdinR.Fd(),
		c.stdoutW.Fd(),
		c.stderrW.Fd(),
		uintptr(unsafe.Pointer(&handle)))

	if r1 != 0 {
		return nil, fmt.Errorf("failed syscall to WslLaunch")
	}
	if handle == windows.Handle(0) {
		return nil, fmt.Errorf("syscall to WslLaunch returned a null handle")
	}

	pid, err := windows.GetProcessId(handle)
	if err != nil {
		return nil, errors.New("failed to find launched process")
	}

	return os.FindProcess(int(pid))
}
