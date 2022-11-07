package WslApi_test

import (
	"WslApi"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

const (
	distroSuffix string = "wsltesting"
	jammyRootFs  string = `C:\Users\edu19\Work\images\jammy.tar.gz`
)

type Tester struct {
	*testing.T
	distros *[]WslApi.Distro
}

func TestMain(m *testing.M) {
	WslApi.Shutdown()
	cleanupAllDistros()

	exitVal := m.Run()

	WslApi.Shutdown()
	cleanupAllDistros()

	os.Exit(exitVal)
}

// NewTester extends Tester with some WSL-specific functionality and cleanup
func NewTester(tst *testing.T) (t Tester) {
	distros := new([]WslApi.Distro)
	t = Tester{T: tst, distros: distros}
	t.Cleanup(func() {
		cleanUpDistros(*distros)
	})
	return t
}

// NewDistro creates a new distro with a mangled name and adds it to list of distros to remove.
// Note that the distro is not registered.
func (t *Tester) NewDistro(name string) WslApi.Distro {
	d := WslApi.Distro{Name: t.mangleName(name)}
	*t.distros = append(*t.distros, d)
	return d
}

// mangleName avoids name collisions with existing distros by adding a suffix
func (t Tester) mangleName(name string) string {
	return fmt.Sprintf("%s_%s_%s", name, t.Name(), distroSuffix)
}

// cleanupAllDistros finds all distros with a mangled name and unregisters them
func cleanupAllDistros() {
	testDistros, err := findTestDistros()
	if err != nil {
		return
	}
	if len(testDistros) != 0 {
		fmt.Printf("The following distros were not properly cleaned up: %v\n", testDistros)
	}
	cleanUpDistros(testDistros)
}

func cleanUpDistros(distros []WslApi.Distro) {
	for _, distro := range distros {
		if r, err := distro.IsRegistered(); err == nil && !r {
			continue
		}

		err := distro.Unregister()
		if err != nil {
			name, test := unmangleName(distro.Name)
			fmt.Printf("failed to clean up test distro (name=%s, test=%s)\n", name, test)
		}
	}
}

// unmangleName retrieves encoded info from distro name
func unmangleName(mangledName string) (name string, test string) {
	words := strings.Split(mangledName, "_")
	l := len(words)
	name = strings.Join(words[:l-2], "_")
	test = words[l-2]
	return name, test
}

// findTestDistros finds all distros with a mangled name
func findTestDistros() ([]WslApi.Distro, error) {
	distros := []WslApi.Distro{}

	outp, err := exec.Command("powershell.exe", "-command", "$env:WSL_UTF8=1 ; wsl.exe --list --quiet").CombinedOutput()
	if err != nil {
		return distros, err
	}

	for _, line := range strings.Fields(string(outp)) {
		if !strings.HasSuffix(line, distroSuffix) {
			continue
		}
		distros = append(distros, WslApi.Distro{Name: line})
	}

	return distros, nil
}
