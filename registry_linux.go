package gowsl

import (
	"errors"
)

// registryKey wraps around a Windows registry key.
// Create it by calling openRegistry. Must be closed after use with registryKey.close.
// This implementation is a mock used for testing.
type registryKey struct {
	path string
}

const (
	lxssPath = `Software/Microsoft/Windows/CurrentVersion/Lxss/`
)

// openRegistry opens a registry key at the chosen path. Multiple arguments will be
// joined to form the path.
// This implementation is a mock used for testing.
func openRegistry(path ...string) (r *registryKey, err error) {
	return nil, errors.New("Not implemented")
}

// close releases the key.
// This implementation is a mock used for testing.
func (r registryKey) close() (err error) {
	return errors.New("not implemented")
}

// field obtains the value of a field. The value must be a string.
// This implementation is a mock used for testing.
func (r registryKey) field(name string) (value string, err error) {
	return "", errors.New("not implemented")
}

// subkeyNames returns a slice containing the names of the current key's children.
// This implementation is a mock used for testing.
func (r registryKey) subkeyNames() (subkeys []string, err error) {
	return nil, errors.New("not implemented")
}
