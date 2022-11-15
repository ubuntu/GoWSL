package wsl_test

// This file conatains testing functionality

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"
	"wsl"
	"wsl/rootfs"

	"github.com/stretchr/testify/require"
)

const (
	nameSuffix string = "wsltesting"
)

type Tester struct {
	*testing.T
	distros []wsl.Distro
	tmpdirs []string
}

func TestMain(m *testing.M) {

	fullCleanup()
	exitVal := m.Run()
	fullCleanup()

	os.Exit(exitVal)
}

func fullCleanup() {
	wsl.Shutdown()
	// Cleanup without nagging
	if cachedDistro != nil {
		cleanUpWslInstancess([]wsl.Distro{*cachedDistro})
	}
	// Cleanup with nagging
	cleanUpTestWslInstances()
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

// NewWslDistro creates a new distro with a mangled name and adds it to list of distros to remove.
// Note that the distro is not registered.
func (t *Tester) NewWslDistro(name string) wsl.Distro {
	d := wsl.Distro{Name: t.mangleName(name)}
	t.distros = append(t.distros, d)
	return d
}

var cachedDistro *wsl.Distro = nil

// CachedDistro provides a distro for non-destructive and generally immutable commands
// without having to create and destroy a new distro for it.
func (t *Tester) CachedDistro() wsl.Distro {
	if cachedDistro == nil {
		cachedDistro = &wsl.Distro{Name: fmt.Sprintf("reusableDistro_TestMain_%s", nameSuffix)}
		err := cachedDistro.Register(t.JammyRootFs())
		require.NoError(t, err)
	}
	return *cachedDistro
}

// NewTmpDir creates a unique directory and adds it to the list of dirs to remove
// Contary to testing.T.TmpDir, this is removed AFTER cleanup
func (t *Tester) NewTmpDir(prefix string) string {
	clean_prefix := strings.Replace(t.Name()+prefix, "/", "_", -1)
	tmpdir, err := ioutil.TempDir(os.TempDir(), clean_prefix)
	require.NoError(t, err)

	t.tmpdirs = append(t.tmpdirs, tmpdir)
	return tmpdir
}

func (t *Tester) cleanUpDistros() {
	cleanUpWslInstancess(t.distros)
	t.distros = []wsl.Distro{}
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

// cleanUpTestWslInstances finds all distros with a mangled name and unregisters them
func cleanUpTestWslInstances() {
	testInstances, err := RegisteredTestWslInstances()
	if err != nil {
		return
	}
	if len(testInstances) != 0 {
		fmt.Printf("The following WSL distros were not properly cleaned up: %v\n", testInstances)
	}
	cleanUpWslInstancess(testInstances)
}

func cleanUpWslInstancess(distros []wsl.Distro) {
	for _, i := range distros {

		if r, err := i.IsRegistered(); err == nil && !r {
			return
		}
		err := i.Unregister()
		if err != nil {
			name, test := unmangleName(i.Name)
			fmt.Printf("Failed to clean up test WSL distro (name=%s, test=%s)\n", name, test)
		}

	}
}

// RegisteredTestWslInstances finds all distros with a mangled name
func RegisteredTestWslInstances() ([]wsl.Distro, error) {
	distros := []wsl.Distro{}

	outp, err := exec.Command("powershell.exe", "-command", "$env:WSL_UTF8=1 ; wsl.exe --list --quiet").CombinedOutput()
	if err != nil {
		return distros, err
	}

	for _, line := range strings.Fields(string(outp)) {
		if !strings.HasSuffix(line, nameSuffix) {
			continue
		}
		distros = append(distros, wsl.Distro{Name: line})
	}

	return distros, nil
}

// mangleName avoids name collisions with existing distros by adding a suffix
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

// registerFromPowershell registers a WSL distro bypassing the wsl.module, for better test segmentation
func (t *Tester) RegisterFromPowershell(i wsl.Distro, image string) {
	tmpdir := t.NewTmpDir(i.Name)

	cmdString := fmt.Sprintf("$env:WSL_UTF8=1 ; wsl.exe --import %s %s %s", i.Name, tmpdir, t.JammyRootFs())

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // WSL sometimes gets stuck installing
	defer cancel()

	output, err := exec.CommandContext(ctx, "powershell.exe", "-Command", cmdString).CombinedOutput()
	require.NoError(t, err, string(output))
}

func (t *Tester) imagesDir() string {
	repo, err := os.Getwd()
	require.NoError(t, err)

	imagesDir := path.Join(repo, "images")
	err = os.MkdirAll(imagesDir, 075)
	require.NoError(t, err)
	return imagesDir
}

func (t *Tester) JammyRootFs() string {
	path, err := rootfs.Get(rootfs.Jammy, t.imagesDir())
	require.NoError(t, err)
	return path
}

func (t *Tester) KineticRootFs() string {
	path, err := rootfs.Get(rootfs.Kinetic, t.imagesDir())
	require.NoError(t, err)
	return path
}

// Empty non-functional image. It registers instantly.
func (t *Tester) EmptyRootfs() string {
	p := path.Join(t.NewTmpDir("emptytargz"), `empty.tar.gz`)

	_, err := os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		return p
	}
	require.NoError(t, err)

	err = ioutil.WriteFile(p, []byte{}, 0755)
	require.NoError(t, err)

	return p
}
