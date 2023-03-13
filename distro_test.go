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
	ctx := context.Background()
	if wsl.MockAvailable() {
		t.Skip("Skipping test because back-end does not implement it")
		ctx = wsl.WithMock(ctx, mock.New())
	}

	d := newTestDistro(t, ctx, rootFs) // Will terminate

	defer startTestLinuxProcess(t, &d)()

	err := wsl.Shutdown(ctx)
	require.NoError(t, err, "Unexpected error attempting to shut down")

	require.False(t, isTestLinuxProcessAlive(&d), "Process was not killed by shutting down.")
}

func TestTerminate(t *testing.T) {
	ctx := context.Background()
	if wsl.MockAvailable() {
		t.Skip("Skipping test because back-end does not implement it")
		ctx = wsl.WithMock(ctx, mock.New())
	}

	sampleDistro := newTestDistro(t, ctx, rootFs)  // Will terminate
	controlDistro := newTestDistro(t, ctx, rootFs) // Will not terminate, used to assert other distros are unaffected

	defer startTestLinuxProcess(t, &sampleDistro)()
	defer startTestLinuxProcess(t, &controlDistro)()

	err := sampleDistro.Terminate()
	require.NoError(t, err, "Unexpected error attempting to terminate")

	require.False(t, isTestLinuxProcessAlive(&sampleDistro), "Process was not killed by termination.")
	require.True(t, isTestLinuxProcessAlive(&controlDistro), "Process was killed by termination of a different distro.")
}

// startTestLinuxProcess starts a linux process that is easy to grep for.
func startTestLinuxProcess(t *testing.T, d *wsl.Distro) context.CancelFunc {
	t.Helper()

	cmd := "$env:WSL_UTF8=1 ; wsl.exe -d " + d.Name() + " -- bash -ec 'sleep 500 && echo LongIdentifyableStringThatICanGrep'"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	c := exec.CommandContext(ctx, "powershell.exe", "-Command", cmd) //nolint:gosec
	err := c.Start()
	require.NoError(t, err, "Unexpected error launching command")

	// Waiting for process to start
	tk := time.NewTicker(100 * time.Microsecond)
	defer tk.Stop()

	for i := 0; i < 10; i++ {
		<-tk.C
		if !isTestLinuxProcessAlive(d) { // Process not started
			continue
		}
		return cancel
	}
	require.Fail(t, "Command failed to start")
	return cancel
}

// isTestLinuxProcessAlive checks if the process strated by startTestLinuxProcess is still alive.
func isTestLinuxProcessAlive(d *wsl.Distro) bool {
	cmd := "$env:WSL_UTF8=1 ; wsl.exe -d " + d.Name() + " -- bash -ec 'ps aux | grep LongIdentifyableStringThatICanGrep | grep -v grep'"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := exec.CommandContext(ctx, "powershell.exe", "-Command", cmd).CombinedOutput() //nolint:gosec
	return err == nil
}

func TestDefaultDistro(t *testing.T) {
	ctx := context.Background()
	if wsl.MockAvailable() {
		t.Parallel()
		ctx = wsl.WithMock(ctx, mock.New())
	}

	want := newTestDistro(t, ctx, emptyRootFs)

	err := setDefaultDistro(ctx, want.Name())
	require.NoError(t, err, "Setup: could not set the default distro")

	got, err := wsl.DefaultDistro(ctx)
	require.NoError(t, err, "unexpected error getting default distro %q", want.Name())
	require.Equal(t, want, got, "Unexpected mismatch in default distro")
}

func TestDistroSetAsDefault(t *testing.T) {
	ctx := context.Background()
	if wsl.MockAvailable() {
		t.Parallel()
		ctx = wsl.WithMock(ctx, mock.New())
	}

	realDistro := newTestDistro(t, ctx, emptyRootFs)
	fakeDistro := wsl.NewDistro(ctx, "This distro sure does not exist")

	testCases := map[string]struct {
		distro  wsl.Distro
		wantErr bool
	}{
		"set an existing distro as default":                 {distro: realDistro},
		"error when setting non-existent distro as default": {distro: fakeDistro, wantErr: true},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			err := tc.distro.SetAsDefault()

			if tc.wantErr {
				require.Errorf(t, err, "Unexpected success setting non-existent distro %q as default", tc.distro.Name())
				return
			}
			require.NoErrorf(t, err, "Unexpected error setting %q as default", tc.distro.Name())

			got, err := defaultDistro(ctx)
			require.NoError(t, err, "unexpected error getting default distro")
			require.Equal(t, tc.distro.Name(), got)
		})
	}
}

func TestDistroString(t *testing.T) {
	ctx := context.Background()
	if wsl.MockAvailable() {
		t.Parallel()
		ctx = wsl.WithMock(ctx, mock.New())
	}

	realDistro := newTestDistro(t, ctx, rootFs)
	fakeDistro := wsl.NewDistro(ctx, uniqueDistroName(t))
	wrongDistro := wsl.NewDistro(ctx, uniqueDistroName(t)+"_\x00_invalid_name")

	realGUID, err := realDistro.GUID()
	require.NoError(t, err, "could not get the test distro's GUID")

	testCases := map[string]struct {
		distro *wsl.Distro

		want string
	}{
		"registered distro": {distro: &realDistro, want: fmt.Sprintf(`WSL distro %q (%s)`, realDistro.Name(), realGUID)},
		"fake distro":       {distro: &fakeDistro, want: fmt.Sprintf(`WSL distro %q (not registered)`, fakeDistro.Name())},
		"wrong distro":      {distro: &wrongDistro, want: fmt.Sprintf(`WSL distro %q (not registered)`, wrongDistro.Name())},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			d := *tc.distro
			got := d.String()
			require.Equal(t, tc.want, got)
		})
	}
}

// The subtests can be parallel but the main body cannot, since it registers a
// distro, possibly interfering with other tests.
//
//nolint:tparallel
func TestGUID(t *testing.T) {
	// This test validates that the GUID is properly obtained and printed.
	// Note that windows.GUID has a String method printing the expected
	// format "{XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}", but syscall.GUID
	// does not have such method and prints its contents like any other
	// struct.
	ctx := context.Background()
	if wsl.MockAvailable() {
		t.Parallel()
		ctx = wsl.WithMock(ctx, mock.New())
	}

	realDistro := wsl.NewDistro(ctx, uniqueDistroName(t))
	fakeDistro := wsl.NewDistro(ctx, uniqueDistroName(t))
	wrongDistro := wsl.NewDistro(ctx, uniqueDistroName(t)+"\x00invalidcharacter")

	err := realDistro.Register(emptyRootFs)
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
		distro *wsl.Distro

		wantErr bool
	}{
		"real distro":  {distro: &realDistro},
		"fake distro":  {distro: &fakeDistro, wantErr: true},
		"wrong distro": {distro: &wrongDistro, wantErr: true},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			guid, err := tc.distro.GUID()
			if tc.wantErr {
				require.Error(t, err, "Unexpected success obtaining GUID of non-eligible distro")
				return
			}
			require.NoError(t, err, "could not obtain GUID")
			require.NotEqual(t, (uuid.UUID{}), guid, "GUID was not initialized")
			require.Regexpf(t, guidRegex, guid.String(), "GUID does not match pattern")
		})
	}
}

func TestConfigurationSetters(t *testing.T) {
	if wsl.MockAvailable() {
		t.Parallel()
	}

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
		setting testedSetting
		distro  distroType
		wantErr bool
	}{
		// DefaultUID
		"success DefaultUID":             {setting: DefaultUID, distro: DistroRegistered},
		"fail DefaultUID \\0 in name":    {setting: DefaultUID, distro: DistroInvalidName, wantErr: true},
		"fail DefaultUID not registered": {setting: DefaultUID, distro: DistroNotRegistered, wantErr: true},

		// InteropEnabled
		"success InteropEnabled":             {setting: InteropEnabled},
		"fail InteropEnabled \\0 in name":    {setting: InteropEnabled, distro: DistroInvalidName, wantErr: true},
		"fail InteropEnabled not registered": {setting: InteropEnabled, distro: DistroNotRegistered, wantErr: true},

		// PathAppended
		"success PathAppended":             {setting: PathAppend},
		"fail PathAppended \\0 in name":    {setting: PathAppend, distro: DistroInvalidName, wantErr: true},
		"fail PathAppended not registered": {setting: PathAppend, distro: DistroNotRegistered, wantErr: true},

		// DriveMountingEnabled
		"success DriveMountingEnabled":             {setting: DriveMounting},
		"fail DriveMountingEnabled \\0 in name":    {setting: DriveMounting, distro: DistroInvalidName, wantErr: true},
		"fail DriveMountingEnabled not registered": {setting: DriveMounting, distro: DistroNotRegistered, wantErr: true},
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

			ctx := context.Background()
			if wsl.MockAvailable() {
				t.Parallel()
				ctx = wsl.WithMock(ctx, mock.New())
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
				d = newTestDistro(t, ctx, rootFs)
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
				require.Errorf(t, err, "unexpected failure when setting config %s", details[tc.setting].name)
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
	ctx := context.Background()
	if wsl.MockAvailable() {
		t.Parallel()
		ctx = wsl.WithMock(ctx, mock.New())
	}

	d := newTestDistro(t, ctx, rootFs)

	testCases := map[string]struct {
		distroName string
		wantErr    bool
	}{
		"success":                {distroName: d.Name()},
		"distro not registered":  {distroName: "IAmNotRegistered", wantErr: true},
		"null character in name": {distroName: "MyName\x00IsNotValid", wantErr: true},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			d := wsl.NewDistro(ctx, tc.distroName)
			c, err := d.GetConfiguration()

			if tc.wantErr {
				require.Error(t, err, "unexpected success in GetConfiguration")
				return
			}
			require.NoError(t, err, "unexpected failure in GetConfiguration")
			assert.Equal(t, c.Version, uint8(2))
			assert.Equal(t, c.DefaultUID, uint32(0))
			assert.Equal(t, c.InteropEnabled, true)
			assert.Equal(t, c.PathAppended, true)
			assert.Equal(t, c.DriveMountingEnabled, true)

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
	ctx := context.Background()
	if wsl.MockAvailable() {
		t.Parallel()
		ctx = wsl.WithMock(ctx, mock.New())
	}

	realDistro := newTestDistro(t, ctx, rootFs)
	nonRegisteredDistro := wsl.NewDistro(ctx, uniqueDistroName(t))

	type action int
	const (
		none = iota
		install
		command
		terminate
	)

	testCases := map[string]struct {
		distro *wsl.Distro
		action action

		want wsl.State
	}{
		"non-registered distro":     {distro: &nonRegisteredDistro, action: none, want: wsl.NonRegistered},
		"real distro is stopped":    {distro: &realDistro, action: terminate, want: wsl.Stopped},
		"real distro is running":    {distro: &realDistro, action: command, want: wsl.Running},
		"distro is being installed": {distro: nil, action: install, want: wsl.Installing},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			switch tc.action {
			case none:
			case install:
				if wsl.MockAvailable() {
					t.Skip("Skipping because mock registers instantly")
				}
				d := asyncNewTestDistro(t, ctx, emptyRootFs)
				tc.distro = &d
				require.Eventually(t, func() bool {
					r, err := d.IsRegistered()
					return err == nil && r
				}, 3*time.Second, 100*time.Millisecond, "Setup: distro never started installing")
			case command:
				_, err := tc.distro.Command(ctx, "exit 0").Output()
				require.NoError(t, err, "Setup: distro.Command should not return an error")
			case terminate:
				_ = tc.distro.Terminate()
			default:
				require.Failf(t, "Setup: unknown action enum", "Value: %d", tc.action)
			}

			got, err := tc.distro.State()
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
		installDistro(t, ctx, d.Name(), loc, rootFs)
	}()

	t.Cleanup(func() {
		wg.Wait()
		if err := uninstallDistro(d, false); err != nil {
			t.Logf("Cleanup: %v", err)
		}
	})

	return d
}
