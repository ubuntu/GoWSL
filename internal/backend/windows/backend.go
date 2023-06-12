// Package windows contains the production backend. It is the
// one used in production code, and makes real syscalls and
// accesses to the registry.
//
// All functions will return an error when ran on Linux.
package windows

import (
	"context"
	"fmt"
	"os/exec"
)

// Backend implements the Backend interface.
type Backend struct{}

// RemoveAppxFamily uninstalls the Appx under the provided family name.
func (Backend) RemoveAppxFamily(ctx context.Context, packageFamilyName string) error {
	cmd := exec.CommandContext(ctx,
		"powershell.exe",
		"-NonInteractive",
		"-NoProfile",
		"-NoLogo",
		"-Command",
		`Get-AppxPackage | Where-Object -Property PackageFamilyName -eq "${env:PackageFamilyName}" | Remove-AppPackage`,
	)
	cmd.Env = append(cmd.Env, fmt.Sprintf("PackageFamilyName=%q", packageFamilyName))

	if out, err := cmd.Output(); err != nil {
		return fmt.Errorf("could not uninstall %q: %v. %s", packageFamilyName, err, out)
	}

	return nil
}
