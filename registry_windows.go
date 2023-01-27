package gowsl

import (
	"errors"
	"fmt"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

// registryKey wraps around a Windows registry key.
// Create it by calling openRegistry. Must be closed after use with registryKey.close.
type registryKey struct {
	key  registry.Key
	path string // For error message purposes
}

const (
	lxssReg  = registry.CURRENT_USER
	lxssPath = `Software\Microsoft\Windows\CurrentVersion\Lxss\`
)

// openRegistry opens a registry key at the chosen path. Multiple arguments will be
// joined to form the path.
func openRegistry(path ...string) (r *registryKey, err error) {
	r = &registryKey{
		path: filepath.Join(path...),
	}
	return r, nil
}

// close releases the key.
func (r registryKey) close() (err error) {
	defer decorate.OnError(&err, "registry: could not close HKEY_CURRENT_USER\\%s", r.path)
	return r.key.Close()
}

// field obtains the value of a field. The value must be a string.
func (r registryKey) field(name string) (value string, err error) {
	value, _, err = r.key.GetStringValue(name)
	if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
		return value, errors.New("field not found")
	}
	if err != nil {
		return value, err
	}
	return value, nil
}

// subkeyNames returns a slice containing the names of the current key's children.
func (r registryKey) subkeyNames() (subkeys []string, err error) {
	defer decorate.OnError(&err, "registry: could not access subkeys under HKEY_CURRENT_USER\\%s", r.path)

	keyInfo, err := r.key.Stat()
	if err != nil {
		return nil, fmt.Errorf("could not stat parent registry key: %v", err)
	}
	return r.key.ReadSubKeyNames(int(keyInfo.SubKeyCount))
}
