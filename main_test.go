package wsl_test

// This file contains testing functionality

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

	"github.com/0xrawsec/golang-utils/log"
	"github.com/stretchr/testify/require"
)

const (
	namePrefix  string = "wsltesting"
	emptyRootFs string = `.\images\empty.tar.gz` // Empty non-functional image. It registers instantly.
	jammyRootFs string = `.\images\jammy.tar.gz` // Fully functional rootfs
)

func TestMain(m *testing.M) {
	exitVal := m.Run()

	err := wsl.Shutdown()
	if err != nil {
		log.Warnf("cleanup: Failed to shutdown WSL")
	}
	cleanUpTestWslInstances()

	os.Exit(exitVal)
}

// uniqueID generates unique ID for distro names.
func uniqueID() string {
	rand.Seed(time.Now().UnixNano())
	// No need for this to be cryptographically secure
	return fmt.Sprintf("%d", rand.Intn(100_000_000)) //nolint:gosec
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

// Generates a unique distro name. It does not create the distro.
func uniqueDistroName(t *testing.T) string {
	t.Helper()
	const maxAttempts = 10
	for i := 0; i < maxAttempts; i++ {
		d := wsl.Distro{Name: sanitizeDistroName(fmt.Sprintf("%s_%s_%s", namePrefix, t.Name(), uniqueID()))}
		// Ensuring no name collision
		exists, err := d.IsRegistered()
		if err != nil {
			t.Logf("Setup: error in test distro name uniqueness check: %v", err)
			continue
		}
		if exists {
			t.Logf("Setup: name collision generating test distro: %q.", d.Name)
			continue
		}
		return d.Name
	}
	require.Fail(t, "Setup: failed to generate a unique name for the test distro.")
	return ""
}

// newTestDistro creates and registers a new distro with a mangled name and adds it to list of distros to remove.
func newTestDistro(t *testing.T, rootfs string) wsl.Distro {
	t.Helper()

	d := wsl.Distro{Name: uniqueDistroName(t)}
	t.Logf("Setup: Registering %q\n", d.Name)

	powershellInstallDistro(t, d.Name, rootfs)

	t.Cleanup(func() {
		err := cleanUpWslInstance(d)
		if err != nil {
			t.Logf("Cleanup: %v\n", err)
		}
	})

	t.Logf("Setup: Distro %q registered\n", d.Name)
	return d
}

// powershellInstallDistro installs using powershell to decouple the tests from Distro.Register
// CommandContext sometimes fails to stop it, so a more aggressive approach is taken by rebooting WSL.
// TODO: Consider if we want to retry.
func powershellInstallDistro(t *testing.T, distroName string, rootfs string) {
	t.Helper()

	// Timeout to attempt a graceful failure
	const gracefulTimeout = 60 * time.Second

	// Timeout to shutdown WSL
	const aggressiveTimeout = gracefulTimeout + 10*time.Second

	// Cannot use context.WithTimeout because I want to quit by doing wsl --shutdown
	expired := time.After(aggressiveTimeout)

	type combinedOutput struct {
		output string
		err    error
	}
	cmdOut := make(chan combinedOutput)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), gracefulTimeout)
		defer cancel()
		cmd := fmt.Sprintf("$env:WSL_UTF8=1 ; wsl.exe --import %s %s %s", distroName, t.TempDir(), rootfs)
		o, e := exec.CommandContext(ctx, "powershell.exe", "-Command", cmd).CombinedOutput() //nolint:gosec

		cmdOut <- combinedOutput{output: string(o), err: e}
		close(cmdOut)
	}()

	var output combinedOutput
	select {
	case <-expired:
		t.Logf("Setup: installation of WSL distro %s got stuck. Rebooting WSL.", distroName)
		e := exec.Command("wsl.exe", "--shutdown").Run()
		require.NoError(t, e, "Setup: failed to shutdown WSL after distro installation got stuck")

		// Almost guaranteed to error out here
		output = <-cmdOut
		require.NoError(t, output.err, output.output)

		require.Fail(t, "Setup: unknown state: successfully registered while WSL was shut down, stdout+stderr:", output.output)
	case output = <-cmdOut:
	}
	require.NoErrorf(t, output.err, "Setup: failed to register %q: %s", distroName, output.output)
}

// cleanUpTestWslInstances finds all distros with a prefixed name and unregisters them.
func cleanUpTestWslInstances() {
	testInstances, err := registeredTestWslInstances()
	if err != nil {
		return
	}
	if len(testInstances) != 0 {
		s := ""
		for _, d := range testInstances {
			s = s + "\n - " + d.Name
		}
		log.Warnf("Cleanup: The following WSL distros were not properly cleaned up:%s", s)
	}

	for _, d := range testInstances {
		err := cleanUpWslInstance(d)
		if err != nil {
			log.Warnf("Cleanup: %v\n", err)
		}
	}
}

// cleanUpWslInstance checks if a distro exists and if it does, it unregisters it.
func cleanUpWslInstance(distro wsl.Distro) error {
	if r, err := distro.IsRegistered(); err == nil && !r {
		return nil
	}
	cmd := fmt.Sprintf("$env:WSL_UTF8=1 ; wsl.exe --unregister %s", distro.Name)
	_, err := exec.Command("powershell.exe", "-command", cmd).CombinedOutput() //nolint: gosec
	if err != nil {
		return fmt.Errorf("failed to clean up test WSL distro %q: %v", distro.Name, err)
	}
	return nil
}

// registeredTestWslInstances finds all distros with a mangled name.
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
