package gowsl_test

import (
	wsl "github.com/ubuntu/gowsl"

	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	testCases := map[string]struct {
		distroSuffix string
		rootfs       string
		wantError    bool
	}{
		"happy path":          {rootfs: rootFs},
		"wrong name":          {rootfs: rootFs, distroSuffix: "--I contain whitespace", wantError: true},
		"null char in name":   {rootfs: rootFs, distroSuffix: "--I \x00 contain a null char", wantError: true},
		"null char in rootfs": {rootfs: "jammy\x00.tar.gz", wantError: true},
		"inexistent rootfs":   {rootfs: "I am not a real file.tar.gz", wantError: true},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			d := wsl.NewDistro(uniqueDistroName(t) + tc.distroSuffix)
			defer func() {
				err := cleanUpWslInstance(d)
				if err != nil {
					t.Logf("Cleanup: %v", err)
				}
			}()

			cancel := wslShutdownTimeout(t, time.Minute)
			t.Logf("Registering %q", d.Name())
			err := d.Register(tc.rootfs)
			cancel()
			t.Log("Registration completed")

			if tc.wantError {
				require.Errorf(t, err, "Unexpected success in registering distro %q.", d.Name())
				return
			}
			require.NoError(t, err, "Unexpected failure in registering distro %q.", d.Name())
			list, err := registeredTestWslInstances()
			require.NoError(t, err, "Failed to read list of registered test distros.")
			require.Contains(t, list, d, "Failed to find distro in list of registered distros.")

			// Testing double registration failure
			cancel = wslShutdownTimeout(t, time.Minute)
			t.Logf("Registering %q", d.Name())
			err = d.Register(tc.rootfs)
			cancel()
			t.Log("Registration completed")

			require.Error(t, err, "Unexpected success in registering distro that was already registered.")
		})
	}
}

func TestRegisteredDistros(t *testing.T) {
	d1 := newTestDistro(t, emptyRootFs)
	d2 := newTestDistro(t, emptyRootFs)
	d3 := wsl.NewDistro(uniqueDistroName(t))

	list, err := wsl.RegisteredDistros()
	require.NoError(t, err)

	assert.Contains(t, list, d1)
	assert.Contains(t, list, d2)
	assert.NotContains(t, list, d3)
}

func TestIsRegistered(t *testing.T) {
	tests := map[string]struct {
		distroSuffix   string
		register       bool
		wantError      bool
		wantRegistered bool
	}{
		"nominal":           {register: true, wantRegistered: true},
		"inexistent":        {},
		"null char in name": {distroSuffix: "Oh no, there is a \x00!"},
	}

	for name, config := range tests {
		name := name
		config := config

		t.Run(name, func(t *testing.T) {
			var distro wsl.Distro
			if config.register {
				distro = newTestDistro(t, emptyRootFs)
			} else {
				distro = wsl.NewDistro(uniqueDistroName(t))
			}

			reg, err := distro.IsRegistered()
			if config.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if config.wantRegistered {
				require.True(t, reg)
			} else {
				require.False(t, reg)
			}
		})
	}
}

func TestUnregister(t *testing.T) {
	realDistro := newTestDistro(t, emptyRootFs)
	fakeDistro := wsl.NewDistro(uniqueDistroName(t))
	wrongDistro := wsl.NewDistro(uniqueDistroName(t) + "This Distro \x00 has a null char")

	testCases := map[string]struct {
		distro    *wsl.Distro
		wantError bool
	}{
		"happy path":        {distro: &realDistro},
		"not registered":    {distro: &fakeDistro, wantError: true},
		"null char in name": {distro: &wrongDistro, wantError: true},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			d := *tc.distro

			cancel := wslShutdownTimeout(t, time.Minute)
			t.Logf("Unregistering %q", d.Name())
			err := d.Unregister()
			cancel()
			t.Log("Unregistration completed")

			if tc.wantError {
				require.Errorf(t, err, "Unexpected success in unregistering distro %q.", d.Name())
			} else {
				require.NoError(t, err, "Unexpected failure in unregistering distro %q.", d.Name())
			}

			list, err := registeredTestWslInstances()
			require.NoError(t, err, "Failed to read list of registered test distros.")
			require.NotContains(t, list, d, "Found allegedly unregistered distro in list of registered distros.")
		})
	}
}

// wslShutdownTimeout starts a timer. When the timer finishes, WSL is shut down.
// Use the returned function to cancel it. Even if you time out, cancel should be
// called in order to deallocate resources. You can call cancel multiple times without
// adverse effect.
func wslShutdownTimeout(t *testing.T, timeout time.Duration) (cancel func()) {
	t.Helper()

	stop := make(chan struct{})
	var cancelled bool

	go func() {
		timer := time.NewTimer(timeout)
		select {
		case <-stop:
		case <-timer.C:
			t.Logf("wslShutdownTimeout timed out")
			err := wsl.Shutdown()
			require.NoError(t, err, "Failed to shutdown WSL after it timed out")
			<-stop
		}
		timer.Stop()
	}()

	return func() {
		if cancelled {
			return
		}
		cancelled = true
		stop <- struct{}{}
		close(stop)
	}
}
