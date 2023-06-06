package gowsl_test

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	wsl "github.com/ubuntu/gowsl"
	"github.com/ubuntu/gowsl/mock"
)

func TestRegister(t *testing.T) {
	setupBackend(t, context.Background())

	testCases := map[string]struct {
		distroSuffix         string
		rootfs               string
		syscallError         bool
		registryInaccessible bool

		wantError bool
	}{
		"Success": {rootfs: rootFS},

		"Error when the distro name contains whitespace":       {rootfs: rootFS, distroSuffix: "--I contain whitespace", wantError: true},
		"Error when the distro name contains a null character": {rootfs: rootFS, distroSuffix: "--I \x00 contain a null char", wantError: true},
		"Error when the rootfs path contains a null character": {rootfs: "jammy\x00.tar.gz", wantError: true},
		"Error when the rootfs path does not exist":            {rootfs: "I am not a real file.tar.gz", wantError: true},

		// Mock-induced errors
		"Error when the registry fails to open": {rootfs: rootFS, registryInaccessible: true, wantError: true},
		"Error when the register syscall fails": {rootfs: rootFS, syscallError: true, wantError: true},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			ctx, modifyMock := setupBackend(t, context.Background())
			if tc.syscallError || tc.registryInaccessible {
				modifyMock(t, func(m *mock.Backend) {
					m.WslRegisterDistributionError = tc.syscallError
					m.OpenLxssKeyError = tc.registryInaccessible
				})
				defer modifyMock(t, (*mock.Backend).ResetErrors)
			}

			d := wsl.NewDistro(ctx, uniqueDistroName(t)+tc.distroSuffix)
			t.Cleanup(func() {
				err := uninstallDistro(d, false)
				if err != nil {
					t.Logf("Cleanup: %v", err)
				}
			})

			cancel := wslExeGuard(time.Minute)
			t.Logf("Registering %q", d.Name())
			err := d.Register(tc.rootfs)
			cancel()
			t.Log("Registration completed")

			if tc.wantError {
				require.Errorf(t, err, "Unexpected success in registering distro %q.", d.Name())
				return
			}
			require.NoError(t, err, "Unexpected failure in registering distro %q.", d.Name())
			list, err := testDistros(ctx)
			require.NoError(t, err, "Failed to read list of registered test distros.")
			require.Contains(t, list, d, "Failed to find distro in list of registered distros.")

			// Testing double registration failure
			cancel = wslExeGuard(time.Minute)
			t.Logf("Registering %q", d.Name())
			err = d.Register(tc.rootfs)
			cancel()
			t.Log("Registration completed")

			require.Error(t, err, "Unexpected success in registering distro that was already registered.")
		})
	}
}

func TestRegisteredDistros(t *testing.T) {
	testCases := map[string]struct {
		registryInaccessible bool

		wantErr bool
	}{
		"Success": {},

		// Mock-induced errors
		"Error when the registry cannot be accessed": {registryInaccessible: true, wantErr: true},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			if tc.registryInaccessible && !wsl.MockAvailable() {
				t.Skip("This test is only available with the mock enabled")
			}

			ctx, modifyMock := setupBackend(t, context.Background())

			d1 := newTestDistro(t, ctx, emptyRootFS)
			d2 := newTestDistro(t, ctx, emptyRootFS)
			d3 := wsl.NewDistro(ctx, uniqueDistroName(t))

			if tc.registryInaccessible {
				modifyMock(t, func(m *mock.Backend) {
					m.OpenLxssKeyError = true
				})
				defer modifyMock(t, (*mock.Backend).ResetErrors)
			}

			list, err := wsl.RegisteredDistros(ctx)
			if tc.wantErr {
				require.Error(t, err, "RegisteredDistros should have returned an error")
				return
			}
			require.NoError(t, err, "RegisteredDistros should have returned no errors")

			assert.Contains(t, list, d1)
			assert.Contains(t, list, d2)
			assert.NotContains(t, list, d3)
		})
	}
}

func TestIsRegistered(t *testing.T) {
	setupBackend(t, context.Background())

	testCases := map[string]struct {
		distroSuffix         string
		register             bool
		syscallError         bool
		registryInaccessible bool

		wantError      bool
		wantRegistered bool
	}{
		"Success with a registered distro":     {register: true, wantRegistered: true},
		"Success with a non-registered distro": {},

		"Error when the distro name has a null char": {distroSuffix: "Oh no, there is a \x00!"},

		// Mock-induced errors
		"Error when the registry cannot be accessed": {registryInaccessible: true, wantError: true},
	}

	for name, tc := range testCases {
		name := name
		tc := tc

		t.Run(name, func(t *testing.T) {
			ctx, modifyMock := setupBackend(t, context.Background())
			if tc.registryInaccessible {
				modifyMock(t, func(m *mock.Backend) {
					m.OpenLxssKeyError = true
				})
				defer modifyMock(t, (*mock.Backend).ResetErrors)
			}

			var distro wsl.Distro
			if tc.register {
				distro = newTestDistro(t, ctx, emptyRootFS)
			} else {
				distro = wsl.NewDistro(ctx, uniqueDistroName(t))
			}

			reg, err := distro.IsRegistered()
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.wantRegistered {
				require.True(t, reg)
			} else {
				require.False(t, reg)
			}
		})
	}
}

func TestUnregister(t *testing.T) {
	testCases := map[string]struct {
		distroname           string
		nonRegistered        bool
		syscallError         bool
		registryInaccessible bool

		wantError bool
	}{
		"Success": {},

		"Error with a non-registered distro": {nonRegistered: true, wantError: true},
		"Error with a null char in name":     {nonRegistered: true, distroname: "This Distro \x00 has a null char", wantError: true},

		// Mock-induced errors
		"Error when the registry fails to open": {registryInaccessible: true, wantError: true},
		"Error when the syscall fails":          {syscallError: true, wantError: true},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			if (tc.syscallError || tc.registryInaccessible) && !wsl.MockAvailable() {
				t.Skip("This test is only available with the mock enabled")
			}

			ctx, modifyMock := setupBackend(t, context.Background())

			var d wsl.Distro
			if tc.nonRegistered {
				d = wsl.NewDistro(ctx, uniqueDistroName(t)+tc.distroname)
			} else {
				d = newTestDistro(t, ctx, emptyRootFS)
			}

			if tc.registryInaccessible || tc.syscallError {
				modifyMock(t, func(m *mock.Backend) {
					m.WslUnregisterDistributionError = tc.syscallError
					m.OpenLxssKeyError = tc.registryInaccessible
				})
				defer modifyMock(t, (*mock.Backend).ResetErrors)
			}

			t.Logf("Unregistering %q", d.Name())

			cancel := wslExeGuard(time.Minute)
			err := d.Unregister()
			cancel()

			t.Log("Unregistration completed")

			if tc.wantError {
				require.Errorf(t, err, "Unexpected success in unregistering distro %q.", d.Name())
				return
			}
			require.NoError(t, err, "Unexpected failure in unregistering distro %q.", d.Name())

			list, err := testDistros(ctx)
			require.NoError(t, err, "Failed to read list of registered test distros.")
			require.NotContains(t, list, d, "Found allegedly unregistered distro in list of registered distros.")
		})
	}
}

func TestInstall(t *testing.T) {
	if wsl.MockAvailable() {
		t.Skip("Skipping because the mock does not capture the compexity of this function")
	}

	testCases := map[string]struct {
		distroName       string
		appxPackage      string
		precancelContext bool

		wantErr bool
	}{
		"Success with a real distro name": {distroName: "Ubuntu-22.04", appxPackage: "CanonicalGroupLimited.Ubuntu22.04LTS"},

		"Error with a not real distro name": {distroName: "Ubuntu-00.04", wantErr: true},
		"Error with an empty string":        {distroName: "", wantErr: true},
		"Error with a cancelled context":    {distroName: "Ubuntu-22.04", precancelContext: true, wantErr: true},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			if tc.appxPackage != "" {
				t.Cleanup(func() {
					cmd := fmt.Sprintf("Remove-AppxPackage (Get-AppxPackage -Name %q)", tc.appxPackage)
					//nolint:gosec // Command with variable is acceptable in test code
					_ = exec.Command("powershell.exe", "-NoProfile", "-NoLogo", "-NonInteractive", "-Command", cmd).Run()
				})
			}

			if tc.precancelContext {
				cancel()
			}

			err := wsl.Install(ctx, tc.distroName)
			if tc.wantErr {
				require.Error(t, err, "Install should return an error")
				return
			}

			require.NoError(t, err, "Install should return no error")

			cmd := fmt.Sprintf("(Get-AppxPackage -Name %q).Status", tc.appxPackage)
			//nolint:gosec // Command with variable is acceptable in test code
			out, err := exec.Command("powershell.exe", "-NoProfile", "-NoLogo", "-NonInteractive", "-Command", cmd).Output()
			require.NoError(t, err, "Get-AppxPackage should return no error. Stdout: %s", string(out))

			require.Contains(t, string(out), "Ok", "Appx was not installed")

			err = wsl.Install(ctx, tc.distroName)
			require.NoError(t, err, "Second call to install should return no error")
		})
	}
}
