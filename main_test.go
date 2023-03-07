package gowsl_test

// This file contains testing functionality

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/0xrawsec/golang-utils/log"
	"github.com/stretchr/testify/require"
	wsl "github.com/ubuntu/gowsl"
)

const (
	namePrefix  string = "wsltesting"
	emptyRootFs string = `./images/empty.tar.gz`  // Empty non-functional image. It registers instantly.
	rootFs      string = `./images/rootfs.tar.gz` // Fully functional rootfs
)

func TestMain(m *testing.M) {
	restore, err := backUpDefaultDistro()
	if err != nil {
		log.Errorf("setup: %v", err)
		os.Exit(1)
	}
	defer restore()

	exitVal := m.Run()

	err = wsl.Shutdown()
	if err != nil {
		log.Warnf("cleanup: Failed to shutdown WSL")
	}
	cleanUpTestWslInstances()

	os.Exit(exitVal)
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
		//nolint:gosec
		// No need for this random number to be cryptographically secure
		d := wsl.NewDistro(sanitizeDistroName(fmt.Sprintf("%s_%s_%d", namePrefix, t.Name(), rand.Uint32())))

		// Ensuring no name collision
		exists, err := d.IsRegistered()
		if err != nil {
			t.Logf("Setup: error in test distro name uniqueness check: %v", err)
			continue
		}
		if exists {
			t.Logf("Setup: name collision generating test distro: %q.", d.Name())
			continue
		}
		return d.Name()
	}
	require.Fail(t, "Setup: failed to generate a unique name for the test distro.")
	return ""
}

// newTestDistro creates and registers a new distro with a mangled name and adds it to list of distros to remove.
func newTestDistro(t *testing.T, rootfs string) wsl.Distro {
	t.Helper()

	d := wsl.NewDistro(uniqueDistroName(t))
	t.Logf("Setup: Registering %q\n", d.Name())

	powershellInstallDistro(t, d.Name(), rootfs)

	t.Cleanup(func() {
		err := cleanUpWslInstance(d)
		if err != nil {
			t.Logf("Cleanup: %v\n", err)
		}
	})

	t.Logf("Setup: Distro %q registered\n", d.Name())
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
			s = s + "\n - " + d.Name()
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
	cmd := fmt.Sprintf("$env:WSL_UTF8=1 ; wsl.exe --unregister %s", distro.Name())
	_, err := exec.Command("powershell.exe", "-command", cmd).CombinedOutput() //nolint: gosec
	if err != nil {
		return fmt.Errorf("failed to clean up test WSL distro %q: %v", distro.Name(), err)
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
		distros = append(distros, wsl.NewDistro(line))
	}

	return distros, nil
}

// defaultDistro gets the default distro's name via wsl.exe to bypass wsl.DefaultDistro in order to
// better decouple tests.
func defaultDistro() (string, error) {
	out, err := exec.Command("powershell.exe", "-Command", "$env:WSL_UTF8=1; wsl.exe --list --verbose").CombinedOutput()
	if err != nil {
		if target := (&exec.ExitError{}); !errors.As(err, &target) {
			return "", fmt.Errorf("failed to find current default distro: %v", err)
		}
		// cannot read from target.StdErr because message is printed to Stdout
		if !strings.Contains(string(out), "Wsl/WSL_E_DEFAULT_DISTRO_NOT_FOUND") {
			return "", fmt.Errorf("failed to find current default distro: %v. Output: %s", err, out)
		}
		return "", nil // No distros installed: no default
	}

	s := bufio.NewScanner(bytes.NewReader(out))
	s.Scan() // Ignore first line (table header)
	for s.Scan() {
		line := s.Text()
		if !strings.HasPrefix(line, "*") {
			continue
		}
		data := strings.Fields(line)
		if len(data) < 2 {
			return "", fmt.Errorf("failed to parse 'wsl.exe --list --verbose' output, line %q", line)
		}
		return data[1], nil
	}

	if err := s.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("failed to find default distro in 'wsl.exe --list --verbose' output:\n%s", string(out))
}

// backUpDefaultDistro returns a function to restore the default distro so the machine is restored to
// its pre-testing state.
func backUpDefaultDistro() (func(), error) {
	distro, err := defaultDistro()
	if err != nil {
		return nil, fmt.Errorf("failed to back up default distro: %v", err)
	}
	if len(distro) == 0 {
		return func() {}, nil // No distros registered: no backup needed
	}
	restore := func() {
		//nolint: gosec // G204: Subprocess launched with a potential tainted input or cmd arguments
		// No threat of code injection, wsl.exe will only interpret this text as a distro name
		// and throw Wsl/Service/WSL_E_DISTRO_NOT_FOUND if it does not exist.
		out, err := exec.Command("wsl.exe", "--set-default", distro).CombinedOutput()
		if err != nil {
			log.Warnf("failed to set distro %q back as default: %v. Output: %s", distro, err, out)
		}
	}
	return restore, nil
}

// keepAwake sends an endless command to the distro to keep it awake.
// The distro will stay awake until the context is cancelled or the cancel
// function is called.
//
// You must call the cancel function to release the associated resources.
//
//nolint:revive
func keepAwake(t *testing.T, ctx context.Context, d *wsl.Distro) context.CancelFunc {
	// Linter says "context-as-argument: context.Context should be the first parameter of a function"
	// This is an abomination that we won't stand for
	t.Helper()
	ctx, cancel := context.WithCancel(ctx)

	cmd := d.Command(ctx, "sleep infinity")
	err := cmd.Start()
	if err != nil {
		cancel()
		require.Failf(t, "failed to Start command to keep the distro alive:", "%v", err)
	}

	return func() {
		cancel()
		//nolint: errcheck
		// not checking error because it is guaranteed to fail: it can only
		// finish by being interrupted. This is the intended behaviour.
		cmd.Wait()
	}
}
