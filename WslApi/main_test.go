package WslApi_test

import (
	"WslApi"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	distroSuffix string = "wsltesting"
	jammyRootFs  string = `C:\Users\edu19\Work\images\jammy.tar.gz`
)

type Tester struct {
	*testing.T
	distros []WslApi.Distro
	tmpdirs []string
}

func TestMain(m *testing.M) {
	WslApi.Shutdown()
	cleanUpTestDistros()

	exitVal := m.Run()

	WslApi.Shutdown()
	cleanUpTestDistros()

	os.Exit(exitVal)
}

// NewTester extends Tester with some WSL-specific functionality and cleanup
func NewTester(tst *testing.T) *Tester {
	t := Tester{T: tst}
	t.Cleanup(func() {
		t.cleanUpDistros()
		t.cleanUpTempDirectories()
	})
	return &t
}

// NewDistro creates a new distro with a mangled name and adds it to list of distros to remove.
// Note that the distro is not registered.
func (t *Tester) NewDistro(name string) WslApi.Distro {
	d := WslApi.Distro{Name: t.mangleName(name)}
	t.distros = append(t.distros, d)
	return d
}

// NewTestDir creates a unique directory and adds it to the list of dirs to remove
func (t *Tester) NewTestDir(prefix string) (string, error) {
	clean_prefix := strings.Replace(t.Name()+prefix, "/", "_", -1)
	tmpdir, err := ioutil.TempDir(os.TempDir(), clean_prefix)
	if err != nil {
		return "", err
	}

	t.tmpdirs = append(t.tmpdirs, tmpdir)
	return tmpdir, nil
}

func (t *Tester) cleanUpDistros() {
	cleanUpDistros(t.distros)
}

func (t *Tester) cleanUpTempDirectories() {
	for _, dir := range t.tmpdirs {
		dir := dir
		go func() {
			err := os.RemoveAll(dir)
			t.Logf("Failed to remove temp directory %s: %v", dir, err)
		}()
	}
}

// cleanUpTestDistros finds all distros with a mangled name and unregisters them
func cleanUpTestDistros() {
	testDistros, err := RegisteredTestDistros()
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
			return
		}
		err := distro.Unregister()
		if err != nil {
			name, test := unmangleName(distro.Name)
			fmt.Printf("Failed to clean up test distro (name=%s, test=%s)\n", name, test)
		}

	}
}

// RegisteredTestDistros finds all distros with a mangled name
func RegisteredTestDistros() ([]WslApi.Distro, error) {
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

// mangleName avoids name collisions with existing distros by adding a suffix
func (t Tester) mangleName(name string) string {
	return fmt.Sprintf("%s_%s_%s", name, strings.ReplaceAll(t.Name(), "/", "--"), distroSuffix)
}

// unmangleName retrieves encoded info from distro name
func unmangleName(mangledName string) (name string, test string) {
	words := strings.Split(mangledName, "_")
	l := len(words)
	name = strings.Join(words[:l-2], "_")
	test = words[l-2]
	return name, test
}

// registerViaCommandline registers a distro bypassing the WslApi module, for better test segmentation
func registerViaCommandline(t *Tester, distro WslApi.Distro) {
	tmpdir, err := t.NewTestDir(distro.Name)
	require.NoError(t, err)

	str, err := exec.Command("wsl.exe", "--import", distro.Name, tmpdir, jammyRootFs).CombinedOutput()
	require.NoError(t, err, str)
}
