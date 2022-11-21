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

	wsl.Shutdown()

	require.False(t, isTestLinuxProcessAlive(t, &d), "Process was not killed by shutting down.")
}

func TestTerminate(t *testing.T) {
	sampleDistro := newTestDistro(t, jammyRootFs)  // Will terminate
	controlDistro := newTestDistro(t, jammyRootFs) // Will not terminate, used to assert other distros are unaffected

	defer startTestLinuxProcess(t, &sampleDistro)()
	defer startTestLinuxProcess(t, &controlDistro)()

	sampleDistro.Terminate()

	require.False(t, isTestLinuxProcessAlive(t, &sampleDistro), "Process was not killed by termination.")
	require.True(t, isTestLinuxProcessAlive(t, &controlDistro), "Process was killed by termination of a diferent distro.")
}

// startTestLinuxProcess starts a linux process that is easy to grep for.
func startTestLinuxProcess(t *testing.T, d *wsl.Distro) context.CancelFunc {
	cmd := "$env:WSL_UTF8=1 ; wsl.exe -d " + d.Name + " -- bash -ec 'sleep 500 && echo LongIdentifyableStringThatICanGrep'"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	c := exec.CommandContext(ctx, "powershell.exe", "-Command", cmd)
	err := c.Start()
	require.NoError(t, err, "Unexpected error launching command")

	// Waiting for process to start
	tk := time.NewTicker(100 * time.Microsecond)
	defer tk.Stop()

	for i := 0; i < 10; i++ {
		<-tk.C
		if !isTestLinuxProcessAlive(t, d) { // Process not started
			continue
		}
		return cancel
	}
	require.Fail(t, "Command failed to start")
	return cancel
}

// isTestLinuxProcessAlive checks if the process strated by startTestLinuxProcess is still alive.
func isTestLinuxProcessAlive(t *testing.T, d *wsl.Distro) bool {
	cmd := "$env:WSL_UTF8=1 ; wsl.exe -d " + d.Name + " -- bash -ec 'ps aux | grep LongIdentifyableStringThatICanGrep | grep -v grep'"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := exec.CommandContext(ctx, "powershell.exe", "-Command", cmd).CombinedOutput()
	return err == nil
}

func TestDistroString(t *testing.T) {
	realDistro := newTestDistro(t, jammyRootFs)
	fakeDistro := wsl.Distro{Name: UniqueDistroName(t)}

	testCases := map[string]struct {
		fakeDistro bool
		withoutEnv bool
		wants      string
	}{
		"nominal": {wants: fmt.Sprintf(`distro: %s
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
		"wrong distro": {fakeDistro: true, wants: fmt.Sprintf(`distro: %s
configuration: failed to get configuration, failed syscall to WslGetDistributionConfiguration
`, fakeDistro.Name)},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			d := realDistro
			if tc.fakeDistro {
				d = fakeDistro
			}

			got := d.String()
			require.Equal(t, tc.wants, got)
		})
	}
}

func TestConfigurationSetters(t *testing.T) {
	type testedSetting uint
	const (
		DEFAULT_UID testedSetting = iota
		INTEROP_ENABLED
		PATH_APPEND
		DRIVE_MOUNTING
	)

	type distroType uint
	const (
		DISTRO_REGISTERED distroType = iota
		DISTRO_NOT_REGISTERED
		DISTRO_WRONG_NAME
	)

	tests := map[string]struct {
		setting testedSetting
		distro  distroType
		wantErr bool
	}{
		// DefaultUID
		"success DefaultUID":             {setting: DEFAULT_UID},
		"fail DefaultUID \\0 in name":    {setting: DEFAULT_UID, distro: DISTRO_WRONG_NAME, wantErr: true},
		"fail DefaultUID not registered": {setting: DEFAULT_UID, distro: DISTRO_NOT_REGISTERED, wantErr: true},

		// InteropEnabled
		"success InteropEnabled":             {setting: INTEROP_ENABLED},
		"fail InteropEnabled \\0 in name":    {setting: INTEROP_ENABLED, distro: DISTRO_WRONG_NAME, wantErr: true},
		"fail InteropEnabled not registered": {setting: INTEROP_ENABLED, distro: DISTRO_NOT_REGISTERED, wantErr: true},

		// PathAppended
		"success PathAppended":             {setting: PATH_APPEND},
		"fail PathAppended \\0 in name":    {setting: PATH_APPEND, distro: DISTRO_WRONG_NAME, wantErr: true},
		"fail PathAppended not registered": {setting: PATH_APPEND, distro: DISTRO_NOT_REGISTERED, wantErr: true},

		// DriveMountingEnabled
		"success DriveMountingEnabled":             {setting: DRIVE_MOUNTING},
		"fail DriveMountingEnabled \\0 in name":    {setting: DRIVE_MOUNTING, distro: DISTRO_WRONG_NAME, wantErr: true},
		"fail DriveMountingEnabled not registered": {setting: DRIVE_MOUNTING, distro: DISTRO_NOT_REGISTERED, wantErr: true},
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
				DEFAULT_UID:     {name: "DefaultUID", byDefault: uint32(0), want: uint32(0)},
				INTEROP_ENABLED: {name: "InteropEnabled", byDefault: true, want: true},
				PATH_APPEND:     {name: "PathAppended", byDefault: true, want: true},
				DRIVE_MOUNTING:  {name: "DriveMountingEnabled", byDefault: true, want: true},
			}

			// errorMsg is a helper map to avoid rewriting the same error message
			errorMsg := map[testedSetting]string{
				DEFAULT_UID:     fmt.Sprintf("config %s changed when it wasn't supposed to", details[DEFAULT_UID].name),
				INTEROP_ENABLED: fmt.Sprintf("config %s changed when it wasn't supposed to", details[INTEROP_ENABLED].name),
				PATH_APPEND:     fmt.Sprintf("config %s changed when it wasn't supposed to", details[PATH_APPEND].name),
				DRIVE_MOUNTING:  fmt.Sprintf("config %s changed when it wasn't supposed to", details[DRIVE_MOUNTING].name),
			}

			// Choosing the distro
			var d wsl.Distro
			switch tc.distro {
			case DISTRO_REGISTERED:
				d = newTestDistro(t, jammyRootFs)
				err := d.Command(context.Background(), "useradd testuser").Run()
				require.NoError(t, err, "unexpectedly failed to add a user to the distro")
			case DISTRO_NOT_REGISTERED:
				d = wsl.Distro{Name: UniqueDistroName(t)}
			case DISTRO_WRONG_NAME:
				d = wsl.Distro{Name: "Wrong character \x00 in name"}
			}

			// Test changing a setting
			var err error
			switch tc.setting {
			case DEFAULT_UID:
				setWant(details, DEFAULT_UID, uint32(1000))
				err = d.DefaultUID(1000)
			case INTEROP_ENABLED:
				setWant(details, INTEROP_ENABLED, false)
				err = d.InteropEnabled(false)
			case PATH_APPEND:
				setWant(details, PATH_APPEND, false)
				err = d.PathAppended(false)
			case DRIVE_MOUNTING:
				setWant(details, DRIVE_MOUNTING, false)
				err = d.DriveMountingEnabled(false)
			}
			if tc.wantErr {
				require.Errorf(t, err, "unexpected failure when setting config %s", details[tc.setting].name)
				return
			} else {
				require.NoErrorf(t, err, "unexpected success when setting config %s", details[tc.setting].name)
			}

			got, err := d.GetConfiguration()
			require.NoError(t, err, "unexpected failure getting configuration")

			errorMsg[tc.setting] = fmt.Sprintf("config %s did not change to the expected value", details[tc.setting].name)
			require.Equal(t, details[DEFAULT_UID].want, got.DefaultUID, errorMsg[DEFAULT_UID])
			require.Equal(t, details[INTEROP_ENABLED].want, got.InteropEnabled, errorMsg[INTEROP_ENABLED])
			require.Equal(t, details[PATH_APPEND].want, got.PathAppended, errorMsg[PATH_APPEND])
			require.Equal(t, details[DRIVE_MOUNTING].want, got.DriveMountingEnabled, errorMsg[DRIVE_MOUNTING])

			// Test restore default
			switch tc.setting {
			case DEFAULT_UID:
				err = d.DefaultUID(0)
			case INTEROP_ENABLED:
				err = d.InteropEnabled(true)
			case PATH_APPEND:
				err = d.PathAppended(true)
			case DRIVE_MOUNTING:
				err = d.DriveMountingEnabled(true)
			}
			require.NoErrorf(t, err, "unexpected failure when setting %s back to the default", details[tc.setting].name)

			setWant(details, DEFAULT_UID, details[DEFAULT_UID].byDefault)
			setWant(details, INTEROP_ENABLED, details[INTEROP_ENABLED].byDefault)
			setWant(details, PATH_APPEND, details[PATH_APPEND].byDefault)
			setWant(details, DRIVE_MOUNTING, details[DRIVE_MOUNTING].byDefault)
			got, err = d.GetConfiguration()
			require.NoErrorf(t, err, "unexpected error calling GetConfiguration after reseting default value for %s", details[tc.setting].name)

			errorMsg[tc.setting] = fmt.Sprintf("config %s was not set back to the default", details[tc.setting].name)
			assert.Equal(t, details[DEFAULT_UID].want, got.DefaultUID, errorMsg[DEFAULT_UID])
			assert.Equal(t, details[INTEROP_ENABLED].want, got.InteropEnabled, errorMsg[INTEROP_ENABLED])
			assert.Equal(t, details[PATH_APPEND].want, got.PathAppended, errorMsg[PATH_APPEND])
			assert.Equal(t, details[DRIVE_MOUNTING].want, got.DriveMountingEnabled, errorMsg[DRIVE_MOUNTING])
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
