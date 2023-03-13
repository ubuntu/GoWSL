package windows

// This file contains mocks for Win32 API definitions and imports.

import (
	"errors"
	"math"
	"os"

	"github.com/ubuntu/decorate"
	"github.com/ubuntu/gowsl/internal/flags"
)

// WslConfigureDistribution is a wrapper around the WslConfigureDistribution
// function in the wslApi.dll Win32 library.
// This implementation will always fail on Linux.
func (Backend) WslConfigureDistribution(distributionName string, defaultUID uint32, wslDistributionFlags flags.WslFlags) (err error) {
	defer decorate.OnError(&err, "WslConfigureDistribution")
	return errors.New("not implemented")
}

// WslGetDistributionConfiguration is a wrapper around the WslGetDistributionConfiguration
// function in the wslApi.dll Win32 library.
// This implementation will always fail on Linux.
func (Backend) WslGetDistributionConfiguration(distributionName string,
	distributionVersion *uint8,
	defaultUID *uint32,
	wslDistributionFlags *flags.WslFlags,
	defaultEnvironmentVariables *map[string]string) (err error) {
	defer decorate.OnError(&err, "WslGetDistributionConfiguration")
	return errors.New("not implemented")
}

// WslLaunch is a wrapper around the WslLaunch
// function in the wslApi.dll Win32 library.
// This implementation will always fail on Linux.
func (Backend) WslLaunch(distroName string,
	command string,
	useCWD bool,
	stdin *os.File,
	stdout *os.File,
	stderr *os.File) (process *os.Process, err error) {
	defer decorate.OnError(&err, "WslLaunch")
	return nil, errors.New("not implemented")
}

// WslLaunchInteractive is a wrapper around the WslLaunchInteractive
// function in the wslApi.dll Win32 library.
// This implementation will always fail on Linux.
func (Backend) WslLaunchInteractive(distributionName string, command string, useCurrentWorkingDirectory bool) (exitCode uint32, err error) {
	defer decorate.OnError(&err, "WslLaunchInteractive")
	return math.MaxUint32, errors.New("not implemented")
}

// WslRegisterDistribution is a wrapper around the WslRegisterDistribution
// function in the wslApi.dll Win32 library.
// This implementation will always fail on Linux.
func (Backend) WslRegisterDistribution(distributionName string, tarGzFilename string) (err error) {
	defer decorate.OnError(&err, "WslRegisterDistribution")
	return errors.New("not implemented")
}

// WslUnregisterDistribution is a wrapper around the WslUnregisterDistribution
// function in the wslApi.dll Win32 library.
// This implementation will always fail on Linux.
func (Backend) WslUnregisterDistribution(distributionName string) (err error) {
	defer decorate.OnError(&err, "WslUnregisterDistribution")
	return errors.New("not implemented")
}
