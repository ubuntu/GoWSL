package windows

// This file contains utilities to access functionality accessed via wsl.exe

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ubuntu/gowsl/internal/state"
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

// State returns the state of a particular distro as seen in `wsl.exe -l -v`.
func (Backend) State(distributionName string) (s state.State, err error) {
	cmd := exec.Command("wsl.exe", "--list", "--all", "--verbose")
	cmd.Env = append(cmd.Env, "WSL_UTF8=1")

	out, err := cmd.Output()
	if err != nil {
		return s, err
	}

	/*
		Sample output:
		   NAME           STATE           VERSION
		 * Ubuntu         Stopped         2
		   Ubuntu-Preview Running         2
	*/

	sc := bufio.NewScanner(bytes.NewReader(out))
	var headerSkipped bool
	for sc.Scan() {
		if !headerSkipped {
			headerSkipped = true
			continue
		}

		data := strings.Fields(sc.Text())
		if len(data) == 4 {
			// default distro, ignoring leading asterisk
			data = data[1:]
		}

		if data[0] == distributionName {
			return state.NewFromString(data[1])
		}
	}

	return state.NotRegistered, nil
}
