// Package mock mocks the WSL api, useful for tests as it allows parallelism,
// decoupling, and execution speed.
package mock

import (
	"path/filepath"
)

// Backend implements the Backend interface.
type Backend struct {
	lxssRootKey *RegistryKey // Map from GUID to key
}

// New constructs a new mocked back-end for WSL.
func New() *Backend {
	return &Backend{
		lxssRootKey: &RegistryKey{
			path: lxssPath,
			children: map[string]*RegistryKey{
				"AppxInstallerCache": {
					path: filepath.Join(lxssPath, "AppxInstallerCache"),
				},
			},
			data: map[string]any{
				"DefaultDistribution": "",
			},
		},
	}
}
