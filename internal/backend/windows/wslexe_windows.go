package windows

// This file contains utilities to access functionality accessed via wsl.exe

import (
	"fmt"
	"os/exec"
)

// Shutdown shuts down all distros
//
// It is analogous to
//
//	`wsl.exe --Shutdown
func (Backend) Shutdown() error {
	out, err := exec.Command("wsl.exe", "--shutdown").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error shutting WSL down: %v: %s", err, out)
	}
	return nil
}

// Terminate shuts down a particular distro
//
// It is analogous to
//
//	`wsl.exe --Terminate <distroName>`
func (Backend) Terminate(distroName string) error {
	out, err := exec.Command("wsl.exe", "--terminate", distroName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error terminating distro %q: %v: %s", distroName, err, out)
	}
	return nil
}

// SetAsDefault sets a particular distribution as the default one.
//
// It is analogous to
//
//	`wsl.exe --set-default <distroName>`
func (Backend) SetAsDefault(distroName string) error {
	out, err := exec.Command("wsl.exe", "--set-default", distroName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error setting %q as default: %v, output: %s", distroName, err, out)
	}
	return nil
}
