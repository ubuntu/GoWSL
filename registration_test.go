package gowsl_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	wsl "github.com/ubuntu/gowsl"
	wslmock "github.com/ubuntu/gowsl/mock"
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
		t.Run(name, func(t *testing.T) {
			ctx, modifyMock := setupBackend(t, context.Background())
			if tc.syscallError || tc.registryInaccessible {
				modifyMock(t, func(m *wslmock.Backend) {
					m.WslRegisterDistributionError = tc.syscallError
					m.OpenLxssKeyError = tc.registryInaccessible
				})
				defer modifyMock(t, (*wslmock.Backend).ResetErrors)
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
		t.Run(name, func(t *testing.T) {
			if tc.registryInaccessible && !wsl.MockAvailable() {
				t.Skip("This test is only available with the mock enabled")
			}

			ctx, modifyMock := setupBackend(t, context.Background())

			d1 := newTestDistro(t, ctx, emptyRootFS)
			d2 := newTestDistro(t, ctx, emptyRootFS)
			d3 := wsl.NewDistro(ctx, uniqueDistroName(t))

			if tc.registryInaccessible {
				modifyMock(t, func(m *wslmock.Backend) {
					m.OpenLxssKeyError = true
				})
				defer modifyMock(t, (*wslmock.Backend).ResetErrors)
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
		t.Run(name, func(t *testing.T) {
			ctx, modifyMock := setupBackend(t, context.Background())
			if tc.registryInaccessible {
				modifyMock(t, func(m *wslmock.Backend) {
					m.OpenLxssKeyError = true
				})
				defer modifyMock(t, (*wslmock.Backend).ResetErrors)
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

		wantError       bool
		wantErrNotExist bool
	}{
		"Success": {},

		"Error with a non-registered distro": {nonRegistered: true, wantError: true, wantErrNotExist: true},
		"Error with a null char in name":     {nonRegistered: true, distroname: "This Distro \x00 has a null char", wantError: true},

		// Mock-induced errors
		"Error when the registry fails to open": {registryInaccessible: true, wantError: true},
		"Error when the syscall fails":          {syscallError: true, wantError: true},
	}

	for name, tc := range testCases {
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
				modifyMock(t, func(m *wslmock.Backend) {
					m.WslUnregisterDistributionError = tc.syscallError
					m.OpenLxssKeyError = tc.registryInaccessible
				})
				defer modifyMock(t, (*wslmock.Backend).ResetErrors)
			}

			t.Logf("Unregistering %q", d.Name())

			cancel := wslExeGuard(time.Minute)
			err := d.Unregister()
			cancel()

			t.Log("Unregistration completed")

			if tc.wantError {
				require.Errorf(t, err, "Unexpected success in unregistering distro %q.", d.Name())
				if tc.wantErrNotExist {
					require.ErrorIs(t, err, wsl.ErrNotExist, "Unregister should have returned ErrNotExist")
				}
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
		t.Parallel()
	}

	type backend = int
	const (
		either backend = iota
		mockOnly
		realOnly
	)

	testCases := map[string]struct {
		distroName       string
		appxPackage      string
		precancelContext bool

		backend backend
		mockErr bool

		wantErr bool
	}{
		"Success with a real distro name": {distroName: "Ubuntu-22.04", appxPackage: "CanonicalGroupLimited.Ubuntu22.04LTS"},

		// Misuse errors
		"Error with an empty string":     {distroName: "", wantErr: true},
		"Error with a cancelled context": {distroName: "Ubuntu-22.04", precancelContext: true, wantErr: true},

		// Backend-specific errors
		"Error from wsl executable due to a not real distro name": {distroName: "Ubuntu-00.04", backend: realOnly, wantErr: true},
		"Error from wsl executable mock":                          {distroName: "Ubuntu-22.04", backend: mockOnly, mockErr: true, wantErr: true},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			if wsl.MockAvailable() {
				// Mock setup
				if tc.backend == realOnly {
					t.Skip("This test is only available with a real back-end")
				}
				t.Parallel()
				m := wslmock.New()
				m.InstallError = tc.mockErr
				ctx = wsl.WithMock(ctx, m)
			} else {
				// Real back-end setup
				if tc.backend == mockOnly {
					t.Skip("This test is only available with a mock back-end")
				}
				if tc.appxPackage != "" {
					t.Cleanup(func() {
						cmd := fmt.Sprintf("Remove-AppxPackage (Get-AppxPackage -Name %q)", tc.appxPackage)
						//nolint:gosec // Command with variable is acceptable in test code
						_ = exec.Command("powershell.exe", "-NoProfile", "-NoLogo", "-NonInteractive", "-Command", cmd).Run()
					})
				}
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

			// Without mock: check that the AppxPackage has been installed
			if !wsl.MockAvailable() {
				cmd := fmt.Sprintf("(Get-AppxPackage -Name %q).Status", tc.appxPackage)
				//nolint:gosec // Command with variable is acceptable in test code
				out, err := exec.Command("powershell.exe", "-NoProfile", "-NoLogo", "-NonInteractive", "-Command", cmd).Output()
				require.NoError(t, err, "Get-AppxPackage should return no error. Stdout: %s", string(out))

				require.Contains(t, string(out), "Ok", "Appx was not installed")
			}

			err = wsl.Install(ctx, tc.distroName)
			require.NoError(t, err, "Second call to install should return no error")
		})
	}
}

func TestUninstall(t *testing.T) {
	if wsl.MockAvailable() {
		t.Parallel()
	}

	type distroInstallType = int
	const (
		notRegistered distroInstallType = iota
		registeredFromAppx
		imported
	)

	testCases := map[string]struct {
		distroInstallType distroInstallType
		preCancelCtx      bool

		mockCannotRemoveAppx   bool
		mockCannotOpenRegistry bool
		mockOnly               bool

		wantError       bool
		wantErrNotExist bool
	}{
		"Success uninstalling an Appx-installed distro": {distroInstallType: registeredFromAppx},
		"Success uninstalling an imported distro":       {distroInstallType: imported},

		// Usage errors
		"Error uninstalling a non-registered distro":     {distroInstallType: notRegistered, wantError: true, wantErrNotExist: true},
		"Error when the context is cancelled beforehand": {distroInstallType: registeredFromAppx, preCancelCtx: true, wantError: true},

		// Mock-triggered errors
		"Error when the registry cannot be accessed": {mockOnly: true, distroInstallType: registeredFromAppx, mockCannotOpenRegistry: true, wantError: true},
		"Error when appx removal fails":              {mockOnly: true, distroInstallType: registeredFromAppx, mockCannotRemoveAppx: true, wantError: true},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()

			var mock *wslmock.Backend
			if wsl.MockAvailable() {
				t.Parallel()
				mock = wslmock.New()
				ctx = wsl.WithMock(ctx, mock)
			} else if tc.mockOnly {
				t.Skip("This test is only available with the mock enabled")
			}

			var d wsl.Distro
			switch tc.distroInstallType {
			case notRegistered:
				d = wsl.NewDistro(ctx, uniqueDistroName(t))
			case registeredFromAppx:
				if wsl.MockAvailable() {
					d = requireInstallFromAppxMock(t, ctx, mock, "Ubuntu-22.04", "Ubuntu-22.04")
				} else {
					d = requireInstallFromAppxWindows(t, ctx, "Ubuntu-22.04", "ubuntu2204.exe", "Ubuntu-22.04")
					defer requireUninstallAppx(t, ctx, "CanonicalGroupLimited.Ubuntu22.04LTS")
				}
			case imported:
				d = newTestDistro(t, ctx, emptyRootFS)
			}

			// Delayed options to avoid breaking the setup
			if wsl.MockAvailable() {
				mock.RemoveAppxFamilyError = tc.mockCannotRemoveAppx
				mock.OpenLxssKeyError = tc.mockCannotOpenRegistry
			}

			//nolint:errcheck // Nothing we can do about this error
			defer d.Unregister()

			// We cannot cancel the original context because that would break the deferred cleanups
			uninstallCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			if tc.preCancelCtx {
				cancel()
			}

			err := d.Uninstall(uninstallCtx)
			if tc.wantError {
				require.Error(t, err, "Uninstall should return an error")
				if tc.wantErrNotExist {
					require.ErrorIs(t, err, wsl.ErrNotExist, "Uninstall should return ErrNotExist")
				}
				return
			}

			require.NoError(t, err, "Uninstall should return no error")

			reg, err := d.IsRegistered()
			require.NoError(t, err, "IsRegistered should return no error")
			require.False(t, reg, "Uninstall should have unregistered the distro")

			if tc.distroInstallType != registeredFromAppx {
				return
			}

			if wsl.MockAvailable() {
				return
			}

			installed, err := appxIsInstalled(ctx, "Ubuntu-22.04")
			require.NoError(t, err, "appxIsInstalled should return no error")
			require.False(t, installed, "Appx should have been uninstalled")
		})
	}
}

func TestImport(t *testing.T) {
	const (
		isOK int = iota
		isBad
		notExist
	)

	testCases := map[string]struct {
		destinationDir      int
		sourceRootfs        int
		breakWslExe         bool
		distroAlreadyExists bool

		wantErr           bool
		wantNotRegistered bool
	}{
		"Success": {},
		"Success when destination directory does not exist": {destinationDir: notExist},

		"Error when the destination directory cannot be created": {destinationDir: isBad, wantErr: true, wantNotRegistered: true},
		"Error when the source root FS does not exist":           {sourceRootfs: notExist, wantErr: true, wantNotRegistered: true},
		"Error when the source root FS is a directory":           {sourceRootfs: isBad, wantErr: true, wantNotRegistered: true},
		"Error when wsl.exe returns error":                       {breakWslExe: true, wantErr: true, wantNotRegistered: true},
		"Error when the distro already exists":                   {distroAlreadyExists: true, wantErr: true},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			if wsl.MockAvailable() {
				t.Parallel()
				ctx = wsl.WithMock(ctx, wslmock.New())
			}

			src := t.TempDir()
			dst := t.TempDir()

			switch tc.destinationDir {
			case isOK:
				dst = filepath.Join(dst, "destinationDir")
			case isBad:
				dst = filepath.Join(dst, "destinationFile")
				err := os.WriteFile(dst, []byte{}, 0600)
				require.NoError(t, err, "Setup: could not create destination file")
			case notExist:
			default:
				panic("Unrecognized value in destinationDir enum")
			}

			var tarball string
			switch tc.sourceRootfs {
			case isOK:
				var contents []byte
				if tc.breakWslExe {
					// This will also break the real WSL because it's not a
					// valid tarball. Note that empty files ARE valid tarballs.
					contents = []byte("MOCK_ERROR")
				}
				tarball = filepath.Join(src, "rootfs.tar.gz")
				err := os.WriteFile(tarball, contents, 0600)
				require.NoError(t, err, "Setup: could not writer fake tarball")
			case notExist:
				tarball = filepath.Join(src, "idontexist")
			case isBad:
				// It's bad because it is a directory
				tarball = src
			default:
				panic("Unrecognized value in sourceRootfs enum")
			}

			distroName := uniqueDistroName(t)
			if tc.distroAlreadyExists {
				distroName = newTestDistro(t, ctx, tarball).Name()
			}
			t.Cleanup(func() {
				err := uninstallDistro(wsl.NewDistro(ctx, distroName), false)
				if err != nil {
					t.Logf("Cleanup: %v", err)
				}
			})

			cancel := wslExeGuard(time.Minute)
			d, err := wsl.Import(ctx, distroName, tarball, dst)
			cancel()
			if tc.wantErr {
				require.Error(t, err, "Import should return error")
			} else {
				require.NoError(t, err, "Import should not return an error")
				require.Equal(t, distroName, d.Name(), "Distro should have the name that it was imported with")
			}

			distros, err := registeredDistros(ctx)
			require.NoError(t, err, "Could not fetch registered distros")

			found := slices.ContainsFunc(distros, func(d wsl.Distro) bool {
				return d.Name() == distroName
			})
			if tc.wantNotRegistered {
				require.False(t, found, "Distro should not have been registered")
				return
			}
			require.True(t, found, "Distro should have been registered")
		})
	}
}

//nolint:revive // No, I won't put the context before the *testing.T.
func requireInstallFromAppxWindows(t *testing.T, ctx context.Context, appxName string, launcherName string, distroName string) (d wsl.Distro) {
	t.Helper()

	if wsl.MockAvailable() {
		panic("This function should be called with the real back-end only")
	}

	d = wsl.NewDistro(ctx, distroName)
	_ = d.Unregister()

	cmd := exec.CommandContext(ctx,
		"wsl.exe",
		"--install",
		appxName,
		"--no-launch",
	)
	out, err := cmd.Output()
	require.NoErrorf(t, err, "could not install: %v. Stdout: %s", err, out)

	defer wslExeGuard(time.Minute)()

	//nolint:gosec // It's fine for tests
	// Need to use powershell in order to find the launcher executable
	cmd = exec.CommandContext(ctx,
		"powershell.exe",
		"-NonInteractive",
		"-NoProfile",
		"-NoLogo",
		"-Command",
		fmt.Sprintf("& %q install --root --ui=none", launcherName),
	)
	out, err = cmd.Output()
	require.NoErrorf(t, err, "could not register: %v. Stdout: %s", err, out)

	return d
}

//nolint:revive // No, I won't put the context before the *testing.T.
func requireUninstallAppx(t *testing.T, ctx context.Context, appxName string) {
	t.Helper()

	if wsl.MockAvailable() {
		panic("This function should be called with the real back-end only")
	}

	//nolint:gosec // It's fine for tests
	cmd := exec.CommandContext(ctx,
		"powershell.exe",
		"-NonInteractive",
		"-NoProfile",
		"-NoLogo",
		"-Command",
		fmt.Sprintf("Get-AppxPackage %q | Remove-AppxPackage", appxName),
	)
	out, err := cmd.Output()
	require.NoError(t, err, "could not clean up appx package. Stdout: %s", out)
}

//nolint:revive // No, I won't put the context before the *testing.T.
func requireInstallFromAppxMock(t *testing.T, ctx context.Context, m *wslmock.Backend, appxName string, distroName string) (d wsl.Distro) {
	t.Helper()

	d = wsl.NewDistro(ctx, distroName)
	err := d.Register(emptyRootFS)
	require.NoError(t, err, "Setup: could not register")

	guid, err := d.GUID()
	require.NoError(t, err, "Setup: could not get GUID to access distro registry")

	k, err := m.OpenLxssRegistry(fmt.Sprintf("{%s}", guid))
	require.NoError(t, err, "Setup: could not access distro registry")
	defer k.Close()

	//nolint:forcetypeassert // We know it's this type of key because we got it from the mock registry.
	k.(*wslmock.RegistryKey).Data["PackageFamilyName"] = appxName
	return d
}

// appxIsInstalled returns true if an AppxPackage is currently installed.
func appxIsInstalled(ctx context.Context, appxName string) (bool, error) {
	if wsl.MockAvailable() {
		panic("This function is only available for real back-ends")
	}

	//nolint:gosec // Variable in test command is fine
	cmd := exec.CommandContext(
		ctx,
		"powershell.exe",
		"-NonInteractive",
		"-NoProfile",
		"-NoLogo",
		"-Command",
		fmt.Sprintf("(Get-AppxPackage -Name %q).Status", appxName),
	)

	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("could not run Get-AppxPackage: %v. Stdout: %s", err, out)
	}

	if strings.Contains(string(out), "Ok") {
		return true, nil
	}

	return false, nil
}
