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
type wBOOL = int // Windows' BOOL
type char = byte // Windows' CHAR (which is the same as C's char)

func coTaskMemFree(p unsafe.Pointer) {
	windows.CoTaskMemFree(p)
}

// Extracting the type of file a handle points to
// https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-getfiletype
type winFileType int

const (
	fileTypeChar    winFileType = 0x0002 // The specified file is a character file, typically an LPT device or a console.
	fileTypeDisk    winFileType = 0x0001 // The specified file is a disk file.
	fileTypePipe    winFileType = 0x0003 // The specified file is a socket, a named pipe, or an anonymous pipe.
	fileTypeRemote  winFileType = 0x8000 // Unused.
	fileTypeUnknown winFileType = 0x0000 // Either the type of the specified file is unknown, or the function failed.
)

func fileType(f *os.File) (winFileType, error) {
	n, err := windows.GetFileType(windows.Handle(f.Fd()))
	if err != nil {
		return fileTypeUnknown, err
	}
	return winFileType(n), nil
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
