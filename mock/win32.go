package mock

// This file contains mocks for Win32 API definitions and imports.

import (
	"errors"
	"math"
	"os"

	"github.com/ubuntu/decorate"
	"github.com/ubuntu/gowsl/internal/flags"
)

// IsPipe mocks checking if a file's descriptor is a pipe vs. any other type of object.
//
// TODO: Not implemented.
func (Backend) IsPipe(f *os.File) (bool, error) {
	return false, errors.New("not implemented")
}

// WslConfigureDistribution mocks the WslConfigureDistribution call to the Win32 API.
//
// TODO: Not implemented.
func (Backend) WslConfigureDistribution(distributionName string, defaultUID uint32, wslDistributionFlags flags.WslFlags) (err error) {
	defer decorate.OnError(&err, "WslConfigureDistribution")
	return errors.New("not implemented")
}

// WslGetDistributionConfiguration mocks the WslGetDistributionConfiguration call to the Win32 API.
//
// TODO: Not implemented.
func (Backend) WslGetDistributionConfiguration(distributionName string,
	distributionVersion *uint8,
	defaultUID *uint32,
	wslDistributionFlags *flags.WslFlags,
	defaultEnvironmentVariables *map[string]string) (err error) {
	defer decorate.OnError(&err, "WslGetDistributionConfiguration")
	return errors.New("not implemented")
}

// WslLaunch mocks the WslLaunch call to the Win32 API.
//
// TODO: Not implemented.
func (Backend) WslLaunch(distroName string,
	command string,
	useCWD bool,
	stdin *os.File,
	stdout *os.File,
	stderr *os.File) (process *os.Process, err error) {
	defer decorate.OnError(&err, "WslLaunch")
	return nil, errors.New("not implemented")
}

// WslLaunchInteractive mocks the WslLaunchInteractive call to the Win32 API.
//
// TODO: Not implemented.
func (Backend) WslLaunchInteractive(distributionName string, command string, useCurrentWorkingDirectory bool) (exitCode uint32, err error) {
	defer decorate.OnError(&err, "WslLaunchInteractive")
	return math.MaxUint32, errors.New("not implemented")
}

// WslRegisterDistribution mocks the WslRegisterDistribution call to the Win32 API.
//
// TODO: Not implemented.
func (Backend) WslRegisterDistribution(distributionName string, tarGzFilename string) (err error) {
	defer decorate.OnError(&err, "WslRegisterDistribution")
	return errors.New("not implemented")
}

// WslUnregisterDistribution mocks the WslUnregisterDistribution call to the Win32 API.
//
// TODO: Not implemented.
func (Backend) WslUnregisterDistribution(distributionName string) (err error) {
	defer decorate.OnError(&err, "WslUnregisterDistribution")
	return errors.New("not implemented")
}
