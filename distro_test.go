package wsl_test

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"
	"wsl"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShutdown(t *testing.T) {
	d := newTestDistro(t, jammyRootFs) // Will terminate

	defer startTestLinuxProcess(t, &d)()

	err := wsl.Shutdown()
	require.NoError(t, err, "Unexpected error attempting to shut down")

	require.False(t, isTestLinuxProcessAlive(&d), "Process was not killed by shutting down.")
}

func TestTerminate(t *testing.T) {
	sampleDistro := newTestDistro(t, jammyRootFs)  // Will terminate
	controlDistro := newTestDistro(t, jammyRootFs) // Will not terminate, used to assert other distros are unaffected

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

	cmd := "$env:WSL_UTF8=1 ; wsl.exe -d " + d.Name + " -- bash -ec 'sleep 500 && echo LongIdentifyableStringThatICanGrep'"
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
	cmd := "$env:WSL_UTF8=1 ; wsl.exe -d " + d.Name + " -- bash -ec 'ps aux | grep LongIdentifyableStringThatICanGrep | grep -v grep'"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := exec.CommandContext(ctx, "powershell.exe", "-Command", cmd).CombinedOutput() //nolint:gosec
	return err == nil
}

func TestDistroString(t *testing.T) {
	realDistro := newTestDistro(t, jammyRootFs)
	fakeDistro := wsl.Distro{Name: UniqueDistroName(t)}
	wrongDistro := wsl.Distro{Name: UniqueDistroName(t) + "_\x00_invalid_name"}

	testCases := map[string]struct {
		distro     *wsl.Distro
		withoutEnv bool
		wants      string
	}{
		"nominal": {
			distro: &realDistro,
			wants: fmt.Sprintf(`distro: %s
configuration:
  - Version: 2
  - DefaultUID: 0
  - InteropEnabled: true
  - PathAppended: true
  - DriveMountingEnabled: true
  - undocumentedWSLVersion: 2
  - DefaultEnvironmentVariables:
    - HOSTTYPE: x86_64
    - LANG: en_US.UTF-8
    - PATH: /usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games
    - TERM: xterm-256color
`, realDistro.Name)},
		"fake distro": {
			distro: &fakeDistro,
			wants: fmt.Sprintf(`distro: %s
configuration: error in GetConfiguration: failed syscall to WslGetDistributionConfiguration
`, fakeDistro.Name)},
		"wrong distro": {
			distro: &wrongDistro,
			wants: fmt.Sprintf(`distro: %s
configuration: error in GetConfiguration: failed to convert %q to UTF16
`, wrongDistro.Name, wrongDistro.Name)},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			d := *tc.distro
			got := d.String()
			require.Equal(t, tc.wants, got)
		})
	}
}

func TestConfigurationSetters(t *testing.T) {
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
		"success DefaultUID":             {setting: DefaultUID},
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
				d = newTestDistro(t, jammyRootFs)
				err := d.Command(context.Background(), "useradd testuser").Run()
				require.NoError(t, err, "unexpectedly failed to add a user to the distro")
			case DistroNotRegistered:
				d = wsl.Distro{Name: UniqueDistroName(t)}
			case DistroInvalidName:
				d = wsl.Distro{Name: "Wrong character \x00 in name"}
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
	d := newTestDistro(t, jammyRootFs)

	testCases := map[string]struct {
		distroName string
		wantErr    bool
	}{
		"success":                {distroName: d.Name},
		"distro not registered":  {distroName: "IAmNotRegistered", wantErr: true},
		"null character in name": {distroName: "MyName\x00IsNotValid", wantErr: true},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			d := wsl.Distro{Name: tc.distroName}
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
