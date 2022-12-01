package wsl

// This file contains utilities to launch commands into WSL distros.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"unsafe"

	"github.com/0xrawsec/golang-utils/log"
)

// Windows' constants.
const (
	WindowsError  uint32 = 4294967295 // Underflowed -1
	ActiveProcess uint32 = 259
)

// Cmd is a wrapper around the Windows process spawned by WslLaunch.
type Cmd struct {
	// Public parameters
	Stdin  syscall.Handle
	Stdout io.Writer
	Stderr syscall.Handle
	UseCWD bool

	// Immutable parameters
	distro  *Distro
	command string

	// Pipes
	closeAfterStart []io.Closer    // IO closers to be invoked after Launching the command
	closeAfterWait  []io.Closer    // IO closers to be invoked after Waiting for the command to end
	goroutine       []func() error // Goroutines that monitor Stdout/Stderr/Stdin and copy them asyncrounously
	errch           chan error     // The gouroutines will send any error down this chanel

	// Book-keeping
	handle     syscall.Handle
	ctx        context.Context
	waitDone   chan struct{}
	exitStatus error
}

// ExitError represents a non-zero exit status from a WSL process.
// Linux's exit errors range from 0 to 255, larger numbers correspond to Windows-side errors.
type ExitError struct {
	Code uint32
}

func (m ExitError) Error() string {
	return fmt.Sprintf("exit error: %d", m.Code)
}

// Is ensures ExitErrors can be matched with errors.Is().
func (m ExitError) Is(target error) bool {
	_, ok := target.(ExitError) // nolint: errorlint
	return ok
}

// Command returns the Cmd struct to execute the named program with
// the given arguments in the same string.
//
// It sets only the command and stdin/stdout/stderr in the returned structure.
//
// The provided context is used to kill the process (by calling
// CloseHandle) if the context becomes done before the command
// completes on its own.
func (d *Distro) Command(ctx context.Context, cmd string) *Cmd {
	if ctx == nil {
		panic("nil Context")
	}
	return &Cmd{
		Stdin:   0,
		Stdout:  nil,
		Stderr:  0,
		UseCWD:  false,
		distro:  d,
		handle:  0,
		command: cmd,
		ctx:     ctx,
	}
}

// Start starts the specified command but does not wait for it to complete.
//
// The Wait method will return the exit code and release associated resources
// once the command exits.
func (c *Cmd) Start() (err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf("wsl: %v", err)
		if c.handle == 0 {
			return
		}
	}()

	distroUTF16, err := syscall.UTF16PtrFromString(c.distro.Name)
	if err != nil {
		return errors.New("failed to convert distro name to UTF16")
	}

	commandUTF16, err := syscall.UTF16PtrFromString(c.command)
	if err != nil {
		return fmt.Errorf("failed to convert command '%s' to UTF16", c.command)
	}

	var useCwd wBOOL = 0
	if c.UseCWD {
		useCwd = 1
	}

	if c.handle != 0 {
		return errors.New("already started")
	}

	if c.ctx != nil {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
		}
	}

	stdout, err := c.stdout()
	if err != nil {
		c.closeDescriptors(c.closeAfterStart)
		c.closeDescriptors(c.closeAfterWait)
		return err
	}

	r1, _, _ := wslLaunch.Call(
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(unsafe.Pointer(commandUTF16)),
		uintptr(useCwd),
		uintptr(c.Stdin),
		stdout.Fd(),
		uintptr(c.Stderr),
		uintptr(unsafe.Pointer(&c.handle)))

	if r1 != 0 {
		c.closeDescriptors(c.closeAfterStart)
		c.closeDescriptors(c.closeAfterWait)
		return fmt.Errorf("failed syscall to WslLaunch")
	}
	if c.handle == syscall.Handle(0) {
		c.closeDescriptors(c.closeAfterStart)
		c.closeDescriptors(c.closeAfterWait)
		return fmt.Errorf("syscall to WslLaunch returned a null handle")
	}

	c.closeDescriptors(c.closeAfterStart)

	// Allocating goroutines that will monitor the pipes to copy them, and collect
	// their errors into c.errch
	if len(c.goroutine) > 0 {
		c.errch = make(chan error, len(c.goroutine))
		for _, fn := range c.goroutine {
			go func(fn func() error) {
				c.errch <- fn()
			}(fn)
		}
	}

	if c.ctx != nil {
		c.waitDone = make(chan struct{})
		go func() {
			select {
			case <-c.waitDone:
				return
			case <-c.ctx.Done():
			}
			err := c.kill()
			if err != nil {
				log.Warnf("wsl: Failed to kill process: %v", err)
			}
		}()
	}

	return nil
}

func (c *Cmd) stdout() (f *os.File, err error) {
	return c.writerDescriptor(c.Stdout)
}

func (c *Cmd) closeDescriptors(closers []io.Closer) {
	for _, fd := range closers {
		fd.Close()
	}
}

// Adapted from exec/exec.go
func (c *Cmd) writerDescriptor(writer io.Writer) (f *os.File, err error) {
	if writer == nil {
		f, err = os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			return
		}
		c.closeAfterStart = append(c.closeAfterStart, f)
		return
	}

	if f, ok := writer.(*os.File); ok {
		return f, nil
	}

	pr, pw, err := os.Pipe()
	if err != nil {
		return
	}

	c.closeAfterStart = append(c.closeAfterStart, pw)
	c.closeAfterWait = append(c.closeAfterWait, pr)
	c.goroutine = append(c.goroutine, func() error {
		_, err := io.Copy(writer, pr)
		pr.Close() // in case io.Copy stopped due to write error
		return err
	})
	return pw, nil
}

// Wait waits for the command to exit and waits for any copying to
// stdin or copying from stdout or stderr to complete.
//
// The command must have been started by Start.
//
// The returned error is nil if the command runs, has no problems
// copying stdin, stdout, and stderr, and exits with a zero exit
// status.
//
// If the command fails to run or doesn't complete successfully, the
// error is of type ExitError. Other error types may be
// returned for I/O problems.
//
// If any of c.Stdin, c.Stdout or c.Stderr are not an *os.File, Wait also waits
// for the respective I/O loop copying to or from the process to complete.
//
// Wait releases any resources associated with the Cmd.
func (c *Cmd) Wait() (err error) {
	defer func() {
		if err == nil {
			return
		}
		if errors.Is(err, ExitError{}) {
			return
		}
		err = fmt.Errorf("error during Distro.Wait: %v", err)
	}()

	defer c.close()
	r1, err := syscall.WaitForSingleObject(c.handle, syscall.INFINITE)

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WaitForSingleObject: %v", err)
	}

	if c.waitDone != nil {
		close(c.waitDone)
	}

	// Collecting errors from pipe redirections
	var copyError error
	for range c.goroutine {
		if e := <-c.errch; e != nil && copyError == nil {
			copyError = err
		}
	}

	c.closeDescriptors(c.closeAfterWait)

	if err := c.status(); err != nil {
		return err
	}

	return copyError
}

// Run starts the specified WslProcess and waits for it to complete.
//
// The returned error is nil if the command runs and exits with a zero exit status.
//
// If the command fails to run or doesn't complete successfully, the error is of type *ExitError.
func (c *Cmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

// close closes a WslProcess. If it was still running, it is terminated,
// although its Linux counterpart may not.
func (c *Cmd) close() error {
	e := syscall.CloseHandle(c.handle)
	if e != nil {
		c.handle = 0
	}
	return e
}

// status querries Windows for the process' status.
func (c *Cmd) status() error {
	if c.exitStatus != nil {
		return c.exitStatus
	}

	var exit uint32
	err := syscall.GetExitCodeProcess(c.handle, &exit)
	if err != nil {
		return fmt.Errorf("failed to retrieve exit status: %v", err)
	}
	if exit == WindowsError {
		return errors.New("failed to Launch Linux command due to Windows-side error")
	}
	if exit != 0 {
		return &ExitError{Code: exit}
	}
	return nil
}

// kill gets the exit status before closing the process, without checking
// if it has finished or not.
func (c *Cmd) kill() error {
	// If the exit code is ActiveProcess, we write a more useful error message
	// indicating it was interrupted.
	c.exitStatus = func() error {
		e := c.status()

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
		return errors.New("process was closed before finishing")
	}()

	return syscall.TerminateProcess(c.handle, ActiveProcess)
}
