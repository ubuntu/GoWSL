package wsl_test

import (
	"context"
	"fmt"
	"testing"
	"wsl"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistroString(t *testing.T) {
	d := newDistro(t, jammyRootFs)
	wants := fmt.Sprintf(`distro: %s
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
`, d.Name)

	got := d.String()
	require.Equal(t, wants, got)
}

func TestDistroStringError(t *testing.T) {
	d := wsl.Distro{Name: "ThisDistroIsNotRegistered"}
	wants := fmt.Sprintf(`distro: %s
configuration: failed to get configuration, failed syscall to WslGetDistributionConfiguration
`, d.Name)

	got := d.String()
	require.Equal(t, wants, got)
}

func TestConfiguration(t *testing.T) {

	distro := newDistro(t, jammyRootFs)

	cmd := distro.Command(context.Background(), "useradd testuser")
	err := cmd.Run()
	require.NoError(t, err)

	default_config, err := distro.GetConfiguration()
	require.NoError(t, err)

	tests := map[string]wsl.Configuration{
		// Root user cases
		"Root":                                                  {DefaultUID: 0, InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false},
		"Root DriveMountingEnabled":                             {DefaultUID: 0, InteropEnabled: false, PathAppended: false, DriveMountingEnabled: true},
		"Root PathAppended":                                     {DefaultUID: 0, InteropEnabled: false, PathAppended: true, DriveMountingEnabled: false},
		"Root PathAppended DriveMountingEnabled":                {DefaultUID: 0, InteropEnabled: false, PathAppended: true, DriveMountingEnabled: true},
		"Root InteropEnabled":                                   {DefaultUID: 0, InteropEnabled: true, PathAppended: false, DriveMountingEnabled: false},
		"Root InteropEnabled DriveMountingEnabled":              {DefaultUID: 0, InteropEnabled: true, PathAppended: false, DriveMountingEnabled: true},
		"Root InteropEnabled PathAppended":                      {DefaultUID: 0, InteropEnabled: true, PathAppended: true, DriveMountingEnabled: false},
		"Root InteropEnabled PathAppended DriveMountingEnabled": {DefaultUID: 0, InteropEnabled: true, PathAppended: true, DriveMountingEnabled: true},

		// Default user cases
		"User":                                                  {DefaultUID: 1000, InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false},
		"User DriveMountingEnabled":                             {DefaultUID: 1000, InteropEnabled: false, PathAppended: false, DriveMountingEnabled: true},
		"User PathAppended":                                     {DefaultUID: 1000, InteropEnabled: false, PathAppended: true, DriveMountingEnabled: false},
		"User PathAppended DriveMountingEnabled":                {DefaultUID: 1000, InteropEnabled: false, PathAppended: true, DriveMountingEnabled: true},
		"User InteropEnabled":                                   {DefaultUID: 1000, InteropEnabled: true, PathAppended: false, DriveMountingEnabled: false},
		"User InteropEnabled DriveMountingEnabled":              {DefaultUID: 1000, InteropEnabled: true, PathAppended: false, DriveMountingEnabled: true},
		"User InteropEnabled PathAppended":                      {DefaultUID: 1000, InteropEnabled: true, PathAppended: true, DriveMountingEnabled: false},
		"User InteropEnabled PathAppended DriveMountingEnabled": {DefaultUID: 1000, InteropEnabled: true, PathAppended: true, DriveMountingEnabled: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			defer func() { // Reseting to default state
				distro.DefaultUID(default_config.DefaultUID)
				distro.InteropEnabled(default_config.InteropEnabled)
				distro.PathAppended(default_config.PathAppended)
				distro.DriveMountingEnabled(default_config.DriveMountingEnabled)
			}()

			distro.DefaultUID(tc.DefaultUID)
			require.NoError(t, err)

			distro.InteropEnabled(tc.InteropEnabled)
			require.NoError(t, err)

			distro.PathAppended(tc.PathAppended)
			require.NoError(t, err)

			distro.DriveMountingEnabled(tc.DriveMountingEnabled)
			require.NoError(t, err)

			got, err := distro.GetConfiguration()
			require.NoError(t, err)

			// Config test
			assert.Equal(t, tc.DefaultUID, got.DefaultUID)
			assert.Equal(t, tc.InteropEnabled, got.InteropEnabled)
			assert.Equal(t, tc.PathAppended, got.PathAppended)
			assert.Equal(t, tc.DriveMountingEnabled, got.DriveMountingEnabled)
		})
	}
}
