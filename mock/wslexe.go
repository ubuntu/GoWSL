package mock

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/ubuntu/gowsl/internal/state"
)

// Shutdown mocks the behaviour of shutting down WSL.
func (backend *Backend) Shutdown() (err error) {
	if backend.ShutdownError {
		return Error{}
	}

	backend.lxssRootKey.mu.RLock()
	defer backend.lxssRootKey.mu.RUnlock()

	for guid, key := range backend.lxssRootKey.children {
		if _, err := uuid.Parse(guid); err != nil {
			// Not distro
			continue
		}

		if e := key.state.Terminate(); e != nil {
			err = errors.Join(err, e)
		}
	}

	return err
}

// Terminate mocks the behaviour of shutting down one WSL distro.
func (backend *Backend) Terminate(distroName string) error {
	if backend.TerminateError {
		return Error{}
	}

	backend.lxssRootKey.mu.RLock()
	defer backend.lxssRootKey.mu.RUnlock()

	guid, key := backend.findDistroKey(distroName)
	if guid == "" {
		return errors.New("Bla bla bla this is localized text, don't assert on it.\nError code: Wsl/Service/WSL_E_DISTRO_NOT_FOUND")
	}

	return key.state.Terminate()
}

// SetAsDefault mocks the behaviour of setting one distro as default.
func (backend *Backend) SetAsDefault(distroName string) error {
	if backend.SetAsDefaultError {
		return Error{}
	}

	if err := validDistroName(distroName); err != nil {
		return err
	}

	backend.lxssRootKey.mu.Lock()
	defer backend.lxssRootKey.mu.Unlock()

	GUID, key := backend.findDistroKey(distroName)
	if key == nil {
		return errors.New("distro not registered")
	}

	backend.lxssRootKey.Data["DefaultDistribution"] = GUID

	return nil
}

// State returns the state of a particular distro as seen in `wsl.exe -l -v`.
func (backend Backend) State(distributionName string) (s state.State, err error) {
	if backend.StateError {
		return state.Error, Error{}
	}

	backend.lxssRootKey.mu.RLock()
	_, key := backend.findDistroKey(distributionName)
	backend.lxssRootKey.mu.RUnlock()

	if key == nil {
		return state.NotRegistered, nil
	}

	if key.state.IsRunning() {
		return state.Running, nil
	}
	return state.Stopped, nil
}

// Install installs a new distro from the Windows store.
func (backend Backend) Install(ctx context.Context, appxName string) (err error) {
	if backend.InstallError {
		return Error{}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if appxName == "" {
		return errors.New(`could not install "": Wsl/InstallDistro/WSL_E_DISTRO_NOT_FOUND`)
	}

	return nil
}
