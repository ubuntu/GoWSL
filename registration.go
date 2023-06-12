package gowsl

// This file contains utilities to create, destroy, stop WSL distros,
// as well as utilities to query this status.

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/ubuntu/decorate"
	"github.com/ubuntu/gowsl/internal/backend"
)

// Register is a wrapper around Win32's WslRegisterDistribution.
// It creates a new distro with a copy of the given tarball as
// its filesystem.
func (d *Distro) Register(rootFsPath string) (err error) {
	defer decorate.OnError(&err, "could not register %s from rootfs in %s", d.name, rootFsPath)

	rootFsPath, err = fixPath(rootFsPath)
	if err != nil {
		return err
	}

	r, err := d.isRegistered()
	if err != nil {
		return err
	}
	if r {
		return errors.New("already registered")
	}

	return d.backend.WslRegisterDistribution(d.Name(), rootFsPath)
}

// RegisteredDistros returns a slice of the registered distros.
func RegisteredDistros(ctx context.Context) (distros []Distro, err error) {
	defer decorate.OnError(&err, "could not obtain registered distros")

	names, err := registeredDistros(selectBackend(ctx))
	if err != nil {
		return distros, err
	}
	for name := range names {
		distros = append(distros, NewDistro(ctx, name))
	}
	return distros, nil
}

// RegisteredDistros returns a map of the registered distros and their GUID.
func registeredDistros(backend backend.Backend) (distros map[string]uuid.UUID, err error) {
	r, err := backend.OpenLxssRegistry(".")
	if err != nil {
		return nil, err
	}
	defer r.Close()

	subkeys, err := r.SubkeyNames()
	if err != nil {
		return distros, err
	}

	distros = make(map[string]uuid.UUID, len(subkeys))
	for _, key := range subkeys {
		guid, err := uuid.Parse(key)
		if err != nil {
			continue // Not a WSL distro
		}

		r, err = backend.OpenLxssRegistry(key)
		if err != nil {
			return nil, err
		}
		defer r.Close()

		name, err := r.Field("DistributionName")
		if err != nil {
			return nil, err
		}

		distros[name] = guid
	}

	return distros, nil
}

// IsRegistered returns a boolean indicating whether a distro is registered or not.
func (d Distro) IsRegistered() (registered bool, err error) {
	r, err := d.isRegistered()
	if err != nil {
		return false, fmt.Errorf("%s: %v", d.name, err)
	}
	return r, nil
}

// isRegistered is the internal way of detecting whether a distro is registered or
// not. Use this one internally to avoid repeating error information.
func (d Distro) isRegistered() (registered bool, err error) {
	defer decorate.OnError(&err, "could not determine if distro is registered")
	distros, err := registeredDistros(d.backend)
	if err != nil {
		return false, err
	}

	_, found := distros[d.Name()]
	return found, nil
}

// Unregister is a wrapper around Win32's WslUnregisterDistribution.
// It irreparably destroys a distro and its filesystem.
func (d *Distro) Unregister() (err error) {
	defer decorate.OnError(&err, "could not unregister %q", d.name)

	r, err := d.isRegistered()
	if err != nil {
		return err
	}
	if !r {
		return errors.New("distro is not registered")
	}

	return d.backend.WslUnregisterDistribution(d.Name())
}

// Install installs a new distro from the Windows store.
func Install(ctx context.Context, appxName string) error {
	return selectBackend(ctx).Install(ctx, appxName)
}

// Uninstall removes the distro's associated AppxPackage (if there is one)
// and unregisters the distro.
func (d *Distro) Uninstall(ctx context.Context) (err error) {
	defer decorate.OnError(&err, "Distro %q uninstall", d.name)

	guid, err := d.GUID()
	if err != nil {
		return err
	}

	k, err := d.backend.OpenLxssRegistry(fmt.Sprintf("{%s}", guid))
	if err != nil {
		return err
	}
	defer k.Close()

	packageFamilyName, err := k.Field("PackageFamilyName")
	if errors.Is(err, fs.ErrNotExist) {
		// Distro was imported, so there is no Appx associated
		return d.backend.WslUnregisterDistribution(d.Name())
	}
	if err != nil {
		return err
	}

	if err := d.backend.RemoveAppxFamily(ctx, packageFamilyName); err != nil {
		return err
	}

	return d.backend.WslUnregisterDistribution(d.Name())
}

// fixPath deals with the fact that WslRegisterDistribuion is
// a bit picky with the path format.
func fixPath(relative string) (string, error) {
	abs, err := filepath.Abs(filepath.FromSlash(relative))
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(abs); errors.Is(err, os.ErrNotExist) {
		return "", errors.New("file not found")
	}
	return abs, nil
}
