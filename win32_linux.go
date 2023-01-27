package gowsl

// This file contains mocks for Win32 API definitions and imports.

import (
	"errors"
	"math"
	"os"
)

func queryFileType(f *os.File) (fileType, error) {
	return 0, errors.New("not implemented")
}

func wslConfigureDistribution(distributionName string, defaultUID uint32, wslDistributionFlags wslFlags) error {
	return errors.New("not implemented")
}

func wslGetDistributionConfiguration(distributionName string,
	distributionVersion *uint8,
	defaultUID *uint32,
	wslDistributionFlags *wslFlags,
	defaultEnvironmentVariables *map[string]string) error {
	return errors.New("not implemented")
}

func wslLaunch(distroName string,
	command string,
	useCWD bool,
	stdin *os.File,
	stdout *os.File,
	stderr *os.File) (process *os.Process, err error) {
	return nil, errors.New("not implemented")
}

func wslLaunchInteractive(distributionName string, command string, useCurrentWorkingDirectory bool) (uint32, error) {
	return math.MaxUint32, errors.New("not implemented")
}

func wslRegisterDistribution(distributionName string, tarGzFilename string) error {
	return errors.New("not implemented")
}

func wslUnregisterDistribution(distributionName string) error {
	return errors.New("not implemented")
}
