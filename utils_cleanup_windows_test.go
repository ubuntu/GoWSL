//go:build !gowslmock

package gowsl_test

import (
	"strings"
	"testing"

	wsl "github.com/ubuntu/gowsl"
	"golang.org/x/sys/windows/registry"
)

// cleanupRegistry attempts to prevent error "0x8000000d An illegal state change was requested" that may arise when registering a distro
// on top of a previously failed registration or unregistration that may have left some leftovers inside the registry.
// This is an educated guess based on this comment: https://github.com/microsoft/WSL/issues/4084#issuecomment-774731220
func cleanupRegistry(t *testing.T, d wsl.Distro) error {
	t.Helper()

	const lxssPath = `Software\Microsoft\Windows\CurrentVersion\Lxss\` // Path to the Lxss registry key. All WSL info is under this path

	k, err := registry.OpenKey(registry.CURRENT_USER, lxssPath, registry.ALL_ACCESS)
	if err != nil {
		return err
	}

	subkeys, err := k.ReadSubKeyNames(0)
	if len(subkeys) == 0 {
		return err
	}
	for _, guid := range subkeys {
		subKey, err := registry.OpenKey(k, guid, registry.ALL_ACCESS)
		if err != nil {
			return err
		}

		name, _, err := subKey.GetStringValue("DistributionName")
		if err != nil {
			return err
		}

		if !strings.EqualFold(name, d.Name()) {
			continue
		}

		if err = registry.DeleteKey(k, guid); err != nil {
			return err
		}
	}

	return nil
}
