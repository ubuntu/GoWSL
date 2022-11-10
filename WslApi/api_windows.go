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

func (i Instance) Terminate() error {
	return exec.Command("wsl.exe", "--terminate", i.Name).Run()
}

// registeredInstances returns a slice of the registered instances
func registeredInstances() ([]Instance, error) {
	lxssKey, err := registry.OpenKey(lxssRegistry, lxssPath, registry.READ)
	if err != nil {
		return nil, fmt.Errorf("cannot list instances: failed to open lxss registry: %v", err)
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

	instCh := make(chan Instance)
	errorCh := make(chan error)

	wg := sync.WaitGroup{}
	for _, skName := range subkeys {
		if skName == "AppxInstallerCache" {
			continue // Not a WSL instance
		}

		skName := skName
		wg.Add(1)
		go func() {
			defer wg.Done()
			d, e := readRegistryDistributionName(skName)
			errorCh <- e
			if e != nil {
				instCh <- Instance{Name: "MALFORMED_WSL_INSTANCE"}
				return
			}
			instCh <- Instance{Name: d}
		}()
	}

	go func() {
		defer close(instCh)
		defer close(errorCh)
		wg.Wait()
	}()

	// Collecting results
	instances := []Instance{}
	e, oke := <-errorCh
	d, okd := <-instCh

	for okd && oke {
		if e != nil {
			return []Instance{}, e
		}
		instances = append(instances, d)
		e, oke = <-errorCh
		d, okd = <-instCh
	}

	return instances, nil
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
