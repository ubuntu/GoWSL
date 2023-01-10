package wsl

// This file contains utilities to access functionality often accessed via wsl.exe,
// with the advantage (sometimes) of not needing to start a subprocess.

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/0xrawsec/golang-utils/log"
	"golang.org/x/sys/windows/registry"
)

// shutdown shuts down all distros
//
// It is analogous to
//  `wsl.exe --shutdown
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
//  `wsl.exe --terminate <distroName>`
func terminate(distroName string) error {
	out, err := exec.Command("wsl.exe", "--terminate", distroName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error terminating distro %q: %v: %s", distroName, err, out)
	}
	return nil
}

func export(distroName string, where string, vhd bool) error {
	args := []string{"--export", where}
	if vhd {
		args = append(args, "--vhd")
	}
	out, err := exec.Command("wsl.exe", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error exporting %s: %v: %s", distroName, err, out)
	}
	return nil
}

func importCopy(distroName string, newVhd string, rootfs string, asVhd bool, wslVersion int) error {
	args := []string{"--import", distroName, newVhd, rootfs, "--version", fmt.Sprint(wslVersion)}
	if asVhd {
		args = append(args, "--vhd")
	}
	out, err := exec.Command("wsl.exe", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error exporting %s: %v: %s", distroName, err, out)
	}
	return nil
}

func importInPlace(distroName string, vhd string) error {
	args := []string{"--import-in-place", distroName, vhd}
	out, err := exec.Command("wsl.exe", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error exporting %s: %v: %s", distroName, err, out)
	}
	return nil
}

// setAsDefault sets a particular distribution as the default one.
//
// It is analogous to
//  `wsl.exe --set-default <distroName>`
func setAsDefault(distroName string) error {
	out, err := exec.Command("wsl.exe", "--set-default", distroName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error setting %q as default: %v, output: %s", distroName, err, out)
	}
	return nil
}

// defaultDistro gets the name of the default distribution.
// If no distros are installed (hence no default), an empty string is returned.
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

	lxssData, err := lxssKey.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat lxss registry key: %v", err)
	}

	if lxssData.SubKeyCount < 2 {
		// lxss contains subkeys:
		// - AppxInstallerCache
		// - {distro 8-4-4-4-8 code}
		// - {distro 8-4-4-4-8 code}
		// - ...
		// We know there are no distros when there is one or fewer subkeys
		return "", nil
	}

	target := "DefaultDistribution"
	distroDir, _, err := lxssKey.GetStringValue(target)
	if err != nil {
		return "", fmt.Errorf("cannot find %s:%s : %v", lxssPath, target, err)
	}

	return readRegistryDistributionName(distroDir)
}

// registeredDistros returns a slice of the registered distros.
//
// It is analogous to
//  `wsl.exe --list`
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
	for _, skName := range subkeys {
		if skName == "AppxInstallerCache" {
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
//   `Software\Microsoft\Windows\CurrentVersion\Lxss\{ee8aef7a-846f-4561-a028-79504ce65cd3}`.
//
// Then, the registryDir is
//   `{ee8aef7a-846f-4561-a028-79504ce65cd3}`
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
