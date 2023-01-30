package gowsl

// This file contains mocks for Win32 API definitions and imports.

import (
	"errors"
	"math"
	"os"

	"github.com/ubuntu/decorate"
)

func queryFileType(f *os.File) (fileType, error) {
	return 0, errors.New("not implemented")
}

func wslConfigureDistribution(distributionName string, defaultUID uint32, wslDistributionFlags wslFlags) (err error) {
	defer decorate.OnError(&err, "WslConfigureDistribution")
	return errors.New("not implemented")
}

func wslGetDistributionConfiguration(distributionName string,
	distributionVersion *uint8,
	defaultUID *uint32,
	wslDistributionFlags *wslFlags,
	defaultEnvironmentVariables *map[string]string) (err error) {
	defer decorate.OnError(&err, "WslGetDistributionConfiguration")
	return errors.New("not implemented")
}

func wslLaunch(distroName string,
	command string,
	useCWD bool,
	stdin *os.File,
	stdout *os.File,
	stderr *os.File) (process *os.Process, err error) {
	defer decorate.OnError(&err, "WslLaunch")
	return nil, errors.New("not implemented")
}

func wslLaunchInteractive(distributionName string, command string, useCurrentWorkingDirectory bool) (exitCode uint32, err error) {
	defer decorate.OnError(&err, "WslLaunchInteractive")
	return math.MaxUint32, errors.New("not implemented")
}

func wslRegisterDistribution(distributionName string, tarGzFilename string) (err error) {
	defer decorate.OnError(&err, "WslRegisterDistribution")
	return errors.New("not implemented")
}

func wslUnregisterDistribution(distributionName string) (err error) {
	defer decorate.OnError(&err, "WslUnregisterDistribution")
	return errors.New("not implemented")
}
