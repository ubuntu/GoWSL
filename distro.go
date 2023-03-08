// Package gowsl wraps around the wslApi.dll (and sometimes wsl.exe) for
// safe and idiomatic use within Go projects.
//
// This package is not thread safe.
package gowsl

// This file contains utilities to interact with a Distro and its configuration

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/ubuntu/decorate"
	"github.com/ubuntu/gowsl/internal/backend"
	"github.com/ubuntu/gowsl/internal/flags"
)

// Distro is an abstraction around a WSL distro.
type Distro struct {
	backend backend.Backend
	name    string
}

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
func DefaultDistro(ctx context.Context) (d Distro, err error) {
	defer decorate.OnError(&err, "could not obtain the default distro")
	backend := selectBackend(ctx)

	// First, we find out the GUID of the default distro
	r, err := backend.OpenLxssRegistry(".")
	if err != nil {
		return d, err
	}
	defer r.Close()

	guid, err := r.Field("DefaultDistribution")
	if err != nil {
		return d, err
	}

	// Safety check: we ensure the gui is valid
	if _, err = uuid.Parse(guid); err != nil {
		return d, fmt.Errorf("registry returned invalid GUID: %s", guid)
	}

	// Last, we find out the name of the distro
	r, err = backend.OpenLxssRegistry(guid)
	if err != nil {
		return d, err
	}
	defer r.Close()

	name, err := r.Field("DistributionName")
	if err != nil {
		return d, err
	}

	return NewDistro(ctx, name), err
}

// Configuration is the configuration of the distro.
type Configuration struct {
	Version                     uint8             // Type of filesystem used (lxfs vs. wslfs, relevant only to WSL1)
	DefaultUID                  uint32            // User ID of default user
	InteropEnabled              bool              // Whether interop with windows is enabled
	PathAppended                bool              // Whether Windows paths are appended
	DriveMountingEnabled        bool              // Whether drive mounting is enabled
	undocumentedWSLVersion      uint8             // Undocumented variable. WSL1 vs. WSL2.
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
	var flags flags.WslFlags

	err = d.backend.WslGetDistributionConfiguration(
		d.Name(),
		&conf.Version,
		&conf.DefaultUID,
		&flags,
		&conf.DefaultEnvironmentVariables,
	)

	if err != nil {
		return conf, err
	}

	conf.unpackFlags(flags)
	return conf, nil
}

// String deserializes a distro its GUID and its configuration as a yaml string.
// If there is an error, it is printed as part of the yaml.
func (d Distro) String() string {
	return fmt.Sprintf("name: %s\n%s\n%s", d.Name(), d.guidToString(), d.configToString())
}

// guidToString shows the GUID as a yaml string.
// It exists to simplify the implementation of (Distro).String
// If it errors out, the message is returned as the value in the yaml.
func (d Distro) guidToString() string {
	registered, err := d.isRegistered()
	if err != nil {
		return fmt.Sprintf("guid: |\n  %v", err)
	}
	if !registered {
		return "guid: distro is not registered"
	}

	id, err := d.GUID()
	if err != nil {
		return fmt.Sprintf("guid: |\n  %v", err)
	}
	return fmt.Sprintf("guid: '%v'", id)
}

// configToString shows the configuration as a yaml string.
// It exists to simplify the implementation of (Distro).String
// If it errors out, the message is returned as the value in the yaml.
func (d Distro) configToString() string {
	c, err := d.GetConfiguration()
	if err != nil {
		return fmt.Sprintf("configuration: |\n  %v\n", err)
	}

	// Get sorted list of environment variables
	envKeys := []string{}
	for k := range c.DefaultEnvironmentVariables {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)

	fmtEnvs := "\n"
	for _, k := range envKeys {
		fmtEnvs = fmt.Sprintf("%s    - %s: %s\n", fmtEnvs, k, c.DefaultEnvironmentVariables[k])
	}

	// Generate the string
	return fmt.Sprintf(`configuration:
  - Version: %d
  - DefaultUID: %d
  - InteropEnabled: %t
  - PathAppended: %t
  - DriveMountingEnabled: %t
  - undocumentedWSLVersion: %d
  - DefaultEnvironmentVariables:%s`, c.Version, c.DefaultUID, c.InteropEnabled, c.PathAppended,
		c.DriveMountingEnabled, c.undocumentedWSLVersion, fmtEnvs)
}

// configure is a wrapper around Win32's WslConfigureDistribution.
// Note that only the following config is mutable:
//   - DefaultUID
//   - InteropEnabled
//   - PathAppended
//   - DriveMountingEnabled
func (d *Distro) configure(config Configuration) error {
	flags, err := config.packFlags()
	if err != nil {
		return err
	}

	return d.backend.WslConfigureDistribution(d.Name(), config.DefaultUID, flags)
}

// unpackFlags examines a winWslFlags object and stores its findings in the Configuration.
func (conf *Configuration) unpackFlags(f flags.WslFlags) {
	conf.InteropEnabled = false
	if flags.ENABLE_INTEROP != 0 {
		conf.InteropEnabled = true
	}

	conf.PathAppended = false
	if f&flags.APPEND_NT_PATH != 0 {
		conf.PathAppended = true
	}

	conf.DriveMountingEnabled = false
	if f&flags.ENABLE_DRIVE_MOUNTING != 0 {
		conf.DriveMountingEnabled = true
	}

	conf.undocumentedWSLVersion = 1
	if f&flags.Undocumented_WSL_VERSION != 0 {
		conf.undocumentedWSLVersion = 2
	}
}

// packFlags generates a winWslFlags object from the Configuration.
func (conf Configuration) packFlags() (flags.WslFlags, error) {
	f := flags.NONE

	if conf.InteropEnabled {
		f = f | flags.ENABLE_INTEROP
	}

	if conf.PathAppended {
		f = f | flags.APPEND_NT_PATH
	}

	if conf.DriveMountingEnabled {
		f = f | flags.ENABLE_DRIVE_MOUNTING
	}

	switch conf.undocumentedWSLVersion {
	case 1:
	case 2:
		f = f | flags.Undocumented_WSL_VERSION
	default:
		return f, fmt.Errorf("unknown WSL version %d", conf.undocumentedWSLVersion)
	}

	return f, nil
}
