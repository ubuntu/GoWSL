// This file contains windows-specific typedefs and constants
package WslApi

import (
	"fmt"
	"os/exec"
	"sync"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

var (
	// WSL api
	wslApiDll                       = syscall.NewLazyDLL("wslapi.dll")
	wslConfigureDistribution        = wslApiDll.NewProc("WslConfigureDistribution")
	wslGetDistributionConfiguration = wslApiDll.NewProc("WslGetDistributionConfiguration")
	wslLaunch                       = wslApiDll.NewProc("WslLaunch")
	wslLaunchInteractive            = wslApiDll.NewProc("WslLaunchInteractive")
	wslRegisterDistribution         = wslApiDll.NewProc("WslRegisterDistribution")
	wslUnregisterDistribution       = wslApiDll.NewProc("WslUnregisterDistribution")
)

const (
	lxssRegistry = registry.CURRENT_USER
	lxssPath     = `Software\Microsoft\Windows\CurrentVersion\Lxss\`
)

func Shutdown() error {
	return exec.Command("wsl.exe", "--shutdown").Run()
}

func (distro Distro) Terminate() error {
	return exec.Command("wsl.exe", "--terminate", distro.Name).Run()
}

// ListDistros returns a slice of the registered distros
func registeredDistros() ([]Distro, error) {
	lxssKey, err := registry.OpenKey(lxssRegistry, lxssPath, registry.READ)
	if err != nil {
		return nil, fmt.Errorf("cannot list distros: failed to open lxss registry: %v", err)
	}
	defer lxssKey.Close()

	lxssData, err := lxssKey.Stat()
	if err != nil {
		panic(err)
	}

	subkeys, err := lxssKey.ReadSubKeyNames(int(lxssData.SubKeyCount))
	if err != nil {
		panic(err)
	}

	distroCh := make(chan Distro)
	errorCh := make(chan error)

	wg := sync.WaitGroup{}
	for _, skName := range subkeys {
		if skName == "AppxInstallerCache" {
			continue // Not a distro
		}

		skName := skName
		wg.Add(1)
		go func() {
			defer wg.Done()
			d, e := readRegistryDistributionName(skName)
			errorCh <- e
			if e != nil {
				distroCh <- Distro{Name: "MALFORMED_DISTRO"}
				return
			}
			distroCh <- Distro{Name: d}
		}()
	}

	go func() {
		defer close(distroCh)
		defer close(errorCh)
		wg.Wait()
	}()

	// Collecting results
	distros := []Distro{}
	e, oke := <-errorCh
	d, okd := <-distroCh

	for okd && oke {
		if e != nil {
			return []Distro{}, e
		}
		distros = append(distros, d)
		e, oke = <-errorCh
		d, okd = <-distroCh
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
	keyPath := lxssPath + registryDir
	key, err := registry.OpenKey(lxssRegistry, keyPath, registry.QUERY_VALUE)
	target := "DistributionName"

	if err != nil {
		return "", fmt.Errorf("cannot find key %s: %v", keyPath, err)
	}
	defer key.Close()

	name, _, err := key.GetStringValue(target)
	if err != nil {
		return "", fmt.Errorf("cannot find %s:%s : %v", keyPath, target, err)
	}
	return name, nil
}
