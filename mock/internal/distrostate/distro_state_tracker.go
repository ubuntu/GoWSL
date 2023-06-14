// Package distrostate implements the mocking of the state of the distro
// (i.e. running, stopped, etc.).
package distrostate

import (
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// DistroState tracks whether a dsitro is active or not.
type DistroState struct {
	// running indicates whether the distro is running or not.
	running bool

	// terminateTimer stops the distro 8 seconds after no processes are left.
	terminateTimer *time.Timer

	// processes is a set of attached processes.
	processes map[*os.Process]struct{}

	// processes is a set of running interactive shells.
	// The moment this set becomes empty, the terminate timer is started.
	shells map[*Shell]struct{}

	// flag to avoid races where you may attach a process after the distro has been uninstalled.
	uninstalled bool

	mu sync.RWMutex
}

// Shell mocks a shell's lifetime: it can be started, closed, and waited for.
// The distro will be kept running so long as the shell is open.
// Terminating or unregistering the distro will close the shell.
type Shell struct {
	done   chan struct{}
	closed atomic.Bool
}

// Close mocks the killing of the shell process.
func (s *Shell) Close() {
	if updated := s.closed.CompareAndSwap(false, true); updated {
		close(s.done)
	}
}

// Wait waits until the shell is closed.
func (s *Shell) Wait() uint32 {
	<-s.done

	// Apparently it always succeeds
	return 0
}

// New creates a new disro state with state Stopped.
func New() *DistroState {
	return &DistroState{
		processes: make(map[*os.Process]struct{}),
		shells:    make(map[*Shell]struct{}),
	}
}

// IsRunning returns whether the distro is running this moment.
func (t *DistroState) IsRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.running
}

// Touch resets the terminate timer if there was one.
func (t *DistroState) Touch() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.uninstalled {
		return errors.New("distro unregistered")
	}

	t.running = true
	t.cancelTimer()
	t.refresh()

	return nil
}

// Terminate mocks the behaviour of `wsl.exe --terminate <distro>`.
func (t *DistroState) Terminate() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.terminate()
}

// MarkUninstalled kills all processes, closes all shells, and marks the distro
// as uninstalled.
func (t *DistroState) MarkUninstalled() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.uninstalled {
		return errors.New("distro unregistered")
	}

	_ = t.terminate()
	t.uninstalled = true

	return nil
}

// AttachProcess wakes up the distro and attaches the process to it.
// Attached processes are killed if the distro is terminated.
func (t *DistroState) AttachProcess(p *os.Process) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.uninstalled {
		return errors.New("distro unregistered")
	}

	t.cancelTimer()

	t.processes[p] = struct{}{}
	t.running = true

	return nil
}

// NewShell wakes up the distro and attaches the process to it.
// Attached processes are killed if the distro is terminated.
func (t *DistroState) NewShell() (*Shell, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.uninstalled {
		return nil, errors.New("distro unregistered")
	}

	t.cancelTimer()

	s := &Shell{done: make(chan struct{})}
	t.shells[s] = struct{}{}

	return s, nil
}

// terminate kills all attached processes and sets running to false
//
// Use under a write mutex.
func (t *DistroState) terminate() error {
	if t.uninstalled {
		return errors.New("distro unregistered")
	}

	// Stopping distro
	t.cancelTimer()
	t.running = false

	// Killing processes
	for p := range t.processes {
		_ = p.Kill()
	}
	t.processes = make(map[*os.Process]struct{})

	// Closing shells
	for p := range t.shells {
		p.Close()
	}
	t.shells = make(map[*Shell]struct{})

	return nil
}

// refresh checks the count of active shells. If the count
// reaches zero, the terminate timer is started.
//
// Use under a write mutex.
func (t *DistroState) refresh() {
	// Garbage-collect shells
	for s := range t.shells {
		if s.closed.Load() {
			delete(t.shells, s)
		}
	}

	// Nothing to do if the distro is already off.
	if !t.running {
		return
	}

	// Nothing to do if there are shells running.
	if len(t.shells) != 0 {
		return
	}

	t.startTimer()
}

// Starts the 8 second timeout terminate timer.
// If it was already ticking, it is restarted.
//
// Use under a write mutex.
func (t *DistroState) startTimer() {
	t.cancelTimer()
	t.terminateTimer = time.AfterFunc(8*time.Second, func() { _ = t.Terminate() })
}

// Cancels the terminate timer.
//
// Use under a write mutex.
func (t *DistroState) cancelTimer() {
	if t.terminateTimer == nil {
		return
	}
	t.terminateTimer.Stop()
	t.terminateTimer = nil
}
