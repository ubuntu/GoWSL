package wsl

// This file contains utilities to launch commands into WSL instances.

import (
	"fmt"
	"syscall"
	"unsafe"
)

// WslProcess is a wrapper around the Windows process spawned by WslLaunch
type WslProcess struct {
	// Public parameters
	Stdout syscall.Handle
	Stdin  syscall.Handle
	Stderr syscall.Handle
	UseCWD bool

	// Immutable parameters
	instance *Instance
	command  string

	// Book-keeping
	handle syscall.Handle
}

type ExitError struct {
	Code ExitCode
}

func (m *ExitError) Error() string {
	return fmt.Sprintf("exit error: %d", m.Code)
}

func (i *Instance) NewWslProcess(command string) WslProcess {
	return WslProcess{
		Stdin:    syscall.Stdin,
		Stdout:   syscall.Stdout,
		Stderr:   syscall.Stderr,
		UseCWD:   false,
		instance: i,
		handle:   0,
		command:  command,
	}
}

// LaunchInteractive is a wrapper around Win32's WslLaunchInteractive.
// This is a syncronous, blocking call.
func (i *Instance) LaunchInteractive(command string, useCWD bool) error {
	instanceUTF16, err := syscall.UTF16PtrFromString(i.Name)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to UTF16", i.Name)
	}

	commandUTF16, err := syscall.UTF16PtrFromString(command)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to UTF16", command)
	}

	var useCwd wBOOL = 0
	if useCWD {
		useCwd = 1
	}

	var exitCode ExitCode

	r1, _, _ := wslLaunchInteractive.Call(
		uintptr(unsafe.Pointer(instanceUTF16)),
		uintptr(unsafe.Pointer(commandUTF16)),
		uintptr(useCwd),
		uintptr(unsafe.Pointer(&exitCode)))

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WslLaunchInteractive")
	}

	if exitCode == WindowsError {
		return fmt.Errorf("error on windows' side on WslLaunchInteractive")
	}

	if exitCode != 0 {
		return &ExitError{Code: exitCode}
	}

	return nil
}

// LaunchInteractive is a wrapper around Win32's WslLaunchInteractive.
// It launches a process asyncronously and returns a handle to it.
// Note that the returned process is the Windows process, and closing it will not close the Linux process it invoked.
func (i *Instance) Launch(command string, useCWD bool, stdIn syscall.Handle, stdOut syscall.Handle, stdErr syscall.Handle) (WslProcess, error) {
	process := i.NewWslProcess(command)
	return process, process.Start()
}

// Start starts the specified WslProcess but does not wait for it to complete.
//
// The Wait method will return the exit code and release associated resources
// once the command exits.
func (p *WslProcess) Start() error {
	instanceUTF16, err := syscall.UTF16PtrFromString(p.instance.Name)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to UTF16", p.instance)
	}

	commandUTF16, err := syscall.UTF16PtrFromString(p.command)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to UTF16", p.command)
	}

	var useCwd wBOOL = 0
	if p.UseCWD {
		useCwd = 1
	}

	r1, _, _ := wslLaunch.Call(
		uintptr(unsafe.Pointer(instanceUTF16)),
		uintptr(unsafe.Pointer(commandUTF16)),
		uintptr(useCwd),
		uintptr(p.Stdin),
		uintptr(p.Stdout),
		uintptr(p.Stderr),
		uintptr(unsafe.Pointer(&p.handle)))

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WslLaunch")
	}
	return nil
}

// Wait blocks execution until the process finishes and returns the process exit status.
//
// The returned error is nil if the command runs and exits with a zero exit status.
//
// If the command fails to run or doesn't complete successfully, the error is of type *ExitError.
func (p WslProcess) Wait() error {
	defer p.Close()
	r1, error := syscall.WaitForSingleObject(p.handle, syscall.INFINITE)
	if r1 != 0 {
		return fmt.Errorf("failed syscall to WaitForSingleObject: %v", error)
	}

	return p.queryStatus()
}

// Run starts the specified WslProcess and waits for it to complete.
//
// The returned error is nil if the command runs and exits with a zero exit status.
//
// If the command fails to run or doesn't complete successfully, the error is of type *ExitError.
func (p *WslProcess) Run() error {
	if err := p.Start(); err != nil {
		return err
	}
	return p.Wait()
}

// Close closes a WslProcess. If it was still running, it is terminated,
// although its Linux counterpart may not.
func (p *WslProcess) Close() error {
	defer func() {
		p.handle = 0
	}()
	return syscall.CloseHandle(p.handle)
}

// queryStatus querries Windows for the process' status.
func (p *WslProcess) queryStatus() error {
	exit := ExitCode(0)
	err := syscall.GetExitCodeProcess(p.handle, &exit)
	if err != nil {
		return err
	}
	if exit != 0 {
		return &ExitError{Code: exit}
	}
	return nil
}
