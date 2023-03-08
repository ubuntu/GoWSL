package windows

import (
	"errors"
	"path/filepath"

	"github.com/ubuntu/decorate"
	"github.com/ubuntu/gowsl/internal/backend"
)

// RegistryKey wraps around a Windows registry key.
// Create it by calling OpenLxssRegistry. Must be closed after use with RegistryKey.close.
// This implementation will always fail on Linux.
type RegistryKey struct {
	path string
}

const (
	lxssPath = `Software/Microsoft/Windows/CurrentVersion/Lxss/`
)

// OpenLxssRegistry opens a registry key at the chosen path.
// This implementation will always fail on Linux.
func (Backend) OpenLxssRegistry(path string) (r backend.RegistryKey, err error) {
	p := filepath.Join(lxssPath, path)
	r = RegistryKey{
		path: p,
	}
	defer decorate.OnError(&err, "registry: could not open HKEY_CURRENT_USER/%s", p)
	return nil, errors.New("Not implemented")
}

// Close releases the key.
// This implementation will always fail on Linux.
func (r RegistryKey) Close() (err error) {
	defer decorate.OnError(&err, "registry: could not close HKEY_CURRENT_USER/%s", r.path)
	return errors.New("not implemented")
}

// Field obtains the value of a Field. The value must be a string.
// This implementation will always fail on Linux.
func (r RegistryKey) Field(name string) (value string, err error) {
	defer decorate.OnError(&err, "registry: could not access field %s in HKEY_CURRENT_USER/%s", name, r.path)
	return "", errors.New("not implemented")
}

// SubkeyNames returns a slice containing the names of the current key's children.
// This implementation will always fail on Linux.
func (r RegistryKey) SubkeyNames() (subkeys []string, err error) {
	defer decorate.OnError(&err, "registry: could not access subkeys under HKEY_CURRENT_USER/%s", r.path)
	return nil, errors.New("not implemented")
}
