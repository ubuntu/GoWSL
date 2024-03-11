package gowsl_test

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/0xrawsec/golang-utils/log"
	"github.com/stretchr/testify/require"
	wsl "github.com/ubuntu/gowsl"
)

// sanitizeDistroName sanitizes the name of the disto as much as possible.
func sanitizeDistroName(candidateName string) string {
	r := strings.NewReplacer(
		`/`, `--`,
		` `, `_`,
		`\`, `--`,
		`:`, `_`,
	)
	return r.Replace(candidateName)
}

// Generates a unique distro name. It does not create the distro.
func uniqueDistroName(t *testing.T) string {
	t.Helper()

	//nolint:gosec // No need to be cryptographically secure for this
	return sanitizeDistroName(fmt.Sprintf("%s_%s_%d", namePrefix, t.Name(), rand.Uint64()))
}

// newTestDistro creates and registers a new distro with a mangled name and adds it to list of distros to remove.
//
//nolint:revive // No, I wont' put the context before the *testing.T.
func newTestDistro(t *testing.T, ctx context.Context, rootfs string) wsl.Distro {
	t.Helper()

	d := wsl.NewDistro(ctx, uniqueDistroName(t))
	t.Logf("Setup: Registering %q\n", d.Name())

	installDistro(t, ctx, d.Name(), t.TempDir(), rootfs)

	t.Cleanup(func() {
		err := uninstallDistro(d, false)
		if err != nil {
			t.Logf("Cleanup: %v\n", err)
		}
	})

	t.Logf("Setup: Distro %q registered\n", d.Name())
	return d
}

// cleanUpTestWslInstances finds all distros with a prefixed name and unregisters them.
func cleanUpTestWslInstances(ctx context.Context) {
	testInstances, err := testDistros(ctx)

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
		err := uninstallDistro(d, true)
		if err != nil {
			log.Warnf("Cleanup: %v\n", err)
		}
	}
}

// backUpDefaultDistro returns a function to restore the default distro so the machine is restored to
// its pre-testing state.
func backUpDefaultDistro(ctx context.Context) (func(), error) {
	distroName, ok, err := defaultDistro(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to back up default distro: %v", err)
	}
	if !ok {
		return func() {}, nil // No distros registered: no backup needed
	}
	restore := func() {
		if err := setDefaultDistro(ctx, distroName); err != nil {
			log.Warnf("failed to restore default distro: %v", err)
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
//nolint:revive // We've agreed to put testing.T before the context
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
		//nolint:errcheck
		// not checking error because it is guaranteed to fail: it can only
		// finish by being interrupted. This is the intended behaviour.
		cmd.Wait()
	}
}

// testDistros finds all distros with a mangled name.
func testDistros(ctx context.Context) (distros []wsl.Distro, err error) {
	registered, err := registeredDistros(ctx)
	if err != nil {
		return registered, err
	}

	for idx := range registered {
		if !strings.HasPrefix(registered[idx].Name(), namePrefix) {
			continue
		}
		distros = append(distros, registered[idx])
	}

	return distros, nil
}
