package gowsl

// This file contains utilities to create, destroy, stop WSL distros,
// as well as utilities to query this status.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// Register is a wrapper around Win32's WslRegisterDistribution.
// It creates a new distro with a copy of the given tarball as
// its filesystem.
func (d *Distro) Register(rootFsPath string) (e error) {
	defer func() {
		if e != nil {
			e = fmt.Errorf("error registering %q: %v", d.Name(), e)
		}
	}()

	rootFsPath, err := fixPath(rootFsPath)
	if err != nil {
		return err
	}

	r, err := d.IsRegistered()
	if err != nil {
		return err
	}
	if r {
		return errors.New("already registered")
	}

	return wslRegisterDistribution(d.Name(), rootFsPath)
}

// RegisteredDistros returns a slice of the registered distros.
func RegisteredDistros() (distros []Distro, err error) {
	names, err := registeredDistros()
	if err != nil {
		return distros, err
	}
	for name := range names {
		distros = append(distros, NewDistro(name))
	}
	return distros, nil
}

// RegisteredDistros returns a map of the registered distros and their GUID.
func registeredDistros() (distros map[string]uuid.UUID, err error) {
	r, err := openRegistry(lxssPath)
	if err != nil {
		return nil, err
	}
	defer r.close()

	subkeys, err := r.subkeyNames()

	distros = make(map[string]uuid.UUID, len(subkeys))
	for _, key := range subkeys {
		guid, err := uuid.Parse(key)
		if err != nil {
			continue // Not a WSL distro
		}

		r, err = openRegistry(lxssPath, key)
		if err != nil {
			return nil, err
		}
		defer r.close()

		name, err := r.field("DistributionName")
		if err != nil {
			return nil, err
		}

		distros[name] = guid
	}

	return distros, nil
}

// IsRegistered returns a boolean indicating whether a distro is registered or not.
func (d Distro) IsRegistered() (registered bool, e error) {
	defer func() {
		if e != nil {
			e = fmt.Errorf("failed to detect if %q is registered: %v", d.Name(), e)
		}
	}()

	distros, err := RegisteredDistros()
	if err != nil {
		return false, err
	}

	for _, dist := range distros {
		if dist.Name() != d.Name() {
			continue
		}
		return true, nil
	}
	return false, nil
}

// Unregister is a wrapper around Win32's WslUnregisterDistribution.
// It irreparably destroys a distro and its filesystem.
func (d *Distro) Unregister() (e error) {
	defer func() {
		if e != nil {
			e = fmt.Errorf("failed to unregister %q: %v", d.Name(), e)
		}
	}()

	r, err := d.IsRegistered()
	if err != nil {
		return err
	}
	if !r {
		return errors.New("not registered")
	}

	return wslUnregisterDistribution(d.Name())
}

// fixPath deals with the fact that WslRegisterDistribuion is
// a bit picky with the path format.
func fixPath(relative string) (string, error) {
	abs, err := filepath.Abs(filepath.FromSlash(relative))
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(abs); errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("file %q does not exist", abs)
	}
	return abs, nil
}
