package gowsl

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"

	"github.com/0xrawsec/golang-utils/log"
	"golang.org/x/sys/windows/registry"
)

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

// registeredDistros returns a slice of the registered distros.
//
// It is analogous to
//
//	`wsl.exe --list`
func registeredDistros() (distros []Distro, err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf("failed to obtain list of registered distros: %v", err)
	}()

	lxssKey, err := registry.OpenKey(lxssRegistry, lxssPath, registry.READ)
	if err != nil {
		return nil, fmt.Errorf("failed to open lxss registry: %v", err)
	}
	defer lxssKey.Close()

	lxssData, err := lxssKey.Stat()
	if err != nil {
		return []Distro{}, fmt.Errorf("failed to stat lxss registry key: %v", err)
	}

	subkeys, err := lxssKey.ReadSubKeyNames(int(lxssData.SubKeyCount))
	if err != nil {
		return []Distro{}, fmt.Errorf("failed to read lxss registry subkeys: %v", err)
	}

	type distroErr struct {
		distro Distro
		err    error
	}
	ch := make(chan distroErr)

	wg := sync.WaitGroup{}

	// Typical Microsoft {8-4-4-4-12} hex code
	// example: {ee8aef7a-846f-4561-a028-79504ce65cd3}
	distroRegex := regexp.MustCompile(`^\{[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}\}$`)
	for _, skName := range subkeys {
		if !distroRegex.MatchString(skName) {
			continue // Not a WSL distro
		}

		skName := skName
		wg.Add(1)
		go func() {
			defer wg.Done()
			name, err := readRegistryDistributionName(skName)

			if err != nil {
				ch <- distroErr{err: fmt.Errorf("failed to parse registry entry %s: %v", skName, err)}
				return
			}
			ch <- distroErr{distro: Distro{Name: name}}
		}()
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	// Collecting results
	for d := range ch {
		if d.err != nil {
			log.Warnf("%v", d.err)
			continue
		}
		distros = append(distros, d.distro)
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
