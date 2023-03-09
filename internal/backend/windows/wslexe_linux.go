package windows

// This file mocks utilities to access functionality accessed via wsl.exe

import (
	"errors"
)

// Shutdown shuts down all distros
// This implementation will always fail on Linux.
func (Backend) Shutdown() error {
	return errors.New("not implemented")
}

// Terminate shuts down a particular distro
// This implementation will always fail on Linux.
func (Backend) Terminate(distroName string) error {
	return errors.New("not implemented")
}

// SetAsDefault sets a particular distribution as the default one.
// This implementation will always fail on Linux.
func (Backend) SetAsDefault(distroName string) error {
	return errors.New("not implemented")
}
