package WslApi_test

// This file conatains testing functionality

import (
	"WslApi"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	nameSuffix  string = "wsltesting"
	emptyRootFs string = `C:\Users\edu19\Work\images\empty.tar.gz` // Empty non-functional image. It registers instantly.
	jammyRootFs string = `C:\Users\edu19\Work\images\jammy.tar.gz` // Fully functional rootfs
)

type Tester struct {
	*testing.T
	instances []WslApi.Instance
	tmpdirs   []string
}

func TestMain(m *testing.M) {
	fullCleanup := func() {
		WslApi.Shutdown()
		cleanUpTestWslInstances()
	}

	fullCleanup()
	defer fullCleanup()

	exitVal := m.Run()
	os.Exit(exitVal)
}

// NewTester extends Tester with some WSL-specific functionality and cleanup
func NewTester(tst *testing.T) *Tester {
	t := Tester{T: tst}
	t.Cleanup(func() {
		t.cleanUpWslInstances()
		t.cleanUpTempDirectories()
	})
	return &t
}

// NewWslInstance creates a new instance with a mangled name and adds it to list of instances to remove.
// Note that the instance is not registered.
func (t *Tester) NewWslInstance(name string) WslApi.Instance {
	d := WslApi.Instance{Name: t.mangleName(name)}
	t.instances = append(t.instances, d)
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

func (t *Tester) cleanUpWslInstances() {
	cleanUpWslInstancess(t.instances)
}

func (t *Tester) cleanUpTempDirectories() {
	for _, dir := range t.tmpdirs {
		dir := dir
		err := os.RemoveAll(dir)
		if err != nil {
			t.Logf("Failed to remove temp directory %s: %v\n", dir, err)
		}
	}
}

// cleanUpTestWslInstances finds all instances with a mangled name and unregisters them
func cleanUpTestWslInstances() {
	testInstances, err := RegisteredTestWslInstances()
	if err != nil {
		return
	}
	if len(testInstances) != 0 {
		fmt.Printf("The following WSL instances were not properly cleaned up: %v\n", testInstances)
	}
	cleanUpWslInstancess(testInstances)
}

func cleanUpWslInstancess(instances []WslApi.Instance) {
	for _, i := range instances {

		if r, err := i.IsRegistered(); err == nil && !r {
			return
		}
		err := i.Unregister()
		if err != nil {
			name, test := unmangleName(i.Name)
			fmt.Printf("Failed to clean up test WSL instance (name=%s, test=%s)\n", name, test)
		}

	}
}

// RegisteredTestWslInstances finds all instances with a mangled name
func RegisteredTestWslInstances() ([]WslApi.Instance, error) {
	instances := []WslApi.Instance{}

	outp, err := exec.Command("powershell.exe", "-command", "$env:WSL_UTF8=1 ; wsl.exe --list --quiet").CombinedOutput()
	if err != nil {
		return instances, err
	}

	for _, line := range strings.Fields(string(outp)) {
		if !strings.HasSuffix(line, nameSuffix) {
			continue
		}
		instances = append(instances, WslApi.Instance{Name: line})
	}

	return instances, nil
}

// mangleName avoids name collisions with existing instances by adding a suffix
func (t Tester) mangleName(name string) string {
	return fmt.Sprintf("%s_%s_%s", name, strings.ReplaceAll(t.Name(), "/", "--"), nameSuffix)
}

// unmangleName retrieves encoded info from a mangled DistroName
func unmangleName(mangledName string) (name string, test string) {
	words := strings.Split(mangledName, "_")
	l := len(words)
	name = strings.Join(words[:l-2], "_")
	test = words[l-2]
	return name, test
}

// registerFromPowershell registers a WSL instance bypassing the WslApi module, for better test segmentation
func (t *Tester) RegisterFromPowershell(i WslApi.Instance, image string) {
	tmpdir, err := t.NewTestDir(i.Name)
	require.NoError(t, err)

	cmdString := fmt.Sprintf("$env:WSL_UTF8=1 ; wsl.exe --import %s %s %s", i.Name, tmpdir, jammyRootFs)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // WSL sometimes gets stuck installing
	defer cancel()

	output, err := exec.CommandContext(ctx, "powershell.exe", "-Command", cmdString).CombinedOutput()
	require.NoError(t, err, string(output))
}
