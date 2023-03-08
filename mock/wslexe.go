package mock

import (
	"errors"
)

// Shutdown mocks the behaviour of shutting down WSL.
func (Backend) Shutdown() error {
	return errors.New("not implemented")
}

// Terminate mocks the behaviour of shutting down one WSL distro.
func (Backend) Terminate(distroName string) error {
	return errors.New("not implemented")
}

// SetAsDefault mocks the behaviour of setting one distro as default.
func (Backend) SetAsDefault(distroName string) error {
	return errors.New("not implemented")
}
