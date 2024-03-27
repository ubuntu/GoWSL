// Package backend defines all the actions that a back-end to GoWSL must
// be able to perform in order to run, or otherwise mock WSL.
package backend

import (
	"context"
	"os"

	"github.com/ubuntu/gowsl/internal/flags"
	"github.com/ubuntu/gowsl/internal/state"
)

// RegistryKey mocks a very small subset of behaviours of a Windows Registry key, enough
// for GoWSL to do the limited amount of traversal and reading that it needs.
type RegistryKey interface {
	Close() error
	Field(name string) (string, error)
	SubkeyNames() ([]string, error)
}

// Backend defines what a back-end to GoWSL must be able to do or mock.
type Backend interface {
	// Registry
	OpenLxssRegistry(path string) (RegistryKey, error)

	// Appx management
	RemoveAppxFamily(ctx context.Context, packageFamilyName string) error

	// wsl.exe
	State(distributionName string) (state.State, error)
	Shutdown() error
	Terminate(distroName string) error
	SetAsDefault(distroName string) error
	Install(ctx context.Context, appxName string) error
	Import(ctx context.Context, distributionName, sourcePath, destinationPath string) error

	// Win32
	WslConfigureDistribution(distributionName string, defaultUID uint32, wslDistributionFlags flags.WslFlags) error
	WslGetDistributionConfiguration(distroName string, distributionVersion *uint8, defaultUID *uint32, wslDistributionFlags *flags.WslFlags, defaultEnvironmentVariables *map[string]string) error
	WslLaunch(distroName string, command string, useCWD bool, stdin *os.File, stdout *os.File, stderr *os.File) (*os.Process, error)
	WslLaunchInteractive(distributionName string, command string, useCurrentWorkingDirectory bool) (uint32, error)
	WslRegisterDistribution(distributionName string, tarGzFilename string) error
	WslUnregisterDistribution(distributionName string) error
}
