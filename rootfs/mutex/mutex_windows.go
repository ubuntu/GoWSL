package mutex

import (
	"errors"
	"fmt"
	"syscall"

	"golang.org/x/sys/windows"
)

type Mutex struct {
	handle windows.Handle
	name   string
}

// New creates a new mutex or opens an exiting one
func New(name string) (m Mutex, e error) {
	m = Mutex{
		handle: 0,
		name:   name,
	}

	nameUTF16, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return m, fmt.Errorf("failed to convert %q to UTF16", name)
	}

	m.handle, e = windows.CreateMutex(nil, false, nameUTF16)

	if e == nil {
		return m, nil
	}

	var asErrNo *syscall.Errno
	if errors.As(e, &asErrNo) && *asErrNo != syscall.ERROR_ALREADY_EXISTS {
		return m, fmt.Errorf("failed call to CreateMutex (handle: %d, name: %s): %v", m.handle, m.name, e)
	}

	return m, nil

}

// Lock is a blocking call that waits for a Mutex to be available and gets control of it
func (m *Mutex) Lock() error {
	_, err := windows.WaitForSingleObject(m.handle, windows.INFINITE)
	if err != nil {
		return fmt.Errorf("failed call to WaitForSingleObject on a mutex (handle: %d, name: %s): %v", m.handle, m.name, err)
	}
	return nil
}

// Release releases the mutex
func (m *Mutex) Release() error {
	err := windows.ReleaseMutex(m.handle)
	if err != nil {
		return fmt.Errorf("failed call to ReleaseMutex (handle: %d, name: %s): %v", m.handle, m.name, err)
	}
	return nil
}

// Close closes the handle to the mutex
func (m *Mutex) Close() error {
	err := windows.CloseHandle(m.handle)
	if err != nil {
		return fmt.Errorf("failed call to CloseHandle on a mutex (handle: %d, name: %s): %v", m.handle, m.name, err)
	}
	return nil
}
