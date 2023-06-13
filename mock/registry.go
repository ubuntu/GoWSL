package mock

import (
	"errors"
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/ubuntu/decorate"
	"github.com/ubuntu/gowsl/internal/backend"
	"github.com/ubuntu/gowsl/mock/internal/distrostate"
)

// RegistryKey wraps around a Windows registry key.
// Create it by calling openRegistry. Must be closed after use with RegistryKey.close.
// This implementation is a mock used for testing.
type RegistryKey struct {
	path string

	children map[string]*RegistryKey
	Data     map[string]any

	state *distrostate.DistroState

	mu sync.RWMutex
}

const (
	lxssPath = `Software/Microsoft/Windows/CurrentVersion/Lxss/`
)

// OpenLxssRegistry opens a registry key at the chosen path subpath of the Lxss key.
//
// This implementation is a mock used for testing.
func (b Backend) OpenLxssRegistry(path string) (r backend.RegistryKey, err error) {
	defer decorate.OnError(&err, "registry: could not open %s", filepath.Join("HKEY_CURRENT_USER", lxssPath, path))

	if b.OpenLxssKeyError {
		return nil, Error{}
	}

	b.lxssRootKey.mu.RLock()
	if path == "." {
		// We "leak" the locked mutex. The user is in charge of releasing it with .Close()
		return b.lxssRootKey, nil
	}

	key, ok := b.lxssRootKey.children[path]
	b.lxssRootKey.mu.RUnlock()
	if !ok {
		return nil, fs.ErrNotExist
	}

	key.mu.RLock()

	return key, nil
}

// Close releases the key.
// This implementation is a mock used for testing.
func (r *RegistryKey) Close() (err error) {
	r.mu.RUnlock()

	return nil
}

// Field obtains the value of a Field. The value must be a string.
// This implementation is a mock used for testing.
func (r *RegistryKey) Field(name string) (value string, err error) {
	defer decorate.OnError(&err, "registry: could not access field %q in %s", name, r.path)

	v, ok := r.Data[name]
	if !ok {
		return "", fs.ErrNotExist
	}

	s, ok := v.(string)
	if !ok {
		return "", errors.New("field is not string")
	}

	return s, nil
}

// SubkeyNames returns a slice containing the names of the current key's children.
// This implementation is a mock used for testing.
func (r *RegistryKey) SubkeyNames() (subkeys []string, err error) {
	defer decorate.OnError(&err, "registry: could not access subkeys under %s", r.path)

	for key := range r.children {
		subkeys = append(subkeys, key)
	}

	return subkeys, nil
}
