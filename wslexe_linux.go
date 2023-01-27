package gowsl

// This file mocks utilities to access functionality accessed via wsl.exe

import (
	"errors"
)

func shutdown() error {
	return errors.New("not implemented")
}

func terminate(distroName string) error {
	return errors.New("not implemented")
}

func setAsDefault(distroName string) error {
	return errors.New("not implemented")
}
