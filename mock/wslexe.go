package mock

import (
	"errors"

	"github.com/ubuntu/gowsl/internal/state"
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

// State returns the state of a particular distro as seen in `wsl.exe -l -v`.
func (Backend) State(distributionName string) (s state.State, err error) {
	return s, errors.New("not implemented")
}
