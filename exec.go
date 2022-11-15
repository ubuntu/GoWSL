package wsl

// This file contains utilities to launch commands into WSL distros.

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

// Windows' constants
const (
	WindowsError  ExitCode = 4294967295 // Underflowed -1
	ActiveProcess ExitCode = 259
)

// Cmd is a wrapper around the Windows process spawned by WslLaunch
type Cmd struct {
	// Public parameters
	Stdout syscall.Handle
	Stdin  syscall.Handle
	Stderr syscall.Handle
	UseCWD bool

	// Immutable parameters
	distro  *Distro
	command string

	// Book-keeping
	handle     syscall.Handle
	ctx        context.Context
	waitDone   chan struct{}
	exitStatus error
}

type ExitError struct {
	Code ExitCode
}

func (m *ExitError) Error() string {
	return fmt.Sprintf("exit error: %d", m.Code)
}

// Command returns the Cmd struct to execute the named program with
// the given arguments in the same string.
//
// It sets only the command and stdin/stdout/stderr in the returned structure.
func (i *Distro) Command(command string) *Cmd {
	return &Cmd{
		Stdin:   syscall.Stdin,
		Stdout:  syscall.Stdout,
		Stderr:  syscall.Stderr,
		UseCWD:  false,
		distro:  i,
		handle:  0,
		command: command,
	}
}

// Start starts the specified WslProcess but does not wait for it to complete.
//
// The Wait method will return the exit code and release associated resources
// once the command exits.
func (p *Cmd) Start() error {
	distroUTF16, err := syscall.UTF16PtrFromString(p.distro.Name)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to UTF16", p.distro)
	}

	commandUTF16, err := syscall.UTF16PtrFromString(p.command)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to UTF16", p.command)
	}

	var useCwd wBOOL = 0
	if p.UseCWD {
		useCwd = 1
	}

	if p.ctx != nil {
		select {
		case <-p.ctx.Done():
			return p.ctx.Err()
		default:
		}
	}

	r1, _, _ := wslLaunch.Call(
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(unsafe.Pointer(commandUTF16)),
		uintptr(useCwd),
		uintptr(p.Stdin),
		uintptr(p.Stdout),
		uintptr(p.Stderr),
		uintptr(unsafe.Pointer(&p.handle)))

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WslLaunch")
	}

	if p.ctx != nil {
		p.waitDone = make(chan struct{})
		// This goroutine monitors the status of the context to kill the process if needed
		go func() {
			select {
			case <-p.ctx.Done():
				p.kill()
			case <-p.waitDone:
			}
		}()
	}

	return nil
}

// Wait blocks execution until the process finishes and returns the process exit status.
//
// The returned error is nil if the command runs and exits with a zero exit status.
//
// If the command fails to run or doesn't complete successfully, the error is of type *ExitError.
func (p *Cmd) Wait() error {
	defer p.close()
	r1, error := syscall.WaitForSingleObject(p.handle, syscall.INFINITE)

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WaitForSingleObject: %v", error)
	}

	if p.waitDone != nil {
		close(p.waitDone)
	}

	return p.queryStatus()
}

// Run starts the specified WslProcess and waits for it to complete.
//
// The returned error is nil if the command runs and exits with a zero exit status.
//
// If the command fails to run or doesn't complete successfully, the error is of type *ExitError.
func (p *Cmd) Run() error {
	if err := p.Start(); err != nil {
		return fmt.Errorf("failed to start command: %v", err)
	}
	return p.Wait()
}

// close closes a WslProcess. If it was still running, it is terminated,
// although its Linux counterpart may not.
func (p *Cmd) close() error {
	defer func() {
		p.handle = 0
	}()

	return syscall.CloseHandle(p.handle)
}

// CommandContext is like Command but includes a context.
//
// The provided context is used to kill the process (by calling
// CloseHandle) if the context becomes done before the command
// completes on its own.
func (d *Distro) CommandContext(ctx context.Context, cmdStr string) *Cmd {
	if ctx == nil {
		panic("nil Context")
	}
	cmd := d.Command(cmdStr)
	cmd.ctx = ctx
	return cmd
}

// queryStatus querries Windows for the process' status.
func (p *Cmd) queryStatus() error {
	if p.exitStatus != nil {
		return p.exitStatus
	}

	exit := ExitCode(0)
	err := syscall.GetExitCodeProcess(p.handle, &exit)
	if err != nil {
		return fmt.Errorf("failed to retrieve exit status: %v", err)
	}
	if exit == WindowsError {
		return errors.New("something went wrong Windows-side")
	}
	if exit != 0 {
		return &ExitError{Code: exit}
	}
	return nil
}

// kill gets the exit status before closing the process, without checking
// if it has finsihed or not.
func (p *Cmd) kill() error {

	// If the exit code is ActiveProcess, we write a more useful error message
	// indicating it was interrupted.
	p.exitStatus = func() error {
		e := p.queryStatus()

		if e == nil {
			return nil
		}
		var asExitError *ExitError
		if !errors.As(e, &asExitError) {
			return e
		}
		if asExitError.Code != ActiveProcess {
			return e
		}
		return errors.New("process was closed before finshing")
	}()

	return syscall.TerminateProcess(p.handle, ActiveProcess)
}
