package windows

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"syscall"

	"github.com/ubuntu/decorate"
	"github.com/ubuntu/gowsl/internal/backend"
	"golang.org/x/sys/windows/registry"
)

// RegistryKey wraps around a Windows registry key.
// Create it by calling OpenLxssRegistry. Must be closed after use with RegistryKey.close.
type RegistryKey struct {
	key  registry.Key
	path string // For error message purposes
}

// OpenLxssRegistry opens a registry key at the chosen path.
func (Backend) OpenLxssRegistry(path string) (r backend.RegistryKey, err error) {
	const lxssPath = `Software\Microsoft\Windows\CurrentVersion\Lxss\` // Path to the Lxss registry key. All WSL info is under this path

	p := filepath.Join(lxssPath, path)
	defer decorate.OnError(&err, "registry: could not open HKEY_CURRENT_USER\\%s", p)

	k, err := registry.OpenKey(registry.CURRENT_USER, p, registry.READ)
	if err != nil {
		return nil, err
	}

	return &RegistryKey{
		path: p,
		key:  k,
	}, nil
}

// Close releases the key.
func (r *RegistryKey) Close() (err error) {
	defer decorate.OnError(&err, "registry: could not close HKEY_CURRENT_USER\\%s", r.path)
	return r.key.Close()
}

// Field obtains the value of a Field. The value must be a string.
func (r *RegistryKey) Field(name string) (value string, err error) {
	defer decorate.OnError(&err, "registry: could not access string field %s in HKEY_CURRENT_USER\\%s", name, r.path)

	value, _, err = r.key.GetStringValue(name)
	if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
		return value, fs.ErrNotExist
	}
	if err != nil {
		return value, err
	}
	return value, nil
}

// SubkeyNames returns a slice containing the names of the current key's children.
func (r *RegistryKey) SubkeyNames() (subkeys []string, err error) {
	defer decorate.OnError(&err, "registry: could not access subkeys under HKEY_CURRENT_USER\\%s", r.path)

	keyInfo, err := r.key.Stat()
	if err != nil {
		return nil, fmt.Errorf("could not stat parent registry key: %v", err)
	}
	return r.key.ReadSubKeyNames(int(keyInfo.SubKeyCount))
}
