package gowsl_test

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	wsl "github.com/ubuntu/gowsl"
	"github.com/ubuntu/gowsl/mock"
)

func TestShutdown(t *testing.T) {
	if wsl.MockAvailable() {
		t.Parallel()
	}

	testCases := map[string]struct {
		mockErr bool

		wantErr bool
	}{
		"Success": {},

		// Mock-induced errors
		"Error because wsl.exe returns an error": {mockErr: true, wantErr: true},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx, modifyMock := setupBackend(t, context.Background())
			d1 := newTestDistro(t, ctx, rootFS)
			d2 := newTestDistro(t, ctx, rootFS)

			if tc.mockErr {
				modifyMock(t, func(m *mock.Backend) {
					m.ShutdownError = true
				})
				defer modifyMock(t, (*mock.Backend).ResetErrors)
			}

			wakeDistroUp(t, d1)
			wakeDistroUp(t, d2)

			err := wsl.Shutdown(ctx)
			if tc.wantErr {
				require.Error(t, err, "Shutdown should have returned an error")
				return
			}
			require.NoError(t, err, "Shutdown should have returned no error")

			requireStatef(t, wsl.Stopped, d1, "All distros should be stopped after Shutdown (d1 wasn't)")
			requireStatef(t, wsl.Stopped, d2, "All distros should be stopped after Shutdown (d2 wasn't)")
		})
	}
}

func TestTerminate(t *testing.T) {
	if wsl.MockAvailable() {
		t.Parallel()
	}

	testCases := map[string]struct {
		mockErr      bool
		dontRegister bool

		wantErr         bool
		wantErrNotExist bool
	}{
		"Success": {},

		// Mock-induced errors
		"Error because the distro is not registered": {dontRegister: true, wantErr: true, wantErrNotExist: true},
		"Error because wsl.exe returns an error":     {mockErr: true, wantErr: true},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx, modifyMock := setupBackend(t, context.Background())

			if tc.mockErr {
				modifyMock(t, func(m *mock.Backend) {
					m.TerminateError = true
				})
				defer modifyMock(t, (*mock.Backend).ResetErrors)
			}

			controlDistro := newTestDistro(t, ctx, rootFS)

			var testDistro wsl.Distro
			if tc.dontRegister {
				testDistro = wsl.NewDistro(ctx, uniqueDistroName(t))
			} else {
				testDistro = newTestDistro(t, ctx, rootFS)
				wakeDistroUp(t, testDistro)
			}
			wakeDistroUp(t, controlDistro)

			err := testDistro.Terminate()
			if tc.wantErr {
				require.Error(t, err, "Terminate should have returned an error")
				if tc.wantErrNotExist {
					require.ErrorIs(t, err, wsl.ErrNotExist, "Terminate error should have been ErrNotExist")
				}
				return
			}
			require.NoError(t, err, "Terminate should have returned no error")

			requireStatef(t, wsl.Stopped, testDistro, "The test distro should be stopped after terminating it")
			requireStatef(t, wsl.Running, controlDistro, "The control distro should be running after terminating the test distro")
		})
	}
}

func TestDefaultDistro(t *testing.T) {
	setupBackend(t, context.Background())

	testCases := map[string]struct {
		registryInaccessible bool
		dontCreateDistro     bool

		overrideDefaultDistroRegistry string

		wantOK  bool
		wantErr bool
	}{
		"Success":                        {wantOK: true},
		"Success with no default distro": {dontCreateDistro: true},

		// Mock-induced errors
		"Error when the registry cannot be accessed":      {registryInaccessible: true, wantErr: true},
		"Error when the registry has an invalid UUID":     {overrideDefaultDistroRegistry: "i-am-not-a-uuid", wantErr: true},
		"Error when the registry has a non-existent UUID": {overrideDefaultDistroRegistry: uuid.UUID{}.String(), wantErr: true},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx, modifyMock := setupBackend(t, context.Background())

			var want string

			var d wsl.Distro
			if !tc.dontCreateDistro {
				d = newTestDistro(t, ctx, emptyRootFS)
				err := setDefaultDistro(ctx, d.Name())
				require.NoError(t, err, "Setup: could not set the default distro")
				want = d.Name()
			}

			if tc.registryInaccessible {
				modifyMock(t, func(m *mock.Backend) {
					m.OpenLxssKeyError = true
				})
			}

			if tc.overrideDefaultDistroRegistry != "" {
				modifyMock(t, func(m *mock.Backend) {
					k, err := m.OpenLxssRegistry(".")
					require.NoError(t, err, "Setup: could not open the lxss registry")
					defer k.Close()

					//nolint:forcetypeassert // We know this is a mock
					k.(*mock.RegistryKey).Data["DefaultDistribution"] = tc.overrideDefaultDistroRegistry
				})
			}

			got, ok, err := wsl.DefaultDistro(ctx)
			if tc.wantErr {
				require.Error(t, err, "expected DefaultDistro to return an error")
				return
			}
			require.NoError(t, err, "unexpected error getting default distro %q", want)

			if !tc.wantOK {
				require.False(t, ok, "DefaultDistro should return OK=false")
				return
			}
			assert.True(t, ok, "DefaultDistro should return OK=true")
			assert.Equal(t, want, got.Name(), "Unexpected mismatch in default distro")
		})
	}
}

func TestDistroSetAsDefault(t *testing.T) {
	setupBackend(t, context.Background())

	testCases := map[string]struct {
		nonRegisteredDistro bool
		wslexeError         bool

		wantErr         bool
		wantErrNotExist bool
	}{
		"Success setting an existing distro as default": {},

		"Error when setting non-existent distro as default": {nonRegisteredDistro: true, wantErr: true, wantErrNotExist: true},

		// Mock-induced errors
		"Error when wsl.exe errors out": {wslexeError: true, wantErr: true},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx, modifyMock := setupBackend(t, context.Background())
			if tc.wslexeError {
				modifyMock(t, func(m *mock.Backend) {
					m.SetAsDefaultError = true
				})
				defer modifyMock(t, (*mock.Backend).ResetErrors)
			}

			var d wsl.Distro
			if tc.nonRegisteredDistro {
				d = wsl.NewDistro(ctx, uniqueDistroName(t))
			} else {
				d = newTestDistro(t, ctx, emptyRootFS)
			}

			err := d.SetAsDefault()
			if tc.wantErr {
				require.Errorf(t, err, "Unexpected success setting non-existent distro %q as default", d.Name())
				if tc.wantErrNotExist {
					require.ErrorIs(t, err, wsl.ErrNotExist, "SetAsDefault should have returned ErrNotExist")
				}
				return
			}
			require.NoErrorf(t, err, "Unexpected error setting %q as default", d.Name())

			got, ok, err := defaultDistro(ctx)
			require.NoError(t, err, "unexpected error getting default distro")
			require.True(t, ok, "DefaultDistro should return OK=true")
			require.Equal(t, d.Name(), got)
		})
	}
}

func TestDistroString(t *testing.T) {
	ctx, _ := setupBackend(t, context.Background())

	realDistro := newTestDistro(t, ctx, rootFS)
	fakeDistro := wsl.NewDistro(ctx, uniqueDistroName(t))
	wrongDistro := wsl.NewDistro(ctx, uniqueDistroName(t)+"_\x00_invalid_name")

	realGUID, err := realDistro.GUID()
	require.NoError(t, err, "could not get the test distro's GUID")

	testCases := map[string]struct {
		distro *wsl.Distro

		want string
	}{
		"Success registering a distro": {distro: &realDistro, want: fmt.Sprintf(`WSL distro %q (%s)`, realDistro.Name(), realGUID)},

		"Error when the distro is not registered":           {distro: &fakeDistro, want: fmt.Sprintf(`WSL distro %q (not registered)`, fakeDistro.Name())},
		"Error when the distro name has invalid characters": {distro: &wrongDistro, want: fmt.Sprintf(`WSL distro %q (not registered)`, wrongDistro.Name())},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			setupBackend(t, context.Background())

			d := *tc.distro
			got := d.String()
			require.Equal(t, tc.want, got)
		})
	}
}

func TestGUID(t *testing.T) {
	// This test validates that the GUID is properly obtained and printed.
	ctx, modifyMock := setupBackend(t, context.Background())

	realDistro := wsl.NewDistro(ctx, uniqueDistroName(t))
	fakeDistro := wsl.NewDistro(ctx, uniqueDistroName(t))
	wrongDistro := wsl.NewDistro(ctx, uniqueDistroName(t)+"\x00invalidcharacter")

	err := realDistro.Register(emptyRootFS)
	require.NoError(t, err, "could not register empty distro")

	//nolint:errcheck // We don't care about cleanup errors
	t.Cleanup(func() { realDistro.Unregister() })

	// We cannot really assert on the GUID without re-implementing the distro.GUID() method,
	// leading to circular logic that would test that our two implementations match rather
	// than their correctness.
	//
	// We can at least check that it adheres to the expected format with a regex
	guidRegex := regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)

	testCases := map[string]struct {
		distro               *wsl.Distro
		registryInaccessible bool

		wantErr         bool
		wantNotExistErr bool
	}{
		"Success with a real distro": {distro: &realDistro},

		"Error with a non-registered distro": {distro: &fakeDistro, wantErr: true, wantNotExistErr: true},
		"Error with an invalid distro name":  {distro: &wrongDistro, wantErr: true},

		// Mock-induced errors
		"Error when the registry is inaccessible": {distro: &realDistro, registryInaccessible: true, wantErr: true},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if tc.registryInaccessible {
				modifyMock(t, func(m *mock.Backend) {
					m.OpenLxssKeyError = true
				})
				defer modifyMock(t, (*mock.Backend).ResetErrors)
			} else {
				// The backend is shared across subtests so tests with broken back-ends must run separately
				t.Parallel()
			}

			guid, err := tc.distro.GUID()
			if tc.wantErr {
				require.Error(t, err, "Unexpected success obtaining GUID of non-eligible distro")
				if tc.wantNotExistErr {
					require.ErrorIs(t, err, wsl.ErrNotExist, "GUID error should have been ErrNotExist")
				}
				return
			}
			require.NoError(t, err, "could not obtain GUID")
			require.NotEqual(t, (uuid.UUID{}), guid, "GUID was not initialized")
			require.Regexpf(t, guidRegex, guid.String(), "GUID does not match pattern")
		})
	}
}

func TestConfigurationSetters(t *testing.T) {
	setupBackend(t, context.Background())

	type testedSetting uint
	const (
		DefaultUID testedSetting = iota
		InteropEnabled
		PathAppend
		DriveMounting
	)

	type distroType uint
	const (
		DistroRegistered distroType = iota
		DistroNotRegistered
		DistroInvalidName
	)

	tests := map[string]struct {
		setting      testedSetting
		distro       distroType
		syscallError bool

		wantErr         bool
		wantErrNotExist bool
	}{
		// DefaultUID
		"Success setting DefaultUID":                        {setting: DefaultUID},
		"Error when setting DefaultUID: \\0 in name":        {setting: DefaultUID, distro: DistroInvalidName, wantErr: true},
		"Error when setting DefaultUID: not registered":     {setting: DefaultUID, distro: DistroNotRegistered, wantErr: true, wantErrNotExist: true},
		"Error when setting DefaultUID: syscall errors out": {setting: DefaultUID, syscallError: true, wantErr: true},

		// InteropEnabled
		"Success setting InteropEnabled":                        {setting: InteropEnabled},
		"Error when setting InteropEnabled: \\0 in name":        {setting: InteropEnabled, distro: DistroInvalidName, wantErr: true},
		"Error when setting InteropEnabled: not registered":     {setting: InteropEnabled, distro: DistroNotRegistered, wantErr: true, wantErrNotExist: true},
		"Error when setting InteropEnabled: syscall errors out": {setting: InteropEnabled, syscallError: true, wantErr: true},

		// PathAppended
		"Success setting PathAppended":                        {setting: PathAppend},
		"Error when setting PathAppended: \\0 in name":        {setting: PathAppend, distro: DistroInvalidName, wantErr: true},
		"Error when setting PathAppended: not registered":     {setting: PathAppend, distro: DistroNotRegistered, wantErr: true, wantErrNotExist: true},
		"Error when setting PathAppended: syscall errors out": {setting: PathAppend, syscallError: true, wantErr: true},

		// DriveMountingEnabled
		"Success setting DriveMountingEnabled":                        {setting: DriveMounting},
		"Error when setting DriveMountingEnabled: \\0 in name":        {setting: DriveMounting, distro: DistroInvalidName, wantErr: true},
		"Error when setting DriveMountingEnabled: not registered":     {setting: DriveMounting, distro: DistroNotRegistered, wantErr: true, wantErrNotExist: true},
		"Error when setting DriveMountingEnabled: syscall errors out": {setting: DriveMounting, syscallError: true, wantErr: true},
	}

	type settingDetails struct {
		name      string // Name of the setting (Used to generate error messages)
		byDefault any    // Default value
		want      any    // Wanted value (will be overridden during test)
	}

	// Overrides the "want" in a settingDetails dict (bypasses the non-addressablity of the struct member)
	setWant := func(d map[testedSetting]settingDetails, setter testedSetting, want any) {
		det := d[setter]
		det.want = want
		d[setter] = det
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// This test has two phases:
			// 1. Changes one of the default settings and asserts that it has changed, and the others have not.
			// 2. It changes this setting back to the default, and asserts that it has changed, and the others have not.

			ctx, modifyMock := setupBackend(t, context.Background())
			if tc.syscallError {
				modifyMock(t, func(m *mock.Backend) {
					m.WslConfigureDistributionError = true
				})
				defer modifyMock(t, (*mock.Backend).ResetErrors)
			}

			// details has info about each of the settings
			details := map[testedSetting]settingDetails{
				DefaultUID:     {name: "DefaultUID", byDefault: uint32(0), want: uint32(0)},
				InteropEnabled: {name: "InteropEnabled", byDefault: true, want: true},
				PathAppend:     {name: "PathAppended", byDefault: true, want: true},
				DriveMounting:  {name: "DriveMountingEnabled", byDefault: true, want: true},
			}

			// errorMsg is a helper map to avoid rewriting the same error message
			errorMsg := map[testedSetting]string{
				DefaultUID:     fmt.Sprintf("config %s changed when it wasn't supposed to", details[DefaultUID].name),
				InteropEnabled: fmt.Sprintf("config %s changed when it wasn't supposed to", details[InteropEnabled].name),
				PathAppend:     fmt.Sprintf("config %s changed when it wasn't supposed to", details[PathAppend].name),
				DriveMounting:  fmt.Sprintf("config %s changed when it wasn't supposed to", details[DriveMounting].name),
			}

			// Choosing the distro
			var d wsl.Distro
			switch tc.distro {
			case DistroRegistered:
				d = newTestDistro(t, ctx, rootFS)
				err := d.Command(context.Background(), "useradd testuser").Run()
				require.NoError(t, err, "unexpectedly failed to add a user to the distro")
			case DistroNotRegistered:
				d = wsl.NewDistro(ctx, uniqueDistroName(t))
			case DistroInvalidName:
				d = wsl.NewDistro(ctx, "Wrong character \x00 in name")
			}

			// Test changing a setting
			var err error
			switch tc.setting {
			case DefaultUID:
				setWant(details, DefaultUID, uint32(1000))
				err = d.DefaultUID(1000)
			case InteropEnabled:
				setWant(details, InteropEnabled, false)
				err = d.InteropEnabled(false)
			case PathAppend:
				setWant(details, PathAppend, false)
				err = d.PathAppended(false)
			case DriveMounting:
				setWant(details, DriveMounting, false)
				err = d.DriveMountingEnabled(false)
			}
			if tc.wantErr {
				require.Errorf(t, err, "unexpected success when setting config %s", details[tc.setting].name)
				if tc.wantErrNotExist {
					require.ErrorIs(t, err, wsl.ErrNotExist, "expected setter to return ErrNotExist")
				}
				return
			}
			require.NoErrorf(t, err, "unexpected success when setting config %s", details[tc.setting].name)

			got, err := d.GetConfiguration()
			require.NoError(t, err, "unexpected failure getting configuration")

			errorMsg[tc.setting] = fmt.Sprintf("config %s did not change to the expected value", details[tc.setting].name)
			require.Equal(t, details[DefaultUID].want, got.DefaultUID, errorMsg[DefaultUID])
			require.Equal(t, details[InteropEnabled].want, got.InteropEnabled, errorMsg[InteropEnabled])
			require.Equal(t, details[PathAppend].want, got.PathAppended, errorMsg[PathAppend])
			require.Equal(t, details[DriveMounting].want, got.DriveMountingEnabled, errorMsg[DriveMounting])

			// Test restore default
			switch tc.setting {
			case DefaultUID:
				err = d.DefaultUID(0)
			case InteropEnabled:
				err = d.InteropEnabled(true)
			case PathAppend:
				err = d.PathAppended(true)
			case DriveMounting:
				err = d.DriveMountingEnabled(true)
			}
			require.NoErrorf(t, err, "unexpected failure when setting %s back to the default", details[tc.setting].name)

			setWant(details, DefaultUID, details[DefaultUID].byDefault)
			setWant(details, InteropEnabled, details[InteropEnabled].byDefault)
			setWant(details, PathAppend, details[PathAppend].byDefault)
			setWant(details, DriveMounting, details[DriveMounting].byDefault)
			got, err = d.GetConfiguration()
			require.NoErrorf(t, err, "unexpected error calling GetConfiguration after reseting default value for %s", details[tc.setting].name)

			errorMsg[tc.setting] = fmt.Sprintf("config %s was not set back to the default", details[tc.setting].name)
			assert.Equal(t, details[DefaultUID].want, got.DefaultUID, errorMsg[DefaultUID])
			assert.Equal(t, details[InteropEnabled].want, got.InteropEnabled, errorMsg[InteropEnabled])
			assert.Equal(t, details[PathAppend].want, got.PathAppended, errorMsg[PathAppend])
			assert.Equal(t, details[DriveMounting].want, got.DriveMountingEnabled, errorMsg[DriveMounting])
		})
	}
}
func TestGetConfiguration(t *testing.T) {
	setupBackend(t, context.Background())

	testCases := map[string]struct {
		distroName   string // Note: distros with custom distro names will not be registered
		syscallError bool

		wantErr         bool
		wantErrNotExist bool
	}{
		"Success": {},

		"Error with non-registered distro":  {distroName: "IAmNotRegistered", wantErr: true, wantErrNotExist: true},
		"Error with null character in name": {distroName: "MyName\x00IsNotValid", wantErr: true},

		// Mock-induced errors
		"Error when the syscall fails": {syscallError: true, wantErr: true},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx, modifyMock := setupBackend(t, context.Background())
			if tc.syscallError {
				modifyMock(t, func(m *mock.Backend) {
					m.WslGetDistributionConfigurationError = true
				})
				defer modifyMock(t, (*mock.Backend).ResetErrors)
			}

			var d wsl.Distro
			if len(tc.distroName) == 0 {
				d = newTestDistro(t, ctx, rootFS)
			} else {
				d = wsl.NewDistro(ctx, uniqueDistroName(t)+tc.distroName)
			}

			c, err := d.GetConfiguration()

			if tc.wantErr {
				require.Error(t, err, "unexpected success in GetConfiguration")
				if tc.wantErrNotExist {
					require.ErrorIs(t, err, wsl.ErrNotExist, "expected GetConfiguration to return ErrNotExist")
				}
				return
			}
			require.NoError(t, err, "unexpected failure in GetConfiguration")
			assert.Equal(t, uint8(2), c.Version)
			assert.Zero(t, c.DefaultUID)
			assert.True(t, c.InteropEnabled)
			assert.True(t, c.PathAppended)
			assert.True(t, c.DriveMountingEnabled)

			defaultEnvs := map[string]string{
				"HOSTTYPE": "x86_64",
				"LANG":     "en_US.UTF-8",
				"PATH":     "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games",
				"TERM":     "xterm-256color",
			}
			assert.Equal(t, c.DefaultEnvironmentVariables, defaultEnvs)
		})
	}
}

func TestDistroState(t *testing.T) {
	ctx, modifyMock := setupBackend(t, context.Background())

	realDistro := newTestDistro(t, ctx, rootFS)
	nonRegisteredDistro := wsl.NewDistro(ctx, uniqueDistroName(t))

	type action int
	const (
		none = iota
		install
		command
		terminate
		injectError
	)

	testCases := map[string]struct {
		distro *wsl.Distro
		action action

		wantErr bool
		want    wsl.State
	}{
		"Non-registered distro":     {distro: &nonRegisteredDistro, action: none, want: wsl.NonRegistered},
		"Real distro is stopped":    {distro: &realDistro, action: terminate, want: wsl.Stopped},
		"Real distro is running":    {distro: &realDistro, action: command, want: wsl.Running},
		"Distro is being installed": {distro: nil, action: install, want: wsl.Installing},

		// Mock-induced errors
		"Error when wsl.exe returns an error": {distro: &realDistro, action: injectError, wantErr: true},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			switch tc.action {
			case none:
			case install:
				if wsl.MockAvailable() {
					t.Skip("Skipping because mock registers instantly")
				}
				d := asyncNewTestDistro(t, ctx, emptyRootFS)
				tc.distro = &d
				require.Eventually(t, func() bool {
					r, err := d.IsRegistered()
					return err == nil && r
				}, 10*time.Second, 100*time.Millisecond, "Setup: distro never started installing")
			case command:
				_, err := tc.distro.Command(ctx, "exit 0").Output()
				require.NoError(t, err, "Setup: distro.Command should not return an error")
			case terminate:
				_ = tc.distro.Terminate()
			case injectError:
				modifyMock(t, func(m *mock.Backend) {
					m.StateError = true
				})
				defer modifyMock(t, (*mock.Backend).ResetErrors)
			default:
				require.Failf(t, "Setup: unknown action enum", "Value: %d", tc.action)
			}

			got, err := tc.distro.State()
			if tc.wantErr {
				require.Error(t, err, "distro.State should return an error")
				return
			}
			require.NoError(t, err, "distro.State should not return an error")

			require.Equal(t, tc.want.String(), got.String(), "distro.State() does not match expected value")
		})
	}
}

//nolint:revive // No, I wont' put the context before the *testing.T.
func asyncNewTestDistro(t *testing.T, ctx context.Context, rootFs string) wsl.Distro {
	t.Helper()

	// This waitgroup ensures we don't try to clean up before installDistro has returned
	var wg sync.WaitGroup
	wg.Add(1)

	d := wsl.NewDistro(ctx, uniqueDistroName(t))
	loc := t.TempDir()

	go func() {
		defer wg.Done()

		defer wslExeGuard(2 * time.Minute)()
		cmd := fmt.Sprintf("$env:WSL_UTF8=1 ;  wsl --import %q %q %q", d.Name(), loc, rootFs)
		//nolint:gosec // Code injection is not a concern in tests.
		out, err := exec.Command("powershell.exe", "-Command", cmd).CombinedOutput()
		if err != nil {
			t.Logf("Setup: failed to register %q: %s", d.Name(), out)
		}
		// We cannot fail here because this is not the main test goroutine
	}()

	t.Cleanup(func() {
		wg.Wait()
		if err := uninstallDistro(d, false); err != nil {
			t.Logf("Cleanup: %v", err)
		}
	})

	return d
}

// wakeDistroUp is a test helper that launches a short command and ensures that the distro is running.
func wakeDistroUp(t *testing.T, d wsl.Distro) {
	t.Helper()

	cmd := d.Command(context.Background(), "exit 0")
	out, err := cmd.Output()
	require.NoErrorf(t, err, "Setup: could not run command to wake distro %q up. Stdout: %s", d, out)

	requireStatef(t, wsl.Running, d, "Setup: distro %q should be running after launching a command", d)
}

// requireStatef ensures that a distro has the expected state.
func requireStatef(t *testing.T, want wsl.State, d wsl.Distro, msg string, args ...any) {
	t.Helper()

	got, err := d.State()
	require.NoErrorf(t, err, "Setup: could not run ascertain test distro state")
	require.Equalf(t, want, got, msg, args)
}
