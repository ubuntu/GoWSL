package wsl_test

// This file conatains testing functionality

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
	"wsl"

	"github.com/stretchr/testify/require"
)

const (
	namePrefix  string = "wsltesting"
	emptyRootFs string = `.\images\empty.tar.gz` // Empty non-functional image. It registers instantly.
	jammyRootFs string = `.\images\jammy.tar.gz` // Fully functional rootfs
)

func TestMain(m *testing.M) {

	exitVal := m.Run()

	wsl.Shutdown()
	cleanUpTestWslInstances()

	os.Exit(exitVal)
}

// distroCounter is used to generate unique IDs for distro names

// uniqueId generates unique ID for distro names.
func uniqueId() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%d", rand.Intn(100_000_000))
}

// sanitizeDistroName sanitizes the name of the disto as much as possible.
func sanitizeDistroName(candidateName string) string {
	r := strings.NewReplacer(
		`/`, `--`,
		` `, `_`,
		`\`, `--`,
	)
	return r.Replace(candidateName)
}

// newTestDistro creates and registers a new distro with a mangled name and adds it to list of distros to remove.
func newTestDistro(t *testing.T, rootfs string) wsl.Distro {
	d := wsl.Distro{Name: sanitizeDistroName(fmt.Sprintf("%s_%s_%s", namePrefix, t.Name(), uniqueId()))}

	temp := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // WSL sometimes gets stuck installing
	defer cancel()
	cmd := fmt.Sprintf("$env:WSL_UTF8=1 ; wsl.exe --import %s %s %s", d.Name, temp, jammyRootFs)
	output, err := exec.CommandContext(ctx, "powershell.exe", "-Command", cmd).CombinedOutput()
	require.NoError(t, err, string(output))

	t.Cleanup(func() {
		err := cleanUpWslInstance(d)
		if err != nil {
			t.Logf("Teardown: %v\n", err)
		}
	})

	return d
}

// cleanUpTestWslInstances finds all distros with a prefixed name and unregisters them
func cleanUpTestWslInstances() {
	testInstances, err := registeredTestWslInstances()
	if err != nil {
		return
	}
	if len(testInstances) != 0 {
		fmt.Printf("The following WSL distros were not properly cleaned up: %v\n", testInstances)
	}

	for _, d := range testInstances {
		cleanUpWslInstance(d)
	}
}

// cleanUpWslInstance checks if a distro exists and if it does, it unregisters it
func cleanUpWslInstance(distro wsl.Distro) error {
	if r, err := distro.IsRegistered(); err == nil && !r {
		return nil
	}
	cmd := fmt.Sprintf("$env:WSL_UTF8=1 ; wsl.exe --unregister %s", distro.Name)
	_, err := exec.Command("powershell.exe", "-command", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clean up test WSL distro %q: %v\n", distro.Name, err)
	}
	return nil
}

// registeredTestWslInstances finds all distros with a mangled name
func registeredTestWslInstances() ([]wsl.Distro, error) {
	distros := []wsl.Distro{}

	outp, err := exec.Command("powershell.exe", "-command", "$env:WSL_UTF8=1 ; wsl.exe --list --quiet").CombinedOutput()
	if err != nil {
		return distros, err
	}

	for _, line := range strings.Fields(string(outp)) {
		if !strings.HasPrefix(line, namePrefix) {
			continue
		}
		distros = append(distros, wsl.Distro{Name: line})
	}

	return distros, nil
}
