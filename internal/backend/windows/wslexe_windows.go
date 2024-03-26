package windows

// This file contains utilities to access functionality accessed via wsl.exe

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
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
	_, err := wslExe(context.Background(), "--shutdown")
	if err != nil {
		return fmt.Errorf("could not shut WSL down: %w", err)
	}
	return nil
}

// Terminate shuts down a particular distro
//
// It is analogous to
//
//	`wsl.exe --Terminate <distroName>`
func (Backend) Terminate(distroName string) error {
	_, err := wslExe(context.Background(), "--terminate", distroName)
	if err != nil {
		return fmt.Errorf("could not terminate distro %q: %w", distroName, err)
	}
	return nil
}

// SetAsDefault sets a particular distribution as the default one.
//
// It is analogous to
//
//	`wsl.exe --set-default <distroName>`
func (Backend) SetAsDefault(distroName string) error {
	_, err := wslExe(context.Background(), "--set-default", distroName)
	if err != nil {
		return fmt.Errorf("could not set %q as default: %w", distroName, err)
	}
	return nil
}

// State returns the state of a particular distro as seen in `wsl.exe -l -v`.
func (Backend) State(distributionName string) (s state.State, err error) {
	out, err := wslExe(context.Background(), "--list", "--all", "--verbose")
	if err != nil {
		return s, fmt.Errorf("could not get states of distros: %w", err)
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

// Install installs a new distro from the Windows store.
func (b Backend) Install(ctx context.Context, appxName string) error {
	// Using --no-launch to avoid registration and (non-interactive) user creation.
	_, err := wslExe(ctx, "--install", appxName, "--no-launch")
	if err != nil {
		return fmt.Errorf("could not install %q: %w", appxName, err)
	}
	return nil
}

func (b Backend) Import(ctx context.Context, distributionName, sourcePath, destinationPath string) error {
	_, err := wslExe(ctx, "--import", distributionName, destinationPath, sourcePath)
	if err != nil {
		return fmt.Errorf("could not install %s: %v", distributionName, err)
	}

	return nil
}

// wslExe is a helper function to run wsl.exe with the given arguments.
// It returns the stdout, or an error containing both stdout and stderr.
func wslExe(ctx context.Context, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, "wsl.exe", args...)

	// Avoid output encoding issues (WSL uses UTF-16 by default)
	cmd.Env = append(os.Environ(), "WSL_UTF8=1")

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return stdout.Bytes(), nil
	}

	if strings.Contains(stdout.String(), "Wsl/Service/WSL_E_DISTRO_NOT_FOUND") {
		return nil, ErrNotExist
	}

	if strings.Contains(stderr.String(), "Wsl/Service/WSL_E_DISTRO_NOT_FOUND") {
		return nil, ErrNotExist
	}

	return nil, fmt.Errorf("%v. Stdout: %s. Stderr: %s", err, stdout.Bytes(), stderr.Bytes())
}
