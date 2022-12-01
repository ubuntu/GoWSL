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

// Cmd is a wrapper around the Windows process spawned by WslLaunch. It is not thread-safe.
//
// A Cmd cannot be reused after calling its Run method.
type Cmd struct {
	// Public parameters
	Stdin  syscall.Handle
	Stdout io.Writer // Writer to write stdout into
	Stderr io.Writer // Writer to write stdout into
	UseCWD bool      // Whether WSL is launched in the current working directory (true) or the home directory (false)

	// Immutable parameters
	distro  *Distro // The distro that the command will be launched into.
	command string  // The command to be launched

	// Pipes
	closeAfterStart []io.Closer    // IO closers to be invoked after Launching the command
	closeAfterWait  []io.Closer    // IO closers to be invoked after Waiting for the command to end
	goroutine       []func() error // Goroutines that monitor Stdout/Stderr/Stdin and copy them asyncrounously
	errch           chan error     // The gouroutines will send any error down this chanel

	// File descriptors for pipes. These are analogous to (*exec.Cmd).childFiles[:3]
	// stdinF  *os.File // File for stdin
	stdoutW *os.File // File that acts as a writer for WSL to write stdout into
	stderrW *os.File // File that acts as a writer for WSL to write stderr into

	// Book-keeping
	handle     syscall.Handle // The windows handle to the WSL process
	finished   bool           // Flag to fail nicely when Wait is invoked twicw
	exitStatus *uint32        // Exit status of the process. Cached because it cannot be read after the preocess is closed.

	// Context management
	ctx      context.Context // Context to kill the process before it finishes
	waitDone chan struct{}   // This chanel prevents the context from attempting to kill the process when it is closed already
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
		Stderr:  nil,
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
		c.closeDescriptors(c.closeAfterStart)
		c.closeDescriptors(c.closeAfterWait)
	}()

	distroUTF16, err := syscall.UTF16PtrFromString(c.distro.Name)
	if err != nil {
		return errors.New("failed to convert distro name to UTF16")
	}

	commandUTF16, err := syscall.UTF16PtrFromString(c.command)
	if err != nil {
		return fmt.Errorf("failed to convert command '%s' to UTF16", c.command)
	}

	var useCwd wBOOL
	if c.UseCWD {
		useCwd = 1
	}

	if c.handle != 0 {
		return errors.New("already started")
	}

	if c.ctx != nil {
		select {
		case <-c.ctx.Done():
			c.closeDescriptors(c.closeAfterStart)
			c.closeDescriptors(c.closeAfterWait)
			return c.ctx.Err()
		default:
		}
	}

	type F func(*Cmd) error
	for _, setupFd := range []F{(*Cmd).stdin, (*Cmd).stdout, (*Cmd).stderr} {
		err := setupFd(c)
		if err != nil {
			c.closeDescriptors(c.closeAfterStart)
			c.closeDescriptors(c.closeAfterWait)
			return err
		}
	}

	r1, _, _ := wslLaunch.Call(
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(unsafe.Pointer(commandUTF16)),
		uintptr(useCwd),
		uintptr(c.Stdin),
		c.stdoutW.Fd(),
		c.stderrW.Fd(),
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

func (c *Cmd) stdin() error {
	// TODO
	return nil
}

func (c *Cmd) stdout() error {
	w, e := c.writerDescriptor(c.Stdout)
	if e == nil {
		c.stdoutW = w
	}
	return e
}

func (c *Cmd) stderr() error {
	// Case where Stdout and Stderr are the same
	if c.Stderr != nil && interfaceEqual(c.Stdout, c.Stderr) {
		c.stderrW = c.stdoutW
		return nil
	}
	// Different stdout and stderr
	w, e := c.writerDescriptor(c.Stderr)
	if e == nil {
		c.stderrW = w
	}
	return e
}

// interfaceEqual protects against panics from doing equality tests on
// two interfaces with non-comparable underlying types.
func interfaceEqual(a, b any) bool {
	defer func() {
		_ = recover()
	}()
	return a == b
}

func (c *Cmd) closeDescriptors(closers []io.Closer) {
	for _, fd := range closers {
		fd.Close()
	}
}

// writerDescriptor connects an arbitrary writer to an os pipe's reader,
// and returns this pipe's writer as a file.
//
// Taken from exec/exec.go.
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
func (c *Cmd) Wait() error {
	if c.handle == 0 {
		return errors.New("in Distro.Wait: not started")
	}
	if c.finished {
		return errors.New("in Distro.Wait: already called")
	}
	c.finished = true

	status, waitError := c.waitProcess()
	// Will deal with waitError after releasing resources

	// Releasing goroutines in charge of listening to context cancellation
	if c.waitDone != nil {
		close(c.waitDone)
	}
	c.exitStatus = &status

	// Releasing goroutines in charge of pipe redirection. Collect
	// their errors.
	var copyError error
	for range c.goroutine {
		if err := <-c.errch; err != nil && copyError == nil {
			copyError = err
		}
	}

	// Releasing pipes
	c.closeDescriptors(c.closeAfterWait)

	// Reporting the errors in order of importance.
	if waitError != nil {
		return waitError
	}

	// Custom errors for particular exit status
	if status == WindowsError {
		return errors.New("command failed due to Windows-side error")
	}
	if status == ActiveProcess { // Process was most likely interrupted by context
		if err := c.ctx.Err(); err != nil {
			return err
		}
	}
	if status != 0 {
		return &ExitError{Code: status}
	}
	return copyError
}

func (c *Cmd) waitProcess() (uint32, error) {
	event, statusError := syscall.WaitForSingleObject(c.handle, syscall.INFINITE)
	if statusError != nil {
		return WindowsError, fmt.Errorf("failed syscall to WaitForSingleObject: %v", statusError)
	}
	if event != syscall.WAIT_OBJECT_0 {
		return WindowsError, fmt.Errorf("failed syscall to WaitForSingleObject, non-zero exit status %d", event)
	}

	status, statusError := c.status()
	ok := statusError == nil && status == 0

	if err := syscall.CloseHandle(c.handle); !ok && err != nil {
		return WindowsError, err
	}
	return status, statusError
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

// status querries Windows for the process' status.
func (c *Cmd) status() (exit uint32, err error) {
	// Retrieving from cache in case the process has been closed
	if c.exitStatus != nil {
		return *c.exitStatus, nil
	}

	err = syscall.GetExitCodeProcess(c.handle, &exit)
	if err != nil {
		return WindowsError, fmt.Errorf("failed to retrieve exit status: %v", err)
	}
	return exit, nil
}

// kill gets the exit status before closing the process, without checking
// if it has finished or not.
func (c *Cmd) kill() error {
	status, err := c.status()
	c.exitStatus = nil
	if err == nil {
		c.exitStatus = &status
	}
	return syscall.TerminateProcess(c.handle, ActiveProcess)
}
