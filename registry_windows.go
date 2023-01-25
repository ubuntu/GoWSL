package gowsl

import (
	"errors"
	"fmt"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// guid is a type alias for windows.GUID. Note that syscall.GUID is not
// equivalent.
type guid = windows.GUID

// defaultDistro gets the name of the default distribution.
func defaultDistro() (name string, err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf("failed to obtain default distro: %v", err)
	}()

	lxssKey, err := registry.OpenKey(lxssRegistry, lxssPath, registry.READ)
	if err != nil {
		return "", fmt.Errorf("failed to open lxss registry: %v", err)
	}
	defer lxssKey.Close()

	target := "DefaultDistribution"
	distroDir, _, err := lxssKey.GetStringValue(target)
	if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
		return "", errors.New("no default distro")
	}
	if err != nil {
		return "", fmt.Errorf("cannot find %s:%s : %v", lxssPath, target, err)
	}

	return readRegistryDistributionName(distroDir)
}

func distroGUIDs() (distros map[string]guid, err error) {
	lxssKey, err := registry.OpenKey(lxssRegistry, lxssPath, registry.READ)
	if err != nil {
		return nil, fmt.Errorf("failed to open lxss registry: %v", err)
	}
	defer lxssKey.Close()

	lxssData, err := lxssKey.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat lxss registry key: %v", err)
	}

	subkeys, err := lxssKey.ReadSubKeyNames(int(lxssData.SubKeyCount))
	if err != nil {
		return nil, fmt.Errorf("failed to read lxss registry subkeys: %v", err)
	}

	distros = make(map[string]guid, len(subkeys))
	for _, key := range subkeys {
		guid, err := windows.GUIDFromString(key)
		if err != nil {
			continue // Not a WSL distro
		}

		name, err := readRegistryDistributionName(key)
		if err != nil {
			return nil, err
		}

		distros[name] = guid
	}

	return distros, nil
}

// registeredDistros returns a slice of the registered distros.
//
// It is analogous to
//
//	`wsl.exe --list`
func registeredDistros() (distros []Distro, err error) {
	registeredDistros, err := distroGUIDs()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain list of registered distros: %v", err)
	}

	distros = make([]Distro, len(registeredDistros))
	for name := range registeredDistros {
		distros = append(distros, NewDistro(name))
	}

	return distros, nil
}

// readRegistryDistributionName returs the value of DistributionName from a registry path.
//
// An example registry path may be
//
//	`Software\Microsoft\Windows\CurrentVersion\Lxss\{ee8aef7a-846f-4561-a028-79504ce65cd3}`.
//
// Then, the registryDir is
//
//	`{ee8aef7a-846f-4561-a028-79504ce65cd3}`
func readRegistryDistributionName(registryDir string) (string, error) {
	keyPath := filepath.Join(lxssPath, registryDir)

	key, err := registry.OpenKey(lxssRegistry, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("cannot find key %s: %v", keyPath, err)
	}
	defer key.Close()

	target := "DistributionName"
	name, _, err := key.GetStringValue(target)
	if err != nil {
		return "", fmt.Errorf("cannot find %s:%s : %v", keyPath, target, err)
	}
	return name, nil
}
