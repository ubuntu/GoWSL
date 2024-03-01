package gowsl

// This file contains utilities to interact with a Distro and its configuration

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/ubuntu/decorate"
	"github.com/ubuntu/gowsl/internal/backend"
	"github.com/ubuntu/gowsl/internal/flags"
	"github.com/ubuntu/gowsl/internal/state"
)

// Distro is an abstraction around a WSL distro.
type Distro struct {
	backend backend.Backend
	name    string
}

// State is the state of a particular distro as seen in `wsl.exe -l -v`.
type State = state.State

// The states here reported are the ones obtained via `wsl.exe -l -v`,
// with the addition of NonRegistered.
const (
	Stopped       = state.Stopped
	Running       = state.Running
	Installing    = state.Installing
	Uninstalling  = state.Uninstalling
	NonRegistered = state.NotRegistered
)

// NewDistro declares a new distribution, but does not register it nor
// check if it exists.
func NewDistro(ctx context.Context, name string) Distro {
	return Distro{
		backend: selectBackend(ctx),
		name:    name,
	}
}

// Name is a getter for the DistroName as shown in "wsl.exe --list".
func (d Distro) Name() string {
	return d.name
}

// GUID returns the Global Unique IDentifier for the distro.
func (d *Distro) GUID() (id uuid.UUID, err error) {
	defer decorate.OnError(&err, "could not obtain GUID of %s", d.name)

	distros, err := registeredDistros(d.backend)
	if err != nil {
		return id, err
	}
	id, ok := distros[d.Name()]
	if !ok {
		return id, errors.New("distro is not registered")
	}
	return id, nil
}

// State returns the current state of the distro.
func (d *Distro) State() (s State, err error) {
	defer decorate.OnError(&err, "could not get distro %q's state", d.Name())

	registered, err := d.isRegistered()
	if err != nil {
		return s, err
	}
	if !registered {
		return state.NotRegistered, nil
	}

	return d.backend.State(d.Name())
}

// Terminate powers off the distro.
// Equivalent to:
//
//	wsl --terminate <distro>
func (d *Distro) Terminate() error {
	return d.backend.Terminate(d.Name())
}

// Shutdown powers off all of WSL, including all other distros.
// Equivalent to:
//
//	wsl --shutdown
func Shutdown(ctx context.Context) error {
	return selectBackend(ctx).Shutdown()
}

// SetAsDefault sets a particular distribution as the default one.
// Equivalent to:
//
//	wsl --set-default <distro>
func (d *Distro) SetAsDefault() error {
	return d.backend.SetAsDefault(d.Name())
}

// DefaultDistro gets the current default distribution.
func DefaultDistro(ctx context.Context) (d Distro, ok bool, err error) {
	defer decorate.OnError(&err, "could not obtain the default distro")
	backend := selectBackend(ctx)

	// First, we find out the GUID of the default distro
	r, err := backend.OpenLxssRegistry(".")
	if err != nil {
		return d, false, err
	}
	defer r.Close()

	guid, err := r.Field("DefaultDistribution")
	if err != nil {
		return d, false, err
	}

	if guid == "" {
		return d, false, nil
	}

	// Safety check: we ensure the gui is valid
	if _, err = uuid.Parse(guid); err != nil {
		return d, false, fmt.Errorf("registry returned invalid GUID: %s", guid)
	}

	// Last, we find out the name of the distro
	r, err = backend.OpenLxssRegistry(guid)
	if err != nil {
		return d, false, err
	}
	defer r.Close()

	name, err := r.Field("DistributionName")
	if err != nil {
		return d, false, err
	}

	return NewDistro(ctx, name), true, err
}

// Configuration is the configuration of the distro.
type Configuration struct {
	Version    uint8  // Type of filesystem used (lxfs vs. wslfs, relevant only to WSL1)
	DefaultUID uint32 // User ID of default user
	flags.Unpacked
	DefaultEnvironmentVariables map[string]string // Environment variables passed to the distro by default
}

// DefaultUID sets the user to the one specified.
func (d *Distro) DefaultUID(uid uint32) (err error) {
	defer decorate.OnError(&err, "could not modify flag DEFAULT_UID for %s", d.name)

	conf, err := d.GetConfiguration()
	if err != nil {
		return err
	}
	conf.DefaultUID = uid
	return d.configure(conf)
}

// InteropEnabled sets the ENABLE_INTEROP flag to the provided value.
// Enabling allows you to launch Windows executables from WSL.
func (d *Distro) InteropEnabled(value bool) (err error) {
	defer decorate.OnError(&err, "could not modify flag ENABLE_INTEROP for %s", d.name)

	conf, err := d.GetConfiguration()
	if err != nil {
		return err
	}
	conf.InteropEnabled = value
	return d.configure(conf)
}

// PathAppended sets the APPEND_NT_PATH flag to the provided value.
// Enabling it allows WSL to append /mnt/c/... (or wherever your mount
// point is) in front of Windows executables.
func (d *Distro) PathAppended(value bool) (err error) {
	defer decorate.OnError(&err, "could not modify flag APPEND_NT_PATH for %s", d.name)

	conf, err := d.GetConfiguration()
	if err != nil {
		return err
	}
	conf.PathAppended = value
	return d.configure(conf)
}

// DriveMountingEnabled sets the ENABLE_DRIVE_MOUNTING flag to the provided value.
// Enabling it mounts the windows filesystem into WSL's.
func (d *Distro) DriveMountingEnabled(value bool) (err error) {
	defer decorate.OnError(&err, "could not modify flag ENABLE_DRIVE_MOUNTING for %s", d.name)

	conf, err := d.GetConfiguration()
	if err != nil {
		return err
	}
	conf.DriveMountingEnabled = value
	return d.configure(conf)
}

// GetConfiguration is a wrapper around Win32's WslGetDistributionConfiguration.
// It returns a configuration object with information about the distro.
func (d Distro) GetConfiguration() (c Configuration, err error) {
	defer decorate.OnError(&err, "could not access configuration for %s", d.name)

	var conf Configuration
	var f flags.WslFlags

	err = d.backend.WslGetDistributionConfiguration(
		d.Name(),
		&conf.Version,
		&conf.DefaultUID,
		&f,
		&conf.DefaultEnvironmentVariables,
	)

	if err != nil {
		return conf, err
	}
	conf.Unpacked = flags.Unpack(f)

	return conf, nil
}

// String deserializes a distro its GUID and its configuration as a yaml string.
// If there is an error, it is printed as part of the yaml.
func (d Distro) String() string {
	guid, err := d.GUID()
	if err != nil {
		return fmt.Sprintf("WSL distro %q (not registered)", d.Name())
	}
	return fmt.Sprintf("WSL distro %q (%s)", d.Name(), guid)
}

// configure is a wrapper around Win32's WslConfigureDistribution.
// Note that only the following config is mutable:
//   - DefaultUID
//   - InteropEnabled
//   - PathAppended
//   - DriveMountingEnabled
func (d *Distro) configure(config Configuration) error {
	flags, err := config.Pack()
	if err != nil {
		return err
	}

	return d.backend.WslConfigureDistribution(d.Name(), config.DefaultUID, flags)
}
