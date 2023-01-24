package gowsl

// This file contains utilities to access functionality often accessed via wsl.exe,
// with the advantage (sometimes) of not needing to start a subprocess.

import (
	"fmt"
	"os/exec"
)

// shutdown shuts down all distros
//
// It is analogous to
//
//	`wsl.exe --shutdown
func shutdown() error {
	out, err := exec.Command("wsl.exe", "--shutdown").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error shutting WSL down: %v: %s", err, out)
	}
	return nil
}

// terminate shuts down a particular distro
//
// It is analogous to
//
//	`wsl.exe --terminate <distroName>`
func terminate(distroName string) error {
	out, err := exec.Command("wsl.exe", "--terminate", distroName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error terminating distro %q: %v: %s", distroName, err, out)
	}
	return nil
}

// setAsDefault sets a particular distribution as the default one.
//
// It is analogous to
//
//	`wsl.exe --set-default <distroName>`
func setAsDefault(distroName string) error {
	out, err := exec.Command("wsl.exe", "--set-default", distroName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error setting %q as default: %v, output: %s", distroName, err, out)
	}
	return nil
}

