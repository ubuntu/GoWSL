package gowsl

import (
	"errors"
	"path/filepath"

	"github.com/ubuntu/decorate"
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
	r = &registryKey{
		path: filepath.Join(path...),
	}
	defer decorate.OnError(&err, "registry: could not open HKEY_CURRENT_USER/%s", r.path)
	return nil, errors.New("Not implemented")
}

// close releases the key.
// This implementation is a mock used for testing.
func (r registryKey) close() (err error) {
	defer decorate.OnError(&err, "registry: could not close HKEY_CURRENT_USER/%s", r.path)
	return errors.New("not implemented")
}

// field obtains the value of a field. The value must be a string.
// This implementation is a mock used for testing.
func (r registryKey) field(name string) (value string, err error) {
	defer decorate.OnError(&err, "registry: could not access field %s in HKEY_CURRENT_USER/%s", name, r.path)
	return "", errors.New("not implemented")
}

// subkeyNames returns a slice containing the names of the current key's children.
// This implementation is a mock used for testing.
func (r registryKey) subkeyNames() (subkeys []string, err error) {
	defer decorate.OnError(&err, "registry: could not access subkeys under HKEY_CURRENT_USER/%s", r.path)
	return nil, errors.New("not implemented")
}
