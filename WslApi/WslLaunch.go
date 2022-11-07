package WslApi

import (
	"fmt"
	"math"
	"syscall"
	"time"
	"unsafe"
)

// WslProcess is a wrapper around the Windows process spawned by WslLaunch
type WslProcess struct {
	syscall.Handle
}

// LaunchInteractive is a wrapper around Win32's WslLaunchInteractive.
// Note that the returned process is the Windows process, and closing it will not close the Linux process it invoked.
func (distro *Distro) Launch(command string, useCWD bool, stdIn syscall.Handle, stdOut syscall.Handle, stdErr syscall.Handle) (WslProcess, error) {
	distroNameUTF16, err := syscall.UTF16PtrFromString(distro.Name)
	if err != nil {
		return WslProcess{0}, fmt.Errorf("failed to convert '%s' to UTF16", distro.Name)
	}

	commandUTF16, err := syscall.UTF16PtrFromString(command)
	if err != nil {
		return WslProcess{0}, fmt.Errorf("failed to convert '%s' to UTF16", command)
	}

	var useCwd wBOOL = 0
	if useCWD {
		useCwd = 1
	}

	var process syscall.Handle = 0

	r1, _, _ := wslLaunch.Call(
		uintptr(unsafe.Pointer(distroNameUTF16)),
		uintptr(unsafe.Pointer(commandUTF16)),
		uintptr(useCwd),
		uintptr(stdIn),
		uintptr(stdOut),
		uintptr(stdErr),
		uintptr(unsafe.Pointer(&process)))

	if r1 != 0 {
		return WslProcess{0}, fmt.Errorf("failed syscall to WslLaunch")
	}

	return WslProcess{Handle: process}, nil
}

// Wait blocks execution until the process finishes and returns the process exit status.
func (process WslProcess) Wait(timeout time.Duration) (ExitCode, error) {
	if process.Handle == 0 {
		return 0, fmt.Errorf("cannot wait on a null process")
	}

	t, err := toWin32Miliseconds(timeout)
	if err != nil {
		return 0, err
	}

	r1, _ := syscall.WaitForSingleObject(process.Handle, t)
	if r1 != 0 {
		return 0, fmt.Errorf("failed syscall to WaitForSingleObject")
	}

	return process.GetStatus()
}

// AsyncWait creates two channels to asyncrounously get the result of a WslProcess.
func (process WslProcess) AsyncWait(timeout time.Duration) (chan ExitCode, chan error) {
	exitStatus := make(chan ExitCode)
	err := make(chan error)

	go func() {
		code, e := process.Wait(timeout)
		err <- e
		exitStatus <- code
	}()

	return exitStatus, err
}

// Close closes a WslProcess and returns its exit status.
func (process *WslProcess) Close() (exitStatus ExitCode, err error) {
	if process.Handle == 0 {
		return 0, fmt.Errorf("cannot close a null process")
	}

	if exitStatus, err = process.GetStatus(); err != nil {
		return exitStatus, err
	}

	err = syscall.CloseHandle(process.Handle)
	process.Handle = 0

	return exitStatus, err
}

// GetStatus querries a process to get its status.
func (process WslProcess) GetStatus() (exitStatus ExitCode, err error) {
	err = syscall.GetExitCodeProcess(process.Handle, &exitStatus)
	return exitStatus, err
}

// toWin32Miliseconds converts time.Duration into a uint32 and performs bounds checking
func toWin32Miliseconds(time time.Duration) (uint32, error) {
	t := time.Milliseconds()
	if t < 0 || t > math.MaxUint32 {
		return uint32(t), fmt.Errorf("out-of-bounds narrowing conversion: Win32 API time must fit in a uint32")
	}
	return uint32(t), nil
}
