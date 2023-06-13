// Package mock mocks the WSL api, useful for tests as it allows parallelism,
// decoupling, and execution speed.
package mock

import (
	"path/filepath"
)

// Backend implements the Backend interface.
type Backend struct {
	lxssRootKey *RegistryKey // Registry mock

	// Error injectors. These all have the form of:
	//
	// NameOfTheFunctionError
	//
	// Their effect is to make the relevant function return an error of type mock.Error
	// instantly upon being called.
	WslConfigureDistributionError        bool
	WslGetDistributionConfigurationError bool
	WslLaunchError                       bool
	WslLaunchInteractiveError            bool
	WslRegisterDistributionError         bool
	WslUnregisterDistributionError       bool
	OpenLxssKeyError                     bool
	ShutdownError                        bool
	TerminateError                       bool
	SetAsDefaultError                    bool
	StateError                           bool
	InstallError                         bool
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

// ResetErrors sets all the error flags to false.
func (b *Backend) ResetErrors() {
	b.WslConfigureDistributionError = false
	b.WslGetDistributionConfigurationError = false
	b.WslLaunchError = false
	b.WslLaunchInteractiveError = false
	b.WslRegisterDistributionError = false
	b.WslUnregisterDistributionError = false
	b.OpenLxssKeyError = false
	b.ShutdownError = false
	b.TerminateError = false
	b.SetAsDefaultError = false
	b.StateError = false
}

// Error is an error triggered by the mock, and not a real problem.
type Error struct{}

func (err Error) Error() string {
	return "error triggered by mock"
}
